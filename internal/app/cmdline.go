package app

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/shell"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// execute runs the command line: a plain cd changes the active panel's
// directory, anything else goes to the shell.
func (a *App) execute() {
	line := a.cmd.Commit()
	p := a.panels[a.active]
	if dir, ok := shell.ParseCd(line); ok {
		a.cd(p, dir)
		return
	}
	a.runLine(p, line)
}

// cd changes p's directory the way the shell's cd would; "-" goes back to
// the previous one.
func (a *App) cd(p *panel.Panel, dir string) {
	if dir == "-" {
		if p.Prev == "" {
			return
		}
		dir = p.Prev
	}
	a.report(p.Load(resolve(p.Path, a.home, dir)))
}

// runLine shows the prompt and line on the terminal's own screen, runs line
// in p's directory, waits for a key and re-reads the panels.
func (a *App) runLine(p *panel.Panel, line string) {
	prompt := a.prompt()
	a.outside(func() {
		fmt.Fprintln(a.console.Out, prompt+line)
		a.console.Run(p.Path, line)
		a.console.Pause()
	})
	a.reloadPanels()
}

// outside hides the panels and gives the terminal to f.
func (a *App) outside(f func()) {
	if err := a.screen.Suspend(); err != nil {
		a.report(err)
		return
	}
	f()
	a.report(a.screen.Resume())
	a.screen.Sync()
}

// openCmd opens a document with its app.
var openCmd = "open"

// enter acts on the entry under the cursor: a directory is entered, a
// program is run, anything else is opened with its app.
func (a *App) enter() {
	p := a.panels[a.active]
	if ok, err := p.Enter(); ok || err != nil {
		a.report(err)
		return
	}
	e := p.Current()
	if e == nil {
		return
	}
	path := filepath.Join(p.Path, e.Name)
	if shell.Runnable(path) {
		a.runLine(p, "./"+shell.Quote(e.Name))
		return
	}
	if out, err := exec.Command(openCmd, path).CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			err = errors.New(msg)
		}
		a.report(err)
	}
}

// insertName appends the name under the cursor to the command line.
func (a *App) insertName() {
	e := a.panels[a.active].Current()
	if e == nil || e.IsUp {
		return
	}
	a.cmd.Insert(shell.Quote(e.Name) + " ")
}

// showHistory lists the commands run; the picked one goes to the command
// line.
func (a *App) showHistory() {
	h := a.cmd.History()
	a.modals.Push(&ui.List{
		Title: "History",
		Items: h,
		Cur:   len(h) - 1,
		Done: func(i int) {
			if i >= 0 {
				a.cmd.Text = h[i]
			}
		},
	})
}

// panelsOff shows the terminal's own screen, with the output of earlier
// commands, until a key is pressed.
func (a *App) panelsOff() {
	a.outside(a.console.WaitKey)
}
