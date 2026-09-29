package viewer

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ui"
)

const searchChunk = 64 << 10 // bytes read at a time while searching

// fold lowercases b rune by rune, except runes whose lowercase has another
// length, so that offsets in the result are offsets in b.
func fold(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if lr := unicode.ToLower(r); r != utf8.RuneError && utf8.RuneLen(lr) == n {
			out = utf8.AppendRune(out, lr)
		} else {
			out = append(out, b[i:i+n]...)
		}
		i += n
	}
	return out
}

// find returns the offset of the first match of pat at or after from,
// ignoring case, or -1.
func (l *layout) find(pat string, from int64) int64 {
	p := fold([]byte(pat))
	if len(p) == 0 {
		return -1
	}
	step := int64(max(searchChunk-len(p)+1, 1))
	for off := from; off < l.size; off += step {
		b := l.read(off, searchChunk)
		if i := bytes.Index(fold(b), p); i >= 0 {
			return off + int64(i)
		}
		if off+int64(len(b)) >= l.size {
			break
		}
	}
	return -1
}

// handleSearchKey handles F7 and the repeat keys; it reports whether ev
// was one of them.
func (v *Viewer) handleSearchKey(ev *tcell.EventKey) bool {
	switch {
	case ev.Key() == tcell.KeyF7 && ev.Modifiers()&tcell.ModShift == 0:
		v.askSearch()
	case ev.Key() == tcell.KeyF7, ev.Key() == tcell.KeyF19, // Shift-F7 in terminals that send it as F19
		ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 && (ev.Rune() == 'n' || ev.Rune() == 'т'): // 'т' is the 'n' key on the Russian layout
		v.searchAgain()
	default:
		return false
	}
	return true
}

func (v *Viewer) askSearch() {
	v.push(&ui.Dialog{
		Title:   "Search",
		Lines:   []string{"Search for"},
		Input:   ui.NewInput(v.pattern),
		Buttons: []string{"Search", "Cancel"},
		Done: func(b int, text string) {
			if b == 0 && text != "" {
				v.pattern = text
				v.search(v.top)
			}
		},
	})
}

func (v *Viewer) searchAgain() {
	switch {
	case v.pattern == "":
		v.askSearch()
	case v.matchEnd > v.match:
		v.search(v.match + 1)
	default:
		v.search(v.top)
	}
}

// search finds the pattern from off on and shows it in the first row.
func (v *Viewer) search(from int64) {
	off := v.find(v.pattern, from)
	if off < 0 {
		v.push(&ui.Dialog{Title: "Search", Lines: []string{"String not found"}, Buttons: []string{"OK"}})
		return
	}
	v.match, v.matchEnd = off, off+int64(len(v.pattern))
	v.top = v.rowStart(off)
	if v.wrap || v.hex {
		return
	}
	line, _ := v.readLine(v.top)
	col := 0
	for _, c := range cells(line) {
		if v.top+int64(c.off) >= off {
			break
		}
		col += c.w
	}
	if col < v.col || col >= v.col+v.width {
		v.col = max(col-v.width/2, 0)
	}
}

func (v *Viewer) push(w ui.View) {
	if v.Push != nil {
		v.Push(w)
	}
}
