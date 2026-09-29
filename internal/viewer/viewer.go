package viewer

import (
	"fmt"
	"io"
	"strconv"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// Viewer is the full-screen F3 window: the title row, the file and the
// viewer's key bar.
type Viewer struct {
	layout
	Title string
	// Push opens a window over the viewer; Close is called when it closes.
	Push  func(ui.View)
	Close func()

	top   int64 // where the first row on screen starts
	col   int   // first column shown when lines are not wrapped
	rows  int   // rows on screen, set by Draw
	next  int64 // where the row after the screen starts, set by Draw
	atEnd bool  // the end of the file is on screen, set by Draw

	pattern         string // last search
	match, matchEnd int64  // the found text, highlighted; empty for none
}

func New(title string, r io.ReaderAt, size int64) *Viewer {
	return &Viewer{Title: title, layout: layout{r: r, size: size, width: 80}, rows: 1}
}

func (v *Viewer) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyF10, tcell.KeyF3:
		if v.Close != nil {
			v.Close()
		}
		return true
	case tcell.KeyUp:
		v.top = v.prevRow(v.top)
	case tcell.KeyDown:
		v.down()
	case tcell.KeyPgUp:
		for range v.rows {
			v.top = v.prevRow(v.top)
		}
	case tcell.KeyPgDn:
		if !v.atEnd {
			v.top = v.next
		}
	case tcell.KeyHome:
		v.top, v.col = 0, 0
	case tcell.KeyEnd:
		v.end()
	case tcell.KeyLeft:
		v.col = max(v.col-1, 0)
	case tcell.KeyRight:
		if !v.wrap && !v.hex {
			v.col++
		}
	case tcell.KeyF2:
		if !v.hex {
			v.wrap = !v.wrap
			v.col = 0
			v.top = v.rowStart(v.top)
		}
	case tcell.KeyF4:
		v.hex = !v.hex
		v.col = 0
		v.top = v.rowStart(v.top)
	}
	return false
}

func (v *Viewer) down() {
	if !v.atEnd {
		v.top = v.nextRow(v.top)
		v.atEnd = v.next >= v.size // until the next Draw, don't run past the end
	}
}

// end shows the last screen of the file.
func (v *Viewer) end() {
	v.col = 0
	if v.size == 0 {
		v.top = 0
		return
	}
	v.top = v.rowStart(v.size - 1)
	for range v.rows - 1 {
		v.top = v.prevRow(v.top)
	}
}

func (v *Viewer) keyLabels() [10]string {
	wrap, hex := "Wrap", "Hex"
	if v.wrap {
		wrap = "Unwrap"
	}
	if v.hex {
		wrap, hex = "", "Text"
	}
	return [10]string{"", wrap, "", hex, "", "", "Search", "", "", "Quit"}
}

func (v *Viewer) Draw(c term.Canvas, w, h int) {
	v.width = w
	v.rows = max(h-2, 1)
	rows, next := v.rowsFrom(v.top, v.rows)
	v.next, v.atEnd = next, next >= v.size

	pct := int64(100)
	if v.size > 0 {
		pct = next * 100 / v.size
	}
	info := fmt.Sprintf(" %s bytes  %3d%% ", strconv.FormatInt(v.size, 10), pct)
	c.HLine(0, 0, w, ' ', term.CursorStyle)
	iw := term.Width(info)
	c.Text(0, 0, term.Fit(v.Title, max(w-iw, 0)), w-iw, term.CursorStyle)
	c.Text(max(w-iw, 0), 0, info, iw, term.CursorStyle)

	c.Fill(0, 1, w, v.rows, ' ', term.PanelStyle)
	for i, r := range rows {
		if v.hex {
			v.drawHex(c, 1+i, r.start)
		} else {
			v.drawText(c, 1+i, w, r)
		}
	}
	ui.DrawKeyBar(c, h-1, w, v.keyLabels())
}

func (v *Viewer) style(off int64) tcell.Style {
	if off >= v.match && off < v.matchEnd {
		return term.CursorStyle
	}
	return term.PanelStyle
}

func (v *Viewer) drawText(c term.Canvas, y, w int, r row) {
	ls := r.start
	if len(r.cells) > 0 {
		ls -= int64(r.cells[0].off)
	}
	x, skip := 0, v.col
	for _, cl := range r.cells {
		if skip > 0 {
			skip -= cl.w
			continue
		}
		if x+cl.w > w {
			return
		}
		c.Put(x, y, cl.r, v.style(ls+int64(cl.off)))
		x += cl.w
	}
}

// drawHex draws "00000010  xx xx … xx  xx … xx  ascii".
func (v *Viewer) drawHex(c term.Canvas, y int, start int64) {
	b := v.read(start, hexWidth)
	c.Text(0, y, fmt.Sprintf("%08x", start), 16, term.PanelStyle)
	for i, x := range b {
		off := start + int64(i)
		hx := 10 + 3*i
		if i >= 8 {
			hx++
		}
		c.Text(hx, y, fmt.Sprintf("%02x", x), 2, v.style(off))
		a := '.'
		if x >= 0x20 && x < 0x7f {
			a = rune(x)
		}
		c.Put(10+3*hexWidth+2+i, y, a, v.style(off))
	}
}
