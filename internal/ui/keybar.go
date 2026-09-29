package ui

import (
	"strconv"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// DrawKeyBar draws NC's bottom row: the numbers 1…10 and their labels.
func DrawKeyBar(c term.Canvas, y, w int, labels [10]string) {
	c.HLine(0, y, w, ' ', term.KeyNumStyle)
	for i, label := range labels {
		x0, x1 := i*w/10, (i+1)*w/10
		n := c.Text(x0, y, strconv.Itoa(i+1), x1-x0, term.KeyNumStyle)
		c.Text(x0+n, y, term.Fit(label, x1-x0-n), x1-x0-n, term.KeyLabelStyle)
	}
}

// KeyBarKey returns the F-key at column x of a key bar w columns wide.
func KeyBarKey(x, w int) tcell.Key {
	i := 0
	for i < 9 && x >= (i+1)*w/10 {
		i++
	}
	return tcell.KeyF1 + tcell.Key(i)
}
