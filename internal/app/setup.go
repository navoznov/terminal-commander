package app

import (
	"github.com/navoznov/terminal-commander/internal/config"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

var (
	modeNames = map[panel.Mode]string{panel.Brief: "brief", panel.Full: "full"}
	sortNames = map[panel.SortMode]string{
		panel.SortName: "name", panel.SortExt: "extension", panel.SortTime: "time",
		panel.SortSize: "size", panel.Unsorted: "unsorted",
	}
)

// lookup returns the key of name in names, or the zero key (the default)
// for an unknown name.
func lookup[K comparable](names map[K]string, name string) K {
	for k, n := range names {
		if n == name {
			return k
		}
	}
	var zero K
	return zero
}

// setupPanel makes a panel with the saved mode, sorting, hidden files and
// size format; the caller loads its directory.
func setupPanel(c config.Panel, showHidden, humanSizes bool) *panel.Panel {
	p := panel.New()
	p.Mode = lookup(modeNames, c.Mode)
	p.Sort = lookup(sortNames, c.Sort)
	p.ShowHidden = showHidden
	p.HumanSizes = humanSizes
	return p
}

// setup is the current setup to save.
func (a *App) setup() config.Config {
	save := func(p *panel.Panel) config.Panel {
		return config.Panel{Path: p.Path, Mode: modeNames[p.Mode], Sort: sortNames[p.Sort]}
	}
	return config.Config{
		Left:       save(a.panels[0]),
		Right:      save(a.panels[1]),
		ShowHidden: a.showHidden,
		HumanSizes: a.humanSizes,
		History:    append([]string{}, a.cmd.History()...), // [] rather than null in the file
	}
}

// saveSetup writes the setup to the config file; without one (in tests)
// it does nothing.
func (a *App) saveSetup() error {
	if a.cfgPath == "" {
		return nil
	}
	return config.Save(a.cfgPath, a.setup())
}

// quitSaving saves the setup and quits. If saving fails, the error is shown
// first and tc quits after it is closed.
func (a *App) quitSaving() {
	err := a.saveSetup()
	if err == nil {
		a.quit = true
		return
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Error",
		Lines:   term.Wrap("Cannot save setup: "+err.Error(), 60),
		Buttons: []string{"OK"},
		Danger:  true,
		Done:    func(int, string) { a.quit = true },
	})
}
