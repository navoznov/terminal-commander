package app

import (
	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

const (
	minWidth  = 80
	minHeight = 24
)

var keyLabels = [10]string{"Help", "Menu", "View", "Edit", "Copy", "RenMov", "Mkdir", "Delete", "PullDn", "Quit"}

// Draw lays out the whole screen: two panels, the command line and the key bar.
func (a *App) Draw() {
	c := term.Canvas{Screen: a.screen}
	w, h := a.screen.Size()
	a.screen.Clear()
	if w < minWidth || h < minHeight {
		a.screen.HideCursor()
		msg := "Window too small"
		c.Text(max(0, (w-len(msg))/2), h/2, msg, w, term.CmdLineStyle)
		return
	}
	ph := h - 2
	for i, p := range a.panels {
		x, pw := panelSpan(i, w)
		p.SetRows(ph - 5)
		p.Draw(c, x, 0, pw, ph, i == a.active, a.home)
	}
	a.drawCmdLine(c, h-2, w)
	ui.DrawKeyBar(c, h-1, w, keyLabels)
	if a.searching {
		a.drawSearch(c, w, ph)
	}
	if !a.modals.Empty() {
		a.screen.HideCursor()
		a.modals.Draw(c, w, h)
	}
}

// panelSpan returns the columns of panel i on a screen w columns wide; the
// right panel gets the odd column.
func panelSpan(i, w int) (x, width int) {
	lw := w / 2
	if i == 0 {
		return 0, lw
	}
	return lw, w - lw
}

// prompt is the command line prompt: the active panel's path and ">".
func (a *App) prompt() string {
	return fs.DisplayPath(a.panels[a.active].Path, a.home) + ">"
}

// drawCmdLine shows the prompt and the command; when they are too long,
// their end is shown, with the cursor after it.
func (a *App) drawCmdLine(c term.Canvas, y, w int) {
	c.HLine(0, y, w, ' ', term.CmdLineStyle)
	n := c.Text(0, y, term.Tail(a.prompt()+a.cmd.Text, w-1), w, term.CmdLineStyle)
	a.screen.ShowCursor(n, y)
}
