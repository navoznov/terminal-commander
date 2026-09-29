package term

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"golang.org/x/text/unicode/norm"
)

// Canvas draws NC-style primitives on a tcell screen.
type Canvas struct {
	Screen tcell.Screen
}

func (c Canvas) Put(x, y int, r rune, st tcell.Style) {
	c.Screen.SetContent(x, y, r, nil, st)
}

// Text draws s (normalized to NFC) at (x, y) using at most max columns and
// returns the number of columns drawn. Zero-width runes are skipped.
func (c Canvas) Text(x, y int, s string, max int, st tcell.Style) int {
	used := 0
	for _, r := range norm.NFC.String(s) {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			continue
		}
		if used+rw > max {
			break
		}
		c.Screen.SetContent(x+used, y, r, nil, st)
		used += rw
	}
	return used
}

func (c Canvas) Fill(x, y, w, h int, r rune, st tcell.Style) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c.Put(x+i, y+j, r, st)
		}
	}
}

func (c Canvas) HLine(x, y, w int, r rune, st tcell.Style) {
	c.Fill(x, y, w, 1, r, st)
}

func (c Canvas) VLine(x, y, h int, r rune, st tcell.Style) {
	c.Fill(x, y, 1, h, r, st)
}

// Box draws a double-line frame.
func (c Canvas) Box(x, y, w, h int, st tcell.Style) {
	if w < 2 || h < 2 {
		return
	}
	c.HLine(x+1, y, w-2, '═', st)
	c.HLine(x+1, y+h-1, w-2, '═', st)
	c.VLine(x, y+1, h-2, '║', st)
	c.VLine(x+w-1, y+1, h-2, '║', st)
	c.Put(x, y, '╔', st)
	c.Put(x+w-1, y, '╗', st)
	c.Put(x, y+h-1, '╚', st)
	c.Put(x+w-1, y+h-1, '╝', st)
}
