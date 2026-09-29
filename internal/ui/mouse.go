package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

type MouseAction int

const (
	Click MouseAction = iota + 1
	DoubleClick
	WheelUp
	WheelDown
)

// Mouse is what the mouse did at a screen cell.
type Mouse struct {
	X, Y   int
	Action MouseAction
}

// MouseView is a View that takes the mouse. HandleMouse reports whether the
// view is finished, like HandleKey.
type MouseView interface {
	HandleMouse(m Mouse) bool
}

// DoubleClickTime is how soon a second click in the same cell makes a
// double click.
const DoubleClickTime = 400 * time.Millisecond

// Clicker turns tcell's mouse events, which report the buttons held down,
// into clicks, double clicks and wheel turns.
type Clicker struct {
	down   bool // the left button is held
	x, y   int
	lastAt time.Time // the last click, zero after a double click
}

// Feed takes a mouse event that came at the given time and returns the
// action, if it makes one.
func (c *Clicker) Feed(ev *tcell.EventMouse, at time.Time) (Mouse, bool) {
	x, y := ev.Position()
	b := ev.Buttons()
	switch {
	case b&tcell.WheelUp != 0:
		return Mouse{x, y, WheelUp}, true
	case b&tcell.WheelDown != 0:
		return Mouse{x, y, WheelDown}, true
	}
	wasDown := c.down
	c.down = b&tcell.Button1 != 0
	if !c.down || wasDown {
		return Mouse{}, false
	}
	m := Mouse{x, y, Click}
	if x == c.x && y == c.y && !c.lastAt.IsZero() && at.Sub(c.lastAt) <= DoubleClickTime {
		m.Action = DoubleClick
		at = time.Time{} // a third click starts over
	}
	c.x, c.y, c.lastAt = x, y, at
	return m, true
}

// HandleMouse sends m to the top view if it takes the mouse, and removes
// the view when it is finished.
func (s *Stack) HandleMouse(m Mouse) {
	if v, ok := s.Top().(MouseView); ok && v.HandleMouse(m) {
		s.Remove(s.Top())
	}
}

// area is a rectangle on the screen that the mouse can hit.
type area struct{ x, y, w, h int }

func (r area) has(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}
