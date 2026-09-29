package app

import (
	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func (a *App) confirmQuit() {
	a.modals.Push(&ui.Dialog{
		Title:   "Terminal Commander",
		Lines:   []string{"Do you want to quit Terminal Commander?"},
		Buttons: []string{"Yes", "No"},
		Done: func(b int, _ string) {
			if b == 0 {
				a.quitSaving()
			}
		},
	})
}

func (a *App) showHelp() {
	a.modals.Push(&ui.TextView{Title: "Help", Lines: helpLines})
}

func (a *App) notImplemented() {
	a.modals.Push(&ui.Dialog{Lines: []string{"Not implemented yet"}, Buttons: []string{"OK"}})
}

const driveLabelWidth = 12

// driveLabel cuts a volume name to 12 columns, NC style.
func driveLabel(name string) string {
	if term.Width(name) > driveLabelWidth {
		return term.Fit(name, driveLabelWidth)
	}
	return name
}

// chooseDrive opens the drive dialog over panel i.
func (a *App) chooseDrive(i int) {
	drives := fs.Drives(a.home)
	labels := make([]string, len(drives))
	for k, d := range drives {
		labels[k] = driveLabel(d.Name)
	}
	side := "left"
	if i == 1 {
		side = "right"
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Drive",
		Lines:   []string{"Choose " + side + " drive:"},
		Buttons: labels,
		Over:    func(w int) (int, int) { return panelSpan(i, w) },
		Done: func(b int, _ string) {
			if b >= 0 {
				a.report(a.panels[i].Load(drives[b].Path))
			}
		},
	})
}

// askMask asks for a mask and selects (on) or unselects matching files in
// the active panel.
func (a *App) askMask(on bool) {
	title := "Select"
	if !on {
		title = "Unselect"
	}
	a.modals.Push(&ui.Dialog{
		Title:   title,
		Input:   ui.NewInput("*"),
		Buttons: []string{"OK", "Cancel"},
		Done: func(b int, mask string) {
			if b == 0 {
				a.report(a.panels[a.active].SelectMask(mask, on))
			}
		},
	})
}
