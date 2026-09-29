package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// window fills a w×h rectangle and draws NC's shadow: two columns to the
// right and one row down.
func window(c term.Canvas, x, y, w, h int, st tcell.Style) {
	c.Fill(x+2, y+1, w, h, ' ', term.ShadowStyle)
	c.Fill(x, y, w, h, ' ', st)
}

// drawTitle centers " title " in row y of a window at x of width w.
func drawTitle(c term.Canvas, x, y, w int, title string, st tcell.Style) {
	if title == "" {
		return
	}
	t := " " + title + " "
	tw := term.Width(t)
	c.Text(x+(w-tw)/2, y, t, tw, st)
}
