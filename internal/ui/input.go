package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// Input is a one-line text field. The cursor stays at the end, like the NC
// command line. Until the first edit the initial text is fresh: typing
// replaces it, Backspace edits it.
type Input struct {
	Text  string
	fresh bool
}

func NewInput(text string) *Input {
	return &Input{Text: text, fresh: true}
}

// HandleKey edits the text and reports whether the key was used.
func (in *Input) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyBackspace:
		if rs := []rune(in.Text); len(rs) > 0 {
			in.Text = string(rs[:len(rs)-1])
		}
	case tcell.KeyRune:
		if ev.Modifiers()&(tcell.ModAlt|tcell.ModCtrl) != 0 {
			return false
		}
		if in.fresh {
			in.Text = ""
		}
		in.Text += string(ev.Rune())
	default:
		return false
	}
	in.fresh = false
	return true
}

// Draw draws the field w columns wide at (x, y) and puts the cursor after
// the text. A long text shows its end.
func (in *Input) Draw(c term.Canvas, x, y, w int) {
	c.HLine(x, y, w, ' ', term.InputStyle)
	n := c.Text(x, y, term.Tail(in.Text, w-1), w, term.InputStyle)
	c.Screen.ShowCursor(x+n, y)
}
