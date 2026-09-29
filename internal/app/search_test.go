package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// newSearchApp adds files "alpha", "beta" and "bravo" to newApp's directory.
func newSearchApp(t *testing.T) (*App, string) {
	a, dir := newApp(t)
	for _, n := range []string{"alpha", "beta", "bravo"} {
		must(t, os.WriteFile(filepath.Join(dir, n), nil, 0o644))
	}
	must(t, a.panels[0].Reload())
	a.Draw()
	return a, dir
}

func current(a *App) string { return a.panels[a.active].Current().Name }

// statusRow is the left panel's status line (y = 21 on the 80×25 test screen).
func statusRow(a *App) string { return screenRow(a, 21) }

func TestAltLetterSearches(t *testing.T) {
	a, _ := newSearchApp(t)
	press(a, tcell.KeyRune, 'b', tcell.ModAlt)
	if current(a) != "beta" {
		t.Fatalf("current %q", current(a))
	}
	typeText(a, "r")
	if current(a) != "bravo" || a.cmd.Text != "" {
		t.Fatalf("current %q, command line %q", current(a), a.cmd.Text)
	}
	if row := statusRow(a); !strings.HasPrefix(row, "║Search: br ") {
		t.Fatalf("status %q", row)
	}
	if x, y, _ := a.screen.(tcell.SimulationScreen).GetCursor(); x != 11 || y != 21 {
		t.Fatalf("cursor at %d,%d", x, y)
	}
	typeText(a, "x") // no "brx": ignored
	if a.search != "br" || current(a) != "bravo" {
		t.Fatalf("search %q current %q", a.search, current(a))
	}
	press(a, tcell.KeyBackspace, 0, 0)
	if a.search != "b" || current(a) != "beta" {
		t.Fatalf("search %q current %q", a.search, current(a))
	}
	press(a, tcell.KeyBackspace, 0, 0)
	if !a.searching || a.search != "" || current(a) != "beta" {
		t.Fatalf("searching %v %q current %q", a.searching, a.search, current(a))
	}
	typeText(a, "a")
	if current(a) != "alpha" {
		t.Fatalf("current %q", current(a))
	}
}

func TestEscEndsSearch(t *testing.T) {
	a, _ := newSearchApp(t)
	press(a, tcell.KeyRune, 'b', tcell.ModAlt)
	press(a, tcell.KeyEscape, 0, 0)
	if a.searching || current(a) != "beta" {
		t.Fatalf("searching %v current %q", a.searching, current(a))
	}
	if strings.Contains(statusRow(a), "Search:") {
		t.Fatalf("status %q", statusRow(a))
	}
	typeText(a, "l") // not Alt-l after the Esc
	if a.searching || a.cmd.Text != "l" {
		t.Fatalf("searching %v command line %q", a.searching, a.cmd.Text)
	}
}

func TestKeysEndSearchAndWork(t *testing.T) {
	a, dir := newSearchApp(t)
	press(a, tcell.KeyRune, 's', tcell.ModAlt)
	press(a, tcell.KeyEnter, 0, 0)
	if a.searching || a.panels[0].Path != filepath.Join(dir, "sub") {
		t.Fatalf("searching %v path %s", a.searching, a.panels[0].Path)
	}
	press(a, tcell.KeyRune, 'f', tcell.ModAlt)
	press(a, tcell.KeyTab, 0, 0)
	if a.searching || a.active != 1 {
		t.Fatalf("searching %v active %d", a.searching, a.active)
	}
}

func TestEscLetterSearches(t *testing.T) {
	a, _ := newSearchApp(t)
	a.HandleEvent(tcell.NewEventKey(tcell.KeyEscape, 0, 0))
	press(a, tcell.KeyRune, 'b', 0)
	if !a.searching || current(a) != "beta" || a.cmd.Text != "" {
		t.Fatalf("searching %v current %q command line %q", a.searching, current(a), a.cmd.Text)
	}
}

func TestClickEndsSearch(t *testing.T) {
	a, _ := newSearchApp(t)
	press(a, tcell.KeyRune, 'b', tcell.ModAlt)
	a.HandleEvent(tcell.NewEventMouse(5, 5, tcell.Button1, 0))
	if a.searching {
		t.Fatal("still searching")
	}
}
