package app

import (
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// Quick search: Alt-letter opens a search field in the active panel's
// status line, and the cursor jumps to the first name starting with the
// typed text. A letter that finds nothing is not added. Esc ends the
// search; any other key ends it and does its usual work.

func (a *App) startSearch(r rune) {
	a.searching, a.search = true, ""
	a.searchRune(r)
}

func (a *App) searchRune(r rune) {
	if s := a.search + string(r); a.panels[a.active].FindPrefix(s) {
		a.search = s
	}
}

// handleSearchKey handles a key while searching and reports whether the
// key was used up.
func (a *App) handleSearchKey(ev *tcell.EventKey) bool {
	switch {
	case ev.Key() == tcell.KeyRune && ev.Modifiers()&tcell.ModCtrl == 0:
		a.searchRune(ev.Rune())
		return true
	case ev.Key() == tcell.KeyBackspace:
		_, n := utf8.DecodeLastRuneInString(a.search)
		a.search = a.search[:len(a.search)-n]
		if a.search != "" {
			a.panels[a.active].FindPrefix(a.search)
		}
		return true
	case ev.Key() == tcell.KeyEscape:
		a.searching = false
		a.keys.Disarm() // the Esc was used; don't make the next key Alt-key
		return true
	}
	a.searching = false
	return false
}

// drawSearch shows the search field over the active panel's status line,
// with the cursor after the text.
func (a *App) drawSearch(c term.Canvas, w, ph int) {
	x, pw := panelSpan(a.active, w)
	y := ph - 2
	c.HLine(x+1, y, pw-2, ' ', term.PanelStyle)
	n := c.Text(x+1, y, term.Tail("Search: "+a.search, pw-3), pw-3, term.PanelStyle)
	a.screen.ShowCursor(x+1+n, y)
}
