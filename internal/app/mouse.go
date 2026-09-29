package app

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ui"
)

// handleMouse acts on the mouse when no window is open: a click picks the
// panel and the entry, a double click acts on the entry like Enter, the
// wheel moves the cursor of the panel under the pointer, and a click on
// the key bar presses its key.
func (a *App) handleMouse(m ui.Mouse) {
	w, h := a.screen.Size()
	if w < minWidth || h < minHeight {
		return
	}
	if m.Y == h-1 {
		if m.Action == ui.Click {
			a.handleKey(tcell.NewEventKey(ui.KeyBarKey(m.X, w), 0, 0))
		}
		return
	}
	if m.Y >= h-2 {
		return // the command line
	}
	i := 0
	if x, _ := panelSpan(1, w); m.X >= x {
		i = 1
	}
	p := a.panels[i]
	switch m.Action {
	case ui.WheelUp:
		p.Move(-1)
	case ui.WheelDown:
		p.Move(1)
	case ui.Click, ui.DoubleClick:
		a.active = i
		k, ok := p.Hit(m.X, m.Y)
		if !ok {
			return
		}
		p.Cursor = k
		if m.Action == ui.DoubleClick {
			a.enter()
		}
	}
}
