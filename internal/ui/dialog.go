package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

const inputMinWidth = 40

// Dialog is an NC dialog: a gray window with a double frame, centered text
// lines, an optional text field and a row of buttons.
type Dialog struct {
	Title   string
	Lines   []string
	Input   *Input // nil for no text field
	Buttons []string
	Focus   int  // focused button
	Danger  bool // white on red, for errors
	// Over gives the columns (x, width) to center the dialog in for a screen
	// w columns wide; nil centers it on the screen.
	Over func(w int) (x, width int)
	// Done is called with the pressed button, or -1 on Esc, and the text
	// of the field.
	Done func(button int, text string)

	buttonAreas []area // set by Draw
}

// HandleMouse presses a clicked button.
func (d *Dialog) HandleMouse(m Mouse) bool {
	if m.Action != Click && m.Action != DoubleClick {
		return false
	}
	for i, r := range d.buttonAreas {
		if r.has(m.X, m.Y) {
			return d.finish(i)
		}
	}
	return false
}

func (d *Dialog) HandleKey(ev *tcell.EventKey) bool {
	n := len(d.Buttons)
	switch ev.Key() {
	case tcell.KeyEscape:
		return d.finish(-1)
	case tcell.KeyEnter:
		return d.finish(d.Focus)
	case tcell.KeyLeft, tcell.KeyBacktab:
		d.Focus = (d.Focus + n - 1) % n
	case tcell.KeyRight, tcell.KeyTab:
		d.Focus = (d.Focus + 1) % n
	default:
		if d.Input != nil {
			d.Input.HandleKey(ev)
			return false
		}
		if ev.Key() == tcell.KeyRune {
			r := strings.ToLower(string(ev.Rune()))
			for i, b := range d.Buttons {
				if strings.HasPrefix(strings.ToLower(b), r) {
					return d.finish(i)
				}
			}
		}
	}
	return false
}

func (d *Dialog) finish(button int) bool {
	if d.Done != nil {
		text := ""
		if d.Input != nil {
			text = d.Input.Text
		}
		d.Done(button, text)
	}
	return true
}

func buttonsWidth(buttons []string) int {
	w := 0
	for i, b := range buttons {
		if i > 0 {
			w += 2
		}
		w += term.Width(b) + 2
	}
	return w
}

// buttonRows splits the buttons into rows that fit in w columns and returns
// the index of the first button of each row.
func (d *Dialog) buttonRows(w int) []int {
	rows := []int{0}
	for i := 1; i < len(d.Buttons); i++ {
		if buttonsWidth(d.Buttons[rows[len(rows)-1]:i+1]) > w {
			rows = append(rows, i)
		}
	}
	return rows
}

// Draw lays the dialog out as: a 2-column, 1-row gray margin, the frame,
// one space of padding, and the content (lines, field, buttons).
func (d *Dialog) Draw(c term.Canvas, w, h int) {
	body, frame := term.DialogStyle, term.DialogFrameStyle
	if d.Danger {
		body, frame = term.ErrorStyle, term.ErrorStyle
	}
	cw := max(buttonsWidth(d.Buttons), term.Width(d.Title)+2)
	for _, l := range d.Lines {
		cw = max(cw, term.Width(l))
	}
	if d.Input != nil {
		cw = max(cw, inputMinWidth)
	}
	cw = min(cw, w-10)
	rows := d.buttonRows(cw)
	ww, wh := cw+8, len(d.Lines)+len(rows)+4
	if d.Input != nil {
		wh++
	}
	ox, ow := 0, w
	if d.Over != nil {
		ox, ow = d.Over(w)
	}
	x := max(min(ox+(ow-ww)/2, w-ww), 0)
	y := max((h-wh)/2, 0)
	window(c, x, y, ww, wh, body)
	c.Box(x+2, y+1, ww-4, wh-2, frame)
	drawTitle(c, x, y+1, ww, d.Title, frame)

	cx, row := x+4, y+2
	for _, l := range d.Lines {
		lw := min(term.Width(l), cw)
		c.Text(cx+(cw-lw)/2, row, l, cw, body)
		row++
	}
	if d.Input != nil {
		d.Input.Draw(c, cx, row, cw)
		row++
	}
	d.buttonAreas = d.buttonAreas[:0]
	for r, first := range rows {
		end := len(d.Buttons)
		if r+1 < len(rows) {
			end = rows[r+1]
		}
		bx := cx + max(cw-buttonsWidth(d.Buttons[first:end]), 0)/2
		for i := first; i < end; i++ {
			st := term.ButtonStyle
			if i == d.Focus {
				st = term.ButtonFocusStyle
			}
			bw := term.Width(d.Buttons[i]) + 2
			c.Text(bx, row, " "+d.Buttons[i]+" ", bw, st)
			d.buttonAreas = append(d.buttonAreas, area{bx, row, bw, 1})
			bx += bw + 2
		}
		row++
	}
}
