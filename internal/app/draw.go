package app

import (
	"strconv"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
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
	lw := w / 2
	for i, p := range a.panels {
		x, pw := 0, lw
		if i == 1 {
			x, pw = lw, w-lw
		}
		p.SetRows(ph - 5)
		p.Draw(c, x, 0, pw, ph, i == a.active, a.home)
	}
	a.drawCmdLine(c, h-2, w)
	a.drawKeyBar(c, h-1, w)
}

func (a *App) drawCmdLine(c term.Canvas, y, w int) {
	c.HLine(0, y, w, ' ', term.CmdLineStyle)
	if a.errMsg != "" {
		c.Text(0, y, a.errMsg, w, term.ErrorStyle)
		a.screen.HideCursor()
		return
	}
	prompt := fs.DisplayPath(a.panels[a.active].Path, a.home) + ">"
	n := c.Text(0, y, prompt, w, term.CmdLineStyle)
	a.screen.ShowCursor(n, y)
}

func (a *App) drawKeyBar(c term.Canvas, y, w int) {
	c.HLine(0, y, w, ' ', term.KeyNumStyle)
	for i, label := range keyLabels {
		x0, x1 := i*w/10, (i+1)*w/10
		n := c.Text(x0, y, strconv.Itoa(i+1), x1-x0, term.KeyNumStyle)
		c.Text(x0+n, y, term.Fit(label, x1-x0-n), x1-x0-n, term.KeyLabelStyle)
	}
}
