package app

import (
	"fmt"

	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/shell"
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
