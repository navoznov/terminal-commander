package app

import "github.com/navoznov/terminal-commander/internal/ui"

func (a *App) confirmQuit() {
	a.modals.Push(&ui.Dialog{
		Title:   "Terminal Commander",
		Lines:   []string{"Do you want to quit Terminal Commander?"},
		Buttons: []string{"Yes", "No"},
		Done: func(b int, _ string) {
			if b == 0 {
				a.quit = true
			}
		},
	})
}

func (a *App) showHelp() {
	a.modals.Push(&ui.TextView{Title: "Help", Lines: helpLines})
}
