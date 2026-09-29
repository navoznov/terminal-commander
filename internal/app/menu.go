package app

import (
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// openMenu opens the F9 menu at the Left or Right menu of the active panel.
func (a *App) openMenu() {
	menus := a.menus()
	cur := 0
	if a.active == 1 {
		cur = len(menus) - 1
	}
	a.modals.Push(ui.NewMenuBar(menus, cur))
}

func (a *App) menus() []ui.Menu {
	return []ui.Menu{
		{Title: "Left", Items: a.panelItems(0)},
		{Title: "Files", Items: []ui.Item{
			{Label: "Help", Key: "F1", Action: a.showHelp},
			{Label: "View", Key: "F3", Action: a.notImplemented},
			{Label: "Edit", Key: "F4", Action: a.notImplemented},
			{Label: "Copy", Key: "F5", Action: func() { a.transfer(false) }},
			{Label: "Rename/Move", Key: "F6", Action: func() { a.transfer(true) }},
			{Label: "Make directory", Key: "F7", Action: a.mkdir},
			{Label: "Delete", Key: "F8", Action: a.trash},
			{Label: "Delete permanently", Key: "Shift-F8", Action: a.remove},
			{},
			{Label: "Select group", Key: "+", Action: func() { a.askMask(true) }},
			{Label: "Unselect group", Key: "-", Action: func() { a.askMask(false) }},
			{Label: "Invert selection", Key: "*", Action: func() { a.panels[a.active].InvertSelection() }},
			{},
			{Label: "Quit", Key: "F10", Action: a.confirmQuit},
		}},
		{Title: "Commands", Items: []ui.Item{
			{Label: "Swap panels", Key: "Control-U", Action: a.swapPanels},
			{Label: "Panels on/off", Key: "Control-O", Action: a.panelsOff},
			{Label: "Command history", Action: a.showHistory},
		}},
		{Title: "Options", Items: []ui.Item{
			{Label: "Show hidden files", Key: "Alt-.", Checked: a.showHidden, Action: a.toggleHidden},
			{Label: "Save setup", Action: func() { a.report(a.saveSetup()) }},
		}},
		{Title: "Right", Items: a.panelItems(1)},
	}
}

func (a *App) panelItems(i int) []ui.Item {
	p := a.panels[i]
	mode := func(label string, m panel.Mode) ui.Item {
		return ui.Item{Label: label, Checked: p.Mode == m, Action: func() { p.SetMode(m) }}
	}
	sort := func(label string, m panel.SortMode) ui.Item {
		return ui.Item{Label: label, Checked: p.Sort == m, Action: func() { a.report(p.SetSort(m)) }}
	}
	return []ui.Item{
		mode("Brief", panel.Brief),
		mode("Full", panel.Full),
		{},
		sort("Name", panel.SortName),
		sort("Extension", panel.SortExt),
		sort("Time", panel.SortTime),
		sort("Size", panel.SortSize),
		sort("Unsorted", panel.Unsorted),
		{},
		{Label: "Re-read", Action: func() { a.report(p.Reload()) }},
		{Label: "Drive…", Action: func() { a.chooseDrive(i) }},
	}
}
