// Package app wires the panels, key handling and screen layout together.
package app

import (
	"os"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/keys"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

type App struct {
	screen     tcell.Screen
	panels     [2]*panel.Panel
	active     int
	keys       keys.Normalizer
	showHidden bool
	home       string
	modals     ui.Stack
	quit       bool
}

// New opens the left and right panels in the given directories, falling back
// to the home directory and then "/".
func New(s tcell.Screen, leftDir, rightDir string) *App {
	home, _ := os.UserHomeDir()
	a := &App{screen: s, home: home}
	for i, dir := range []string{leftDir, rightDir} {
		p := panel.New()
		if err := p.Load(dir); err != nil {
			a.report(err)
			if p.Load(home) != nil {
				a.report(p.Load("/"))
			}
		}
		a.panels[i] = p
	}
	return a
}

func (a *App) Run() {
	for !a.quit {
		a.Draw()
		a.screen.Show()
		a.HandleEvent(a.screen.PollEvent())
	}
}

func (a *App) HandleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case nil:
		a.quit = true // screen finalized
	case *tcell.EventResize:
		a.screen.Sync()
	case *tcell.EventKey:
		ev = a.keys.Feed(ev, ev.When())
		if a.modals.Empty() {
			a.handleKey(ev)
		} else {
			a.modals.HandleKey(ev)
		}
	}
}

// report shows err, if any, in a red dialog.
func (a *App) report(err error) {
	if err == nil {
		return
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Error",
		Lines:   term.Wrap(err.Error(), 60),
		Buttons: []string{"OK"},
		Danger:  true,
	})
}

func (a *App) handleKey(ev *tcell.EventKey) {
	p := a.panels[a.active]
	switch ev.Key() {
	case tcell.KeyF10:
		a.confirmQuit()
	case tcell.KeyF1:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			a.chooseDrive(0)
		} else {
			a.showHelp()
		}
	case tcell.KeyF2:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			a.chooseDrive(1)
		} else {
			a.openMenu()
		}
	case tcell.KeyF9:
		a.openMenu()
	case tcell.KeyTab:
		a.active = 1 - a.active
	case tcell.KeyUp:
		p.Move(-1)
	case tcell.KeyDown:
		p.Move(1)
	case tcell.KeyLeft:
		p.Left()
	case tcell.KeyRight:
		p.Right()
	case tcell.KeyHome:
		p.Home()
	case tcell.KeyEnd:
		p.End()
	case tcell.KeyPgUp:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			a.report(p.Up())
		} else {
			p.PageUp()
		}
	case tcell.KeyPgDn:
		p.PageDown()
	case tcell.KeyEnter:
		_, err := p.Enter()
		a.report(err)
	case tcell.KeyBackspace:
		a.report(p.Up())
	case tcell.KeyInsert:
		p.ToggleSelect()
	case tcell.KeyCtrlR:
		a.report(p.Reload())
	case tcell.KeyCtrlT:
		if p.Mode == panel.Brief {
			p.SetMode(panel.Full)
		} else {
			p.SetMode(panel.Brief)
		}
	case tcell.KeyCtrlU:
		a.swapPanels()
	case tcell.KeyRune:
		a.handleRune(ev.Rune(), ev.Modifiers())
	}
}

func (a *App) handleRune(r rune, mod tcell.ModMask) {
	p := a.panels[a.active]
	switch {
	case (r == '.' || r == 'ю') && mod&tcell.ModAlt != 0: // 'ю' is the '.' key on the Russian layout
		a.toggleHidden()
	case r == '1' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Brief)
	case r == '2' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Full)
	case r == ' ' && mod == 0:
		p.ToggleSelect()
	case r == '+' && mod == 0:
		a.askMask(true)
	case r == '-' && mod == 0:
		a.askMask(false)
	case r == '*' && mod == 0:
		p.InvertSelection()
	}
}

func (a *App) swapPanels() {
	a.panels[0], a.panels[1] = a.panels[1], a.panels[0]
	a.active = 1 - a.active
}

func (a *App) toggleHidden() {
	a.showHidden = !a.showHidden
	for _, p := range a.panels {
		a.report(p.SetShowHidden(a.showHidden))
	}
}
