// Package keytest shows how key presses reach the program (tc --keytest).
package keytest

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/keys"
	"github.com/navoznov/terminal-commander/internal/term"
)

// Describe formats a raw key event and the result of normalizing it.
func Describe(raw, norm *tcell.EventKey) string {
	return fmt.Sprintf("%-22s key=%-4d rune=%-6q mod=%-2d -> %s",
		raw.Name(), raw.Key(), raw.Rune(), raw.Modifiers(), norm.Name())
}

// Run prints every key event until Control-C.
func Run(s tcell.Screen) {
	var n keys.Normalizer
	lines := []string{"Press keys to see how they are recognised. Control-C quits."}
	for {
		s.Clear()
		c := term.Canvas{Screen: s}
		w, h := s.Size()
		for i, l := range lines[max(0, len(lines)-h):] {
			c.Text(0, i, l, w, tcell.StyleDefault)
		}
		s.Show()
		switch ev := s.PollEvent().(type) {
		case nil:
			return
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlC {
				return
			}
			lines = append(lines, Describe(ev, n.Feed(ev, ev.When())))
		}
	}
}
