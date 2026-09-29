// Package viewer is the F3 file viewer: text with or without wrapping, hex
// and search. The file is never read whole: every screen reads only what it
// shows, so big files open at once.
package viewer

import (
	"bytes"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

const (
	// maxLine is the longest line shown as one: a longer line is cut at
	// offsets that are multiples of maxLine, so its pieces can be found
	// from anywhere by looking back at most 2*maxLine bytes.
	maxLine  = 4096
	tabWidth = 8
	hexWidth = 16 // bytes in a hex row
)

// layout splits the file into screen rows. A row is a line in text mode,
// a piece of a line that fits the width in wrap mode, and 16 bytes in hex
// mode. Rows are found by their byte offsets.
type layout struct {
	r     io.ReaderAt
	size  int64
	wrap  bool
	hex   bool
	width int // columns for text
}

// read returns up to n bytes at off; a read error gives what was read.
func (l *layout) read(off int64, n int) []byte {
	n = int(min(int64(n), l.size-off))
	if n <= 0 {
		return nil
	}
	b := make([]byte, n)
	k, _ := l.r.ReadAt(b, off)
	return b[:k]
}

// readLine returns the line (or piece of a long line) starting at start,
// without its "\n" and a "\r" before it, and where the next one starts.
func (l *layout) readLine(start int64) (line []byte, next int64) {
	b := l.read(start, maxLine+1)
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		return bytes.TrimSuffix(b[:i], []byte{'\r'}), start + int64(i) + 1
	}
	if len(b) > maxLine {
		end := start - start%maxLine + maxLine
		return b[:end-start], end
	}
	return b, start + int64(len(b))
}

// lineStart returns where the line (or piece) holding off starts.
func (l *layout) lineStart(off int64) int64 {
	from := max(off-2*maxLine, 0)
	var s int64
	if i := bytes.LastIndexByte(l.read(from, int(off-from)), '\n'); i >= 0 {
		s = from + int64(i) + 1
	} else if from > 0 {
		// A line longer than 2*maxLine: this multiple of maxLine starts a piece.
		s = off - off%maxLine - maxLine
	}
	for {
		_, next := l.readLine(s)
		if next > off || next <= s {
			return s
		}
		s = next
	}
}

// cell is one screen column of a text line.
type cell struct {
	r   rune
	w   int // columns: 1 or 2
	off int // byte offset in the line
}

// cells lays a line out in columns: tabs become spaces up to the next stop
// of 8, bad UTF-8 and control characters become '·', zero-width runes are
// dropped.
func cells(line []byte) []cell {
	var cs []cell
	col := 0
	for i := 0; i < len(line); {
		r, n := utf8.DecodeRune(line[i:])
		switch {
		case r == '\t':
			for w := tabWidth - col%tabWidth; w > 0; w-- {
				cs = append(cs, cell{' ', 1, i})
				col++
			}
		case r == utf8.RuneError && n == 1, unicode.IsControl(r):
			cs = append(cs, cell{'·', 1, i})
			col++
		default:
			if w := runewidth.RuneWidth(r); w > 0 {
				cs = append(cs, cell{r, w, i})
				col += w
			}
		}
		i += n
	}
	return cs
}

// wrapCuts returns the indexes in cs where rows of at most w columns start;
// the first is 0. A tab cut at the end of a row is not continued.
func wrapCuts(cs []cell, w int) []int {
	cuts := []int{0}
	col := 0
	for i, c := range cs {
		if col+c.w > w && col > 0 {
			if cs[i-1].off == c.off {
				continue // the rest of a tab
			}
			cuts = append(cuts, i)
			col = 0
		}
		col += c.w
	}
	return cuts
}

// row is a screen row: where it starts and, in text mode, its cells.
type row struct {
	start int64
	cells []cell
}

// pieces splits the line starting at ls into rows.
func (l *layout) pieces(ls int64, line []byte) []row {
	cs := cells(line)
	if !l.wrap {
		return []row{{ls, cs}}
	}
	cuts := wrapCuts(cs, max(l.width, 1))
	rows := make([]row, len(cuts))
	for k, c := range cuts {
		end := len(cs)
		if k+1 < len(cuts) {
			end = cuts[k+1]
		}
		start := ls
		if c < len(cs) {
			start += int64(cs[c].off)
		}
		rows[k] = row{start, cs[c:end]}
	}
	return rows
}

// rowsFrom returns up to n rows starting at the row start top, and where
// the row after them starts (the file size at the end).
func (l *layout) rowsFrom(top int64, n int) (rows []row, next int64) {
	if l.hex {
		for s := top; s < l.size; s += hexWidth {
			if len(rows) == n {
				return rows, s
			}
			rows = append(rows, row{start: s})
		}
		return rows, l.size
	}
	for ls := l.lineStart(top); ls < l.size; {
		line, nx := l.readLine(ls)
		for _, r := range l.pieces(ls, line) {
			if r.start < top {
				continue
			}
			if len(rows) == n {
				return rows, r.start
			}
			rows = append(rows, r)
		}
		ls = nx
	}
	return rows, l.size
}

// rowStart returns the start of the row holding off.
func (l *layout) rowStart(off int64) int64 {
	if l.hex {
		return off - off%hexWidth
	}
	ls := l.lineStart(off)
	line, _ := l.readLine(ls)
	s := ls
	for _, r := range l.pieces(ls, line) {
		if r.start <= off {
			s = r.start
		}
	}
	return s
}

// nextRow returns the start of the row after the one starting at start.
func (l *layout) nextRow(start int64) int64 {
	_, next := l.rowsFrom(start, 1)
	return next
}

// prevRow returns the start of the row before the one starting at start.
func (l *layout) prevRow(start int64) int64 {
	if start <= 0 {
		return 0
	}
	return l.rowStart(start - 1)
}
