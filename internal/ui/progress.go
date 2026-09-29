package ui

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

const progressWidth = 50

// Progress is the window of a running file operation: the current file,
// progress bars and a Cancel button. Keys never close it; its owner removes
// it when the operation stops.
type Progress struct {
	Title  string
	File   string
	Bars   []float64 // each 0…1, one row each
	Hidden bool      // not shown yet: drawn as nothing, keys ignored
	Cancel func()

	cancelArea area // set by Draw
}

// HandleMouse presses Cancel when it is clicked.
func (p *Progress) HandleMouse(m Mouse) bool {
	if !p.Hidden && m.Action == Click && p.cancelArea.has(m.X, m.Y) {
		p.Cancel()
	}
	return false
}

func (p *Progress) HandleKey(ev *tcell.EventKey) bool {
	if p.Hidden {
		return false
	}
	switch {
	case ev.Key() == tcell.KeyEscape, ev.Key() == tcell.KeyEnter,
		ev.Key() == tcell.KeyRune && unicode.ToLower(ev.Rune()) == 'c':
		p.Cancel()
	}
	return false
}

// Draw uses the dialog layout: gray margin, double frame, one space of
// padding, then the file name, the bars and the Cancel button.
func (p *Progress) Draw(c term.Canvas, w, h int) {
	if p.Hidden {
		return
	}
	body := term.DialogStyle
	cw := min(progressWidth, w-10)
	ww, wh := cw+8, len(p.Bars)+6
	x, y := max((w-ww)/2, 0), max((h-wh)/2, 0)
	window(c, x, y, ww, wh, body)
	c.Box(x+2, y+1, ww-4, wh-2, term.DialogFrameStyle)
	drawTitle(c, x, y+1, ww, p.Title, term.DialogFrameStyle)
	c.Text(x+4, y+2, term.Fit(p.File, cw), cw, body)
	for i, f := range p.Bars {
		n := int(max(min(f, 1), 0)*float64(cw) + 0.5)
		c.Text(x+4, y+3+i, strings.Repeat("█", n)+strings.Repeat("░", cw-n), cw, body)
	}
	b := " Cancel "
	p.cancelArea = area{x + 4 + (cw-len(b))/2, y + 3 + len(p.Bars), len(b), 1}
	c.Text(p.cancelArea.x, p.cancelArea.y, b, len(b), term.ButtonFocusStyle)
}
