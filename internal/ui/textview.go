package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// TextView is a scrollable read-only window, used for help.
type TextView struct {
	Title string
	Lines []string
	Top   int // first visible line
	rows  int // visible lines, set by Draw
}

func (v *TextView) scroll(d int) {
	v.Top = max(min(v.Top+d, len(v.Lines)-v.rows), 0)
}

func (v *TextView) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter, tcell.KeyF1, tcell.KeyF10:
		return true
	case tcell.KeyUp:
		v.scroll(-1)
	case tcell.KeyDown:
		v.scroll(1)
	case tcell.KeyPgUp:
		v.scroll(-v.rows)
	case tcell.KeyPgDn:
		v.scroll(v.rows)
	case tcell.KeyHome:
		v.scroll(-len(v.Lines))
	case tcell.KeyEnd:
		v.scroll(len(v.Lines))
	}
	return false
}

// Draw uses the dialog layout: gray margin, double frame, one space padding.
func (v *TextView) Draw(c term.Canvas, w, h int) {
	cw := term.Width(v.Title) + 2
	for _, l := range v.Lines {
		cw = max(cw, term.Width(l))
	}
	cw = min(cw, w-10)
	v.rows = max(min(len(v.Lines), h-6), 1)
	v.scroll(0)
	ww, wh := cw+8, v.rows+4
	x, y := max((w-ww)/2, 0), max((h-wh)/2, 0)
	window(c, x, y, ww, wh, term.DialogStyle)
	c.Box(x+2, y+1, ww-4, wh-2, term.DialogFrameStyle)
	drawTitle(c, x, y+1, ww, v.Title, term.DialogFrameStyle)
	for i := 0; i < v.rows && v.Top+i < len(v.Lines); i++ {
		c.Text(x+4, y+2+i, v.Lines[v.Top+i], cw, term.DialogStyle)
	}
}
