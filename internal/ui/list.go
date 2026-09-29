package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// List is a window with lines to pick one from, used for the command
// history. It looks like a dialog; the cursor line is black on cyan.
type List struct {
	Title string
	Items []string
	Cur   int // line under the cursor
	Top   int // first visible line
	rows  int // visible lines, set by Draw
	// Done is called with the picked line, or -1 on Esc or in an empty list.
	Done func(i int)
}

func (l *List) move(d int) {
	l.Cur = max(min(l.Cur+d, len(l.Items)-1), 0)
}

func (l *List) HandleKey(ev *tcell.EventKey) bool {
	page := max(l.rows, 1)
	switch ev.Key() {
	case tcell.KeyEscape:
		l.finish(-1)
		return true
	case tcell.KeyEnter:
		if len(l.Items) == 0 {
			l.finish(-1)
		} else {
			l.finish(l.Cur)
		}
		return true
	case tcell.KeyUp:
		l.move(-1)
	case tcell.KeyDown:
		l.move(1)
	case tcell.KeyPgUp:
		l.move(-page)
	case tcell.KeyPgDn:
		l.move(page)
	case tcell.KeyHome:
		l.move(-len(l.Items))
	case tcell.KeyEnd:
		l.move(len(l.Items))
	}
	return false
}

func (l *List) finish(i int) {
	if l.Done != nil {
		l.Done(i)
	}
}

// Draw uses the dialog layout, like TextView, and scrolls to the cursor.
func (l *List) Draw(c term.Canvas, w, h int) {
	cw := max(term.Width(l.Title)+2, 20)
	for _, s := range l.Items {
		cw = max(cw, term.Width(s))
	}
	cw = min(cw, w-10)
	l.rows = max(min(len(l.Items), h-6), 1)
	l.move(0)
	if l.Cur < l.Top {
		l.Top = l.Cur
	}
	if l.Cur >= l.Top+l.rows {
		l.Top = l.Cur - l.rows + 1
	}
	ww, wh := cw+8, l.rows+4
	x, y := max((w-ww)/2, 0), max((h-wh)/2, 0)
	window(c, x, y, ww, wh, term.DialogStyle)
	c.Box(x+2, y+1, ww-4, wh-2, term.DialogFrameStyle)
	drawTitle(c, x, y+1, ww, l.Title, term.DialogFrameStyle)
	for i := 0; i < l.rows && l.Top+i < len(l.Items); i++ {
		st := term.DialogStyle
		if l.Top+i == l.Cur {
			st = term.CursorStyle
		}
		c.Text(x+4, y+2+i, term.Fit(l.Items[l.Top+i], cw), cw, st)
	}
}
