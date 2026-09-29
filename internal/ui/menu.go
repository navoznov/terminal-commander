package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// Item is one line of a pull-down menu; an empty Label makes a separator.
type Item struct {
	Label   string
	Key     string // hot key shown on the right
	Checked bool
	Action  func()
}

type Menu struct {
	Title string
	Items []Item
}

// MenuBar is the F9 menu: the bar in the top row with one pull-down open.
// Every menu must have at least one item that is not a separator.
type MenuBar struct {
	Menus []Menu
	Cur   int // open menu
	Sel   int // selected item
}

const barX = 2

func NewMenuBar(menus []Menu, cur int) *MenuBar {
	m := &MenuBar{Menus: menus}
	m.open(cur)
	return m
}

func (m *MenuBar) open(i int) {
	n := len(m.Menus)
	m.Cur = (i%n + n) % n
	m.Sel = -1
	m.move(1)
}

// move selects the next item that is not a separator in direction d,
// wrapping around.
func (m *MenuBar) move(d int) {
	items := m.Menus[m.Cur].Items
	n := len(items)
	for i := 1; i <= n; i++ {
		j := ((m.Sel+d*i)%n + n) % n
		if items[j].Label != "" {
			m.Sel = j
			return
		}
	}
}

func (m *MenuBar) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyF9, tcell.KeyF10:
		return true
	case tcell.KeyLeft:
		m.open(m.Cur - 1)
	case tcell.KeyRight:
		m.open(m.Cur + 1)
	case tcell.KeyUp:
		m.move(-1)
	case tcell.KeyDown:
		m.move(1)
	case tcell.KeyHome:
		m.Sel = -1
		m.move(1)
	case tcell.KeyEnd:
		m.Sel = len(m.Menus[m.Cur].Items)
		m.move(-1)
	case tcell.KeyEnter:
		if a := m.Menus[m.Cur].Items[m.Sel].Action; a != nil {
			a()
		}
		return true
	}
	return false
}

// titleX is the column where the " Title " of menu i starts.
func (m *MenuBar) titleX(i int) int {
	x := barX
	for _, mm := range m.Menus[:i] {
		x += term.Width(mm.Title) + 4
	}
	return x
}

func (m *MenuBar) Draw(c term.Canvas, w, h int) {
	c.HLine(0, 0, w, ' ', term.MenuStyle)
	for i, mm := range m.Menus {
		st := term.MenuStyle
		if i == m.Cur {
			st = term.MenuSelStyle
		}
		t := " " + mm.Title + " "
		c.Text(m.titleX(i), 0, t, term.Width(t), st)
	}
	m.drawDropdown(c, w)
}

func (m *MenuBar) drawDropdown(c term.Canvas, w int) {
	items := m.Menus[m.Cur].Items
	lw, kw := 0, 0
	for _, it := range items {
		lw = max(lw, term.Width(it.Label))
		kw = max(kw, term.Width(it.Key))
	}
	gap := ""
	if kw > 0 {
		gap = "  "
	}
	iw := 1 + lw + len(gap) + kw + 1
	bw, bh := iw+2, len(items)+2
	x := max(min(m.titleX(m.Cur), w-bw-2), 0)
	y := 1
	window(c, x, y, bw, bh, term.MenuStyle)
	c.Box(x, y, bw, bh, term.MenuStyle)
	for k, it := range items {
		row := y + 1 + k
		if it.Label == "" {
			c.Put(x, row, '╟', term.MenuStyle)
			c.HLine(x+1, row, iw, '─', term.MenuStyle)
			c.Put(x+bw-1, row, '╢', term.MenuStyle)
			continue
		}
		check := " "
		if it.Checked {
			check = "√"
		}
		key := strings.Repeat(" ", kw-term.Width(it.Key)) + it.Key
		st := term.MenuStyle
		if k == m.Sel {
			st = term.MenuSelStyle
		}
		c.Text(x+1, row, check+term.Fit(it.Label, lw)+gap+key+" ", iw, st)
	}
}
