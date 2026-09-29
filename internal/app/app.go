// Package app wires the panels, key handling and screen layout together.
package app

import (
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/config"
	"github.com/navoznov/terminal-commander/internal/keys"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/shell"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

type App struct {
	screen     tcell.Screen
	panels     [2]*panel.Panel
	active     int
	keys       keys.Normalizer
	clicker    ui.Clicker
	showHidden bool
	home       string
	modals     ui.Stack
	calls      chan func()   // work for the UI goroutine, sent by operations
	op         *op           // the running file operation, or nil
	cmd        shell.Line    // the command line
	console    shell.Console // where commands run
	cfgPath    string        // where the setup is saved; "" in tests
	quit       bool
}

// New sets tc up from cfg and opens the panels in its directories, falling
// back to the home directory and then "/". The setup is saved to cfgPath;
// "" means it is not saved.
func New(s tcell.Screen, cfg config.Config, cfgPath string) *App {
	home, _ := os.UserHomeDir()
	a := &App{screen: s, home: home, calls: make(chan func(), 16), console: shell.Console{In: os.Stdin, Out: os.Stdout}}
	a.cfgPath = cfgPath
	a.showHidden = cfg.ShowHidden
	a.cmd.SetHistory(cfg.History)
	for i, c := range []config.Panel{cfg.Left, cfg.Right} {
		p := setupPanel(c, cfg.ShowHidden)
		if err := p.Load(c.Path); err != nil {
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
	events := make(chan tcell.Event)
	go a.screen.ChannelEvents(events, nil)
	for !a.quit {
		a.Draw()
		a.screen.Show()
		select {
		case ev, ok := <-events:
			if !ok {
				return // screen finalized
			}
			a.HandleEvent(ev)
		case f := <-a.calls:
			f()
		}
	}
}

func (a *App) HandleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case nil:
		a.quit = true // screen finalized
	case *tcell.EventResize:
		a.screen.Sync()
	case *tcell.EventMouse:
		if m, ok := a.clicker.Feed(ev, ev.When()); ok {
			if a.modals.Empty() {
				a.handleMouse(m)
			} else {
				a.modals.HandleMouse(m)
			}
		}
	case *tcell.EventKey:
		ev = a.keys.Feed(ev, ev.When())
		if a.modals.Empty() {
			a.handleKey(ev)
		} else {
			a.modals.HandleKey(ev)
			if ev.Key() == tcell.KeyEscape {
				// Esc closed a window; don't turn the next key into Alt-key,
				// or Esc 9 in the menu would open it again.
				a.keys.Disarm()
			}
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
	case tcell.KeyF3:
		a.view()
	case tcell.KeyF4:
		a.edit()
	case tcell.KeyF5:
		a.transfer(false)
	case tcell.KeyF6:
		a.transfer(true)
	case tcell.KeyF7:
		a.mkdir()
	case tcell.KeyF8:
		if ev.Modifiers()&tcell.ModShift != 0 {
			a.remove()
		} else {
			a.trash()
		}
	case tcell.KeyF20: // Shift-F8 in terminals that send it as F20
		a.remove()
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
		switch {
		case ev.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0:
			a.insertName()
		case strings.TrimSpace(a.cmd.Text) != "":
			a.execute()
		default:
			a.cmd.Clear()
			a.enter()
		}
	case tcell.KeyCtrlJ: // Control-Enter in terminals that send it as LF
		a.insertName()
	case tcell.KeyBackspace:
		if a.cmd.Text != "" {
			a.cmd.Backspace()
		} else {
			a.report(p.Up())
		}
	case tcell.KeyEscape:
		if a.cmd.Text != "" {
			a.cmd.Clear()
			a.keys.Disarm() // the Esc was used; don't make the next key Alt-key
		}
	case tcell.KeyInsert:
		p.ToggleSelect()
	case tcell.KeyCtrlE:
		a.cmd.Prev()
	case tcell.KeyCtrlX:
		a.cmd.Next()
	case tcell.KeyCtrlO:
		a.panelsOff()
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
	empty := a.cmd.Text == ""
	switch {
	case (r == '.' || r == 'ю') && mod&tcell.ModAlt != 0: // 'ю' is the '.' key on the Russian layout
		a.toggleHidden()
	case r == '1' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Brief)
	case r == '2' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Full)
	case mod&(tcell.ModAlt|tcell.ModCtrl) != 0:
		// other Alt and Control keys do nothing
	case r == ' ' && empty:
		p.ToggleSelect()
	case r == '+' && empty:
		a.askMask(true)
	case r == '-' && empty:
		a.askMask(false)
	case r == '*' && empty:
		p.InvertSelection()
	default:
		a.cmd.Insert(string(r))
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
