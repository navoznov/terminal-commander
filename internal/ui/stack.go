// Package ui has the modal windows drawn over the panels: dialogs, the
// pull-down menu and the help window.
package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// View is a modal window.
type View interface {
	// Draw draws the view on a screen of w×h cells.
	Draw(c term.Canvas, w, h int)
	// HandleKey processes a key and reports whether the view is finished
	// and should be removed.
	HandleKey(ev *tcell.EventKey) bool
}

// Stack holds the open modal windows; the last one gets the keys.
type Stack struct {
	views []View
}

func (s *Stack) Push(v View) { s.views = append(s.views, v) }
func (s *Stack) Empty() bool { return len(s.views) == 0 }
func (s *Stack) Len() int    { return len(s.views) }

func (s *Stack) Top() View {
	if s.Empty() {
		return nil
	}
	return s.views[len(s.views)-1]
}

// Draw draws the views from the bottom up.
func (s *Stack) Draw(c term.Canvas, w, h int) {
	for _, v := range s.views {
		v.Draw(c, w, h)
	}
}

// HandleKey sends ev to the top view. A finished view is removed even when
// it opened another view while handling the key.
func (s *Stack) HandleKey(ev *tcell.EventKey) {
	v := s.Top()
	if v == nil || !v.HandleKey(ev) {
		return
	}
	s.Remove(v)
}

// Remove takes v off the stack wherever it is.
func (s *Stack) Remove(v View) {
	for i, x := range s.views {
		if x == v {
			s.views = append(s.views[:i], s.views[i+1:]...)
			return
		}
	}
}
