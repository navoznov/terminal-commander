package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ui"
)

// clickAt sends a press and a release of the left button at (x, y).
func clickAt(a *App, x, y int) {
	a.HandleEvent(tcell.NewEventMouse(x, y, tcell.Button1, 0))
	a.HandleEvent(tcell.NewEventMouse(x, y, 0, 0))
	a.Draw()
}

// The test screen is 80×25; the right panel starts at x = 40, the first
// entry row is y = 2, brief columns are 12 wide.
func TestClickPicksPanelAndEntry(t *testing.T) {
	a, _ := newApp(t)
	clickAt(a, 41, 3) // right panel, second entry
	if a.active != 1 || a.panels[1].Cursor != 1 {
		t.Fatalf("active %d cursor %d", a.active, a.panels[1].Cursor)
	}
	clickAt(a, 41, 20) // below the entries
	if a.panels[1].Cursor != 1 {
		t.Fatalf("cursor %d", a.panels[1].Cursor)
	}
}

func TestDoubleClickEntersDirectory(t *testing.T) {
	a, dir := newApp(t)
	a.panels[0].Focus("sub")
	y := 2 + a.panels[0].Cursor
	a.cmd.Text = "ls" // the command line is not run
	clickAt(a, 1, y)
	clickAt(a, 1, y)
	if a.panels[0].Path != filepath.Join(dir, "sub") || a.cmd.Text != "ls" {
		t.Fatalf("path %s text %q", a.panels[0].Path, a.cmd.Text)
	}
}

func TestWheelScrollsPanelUnderPointer(t *testing.T) {
	a, dir := newApp(t)
	for _, n := range []string{"a", "b", "c"} {
		must(t, os.WriteFile(filepath.Join(dir, n), nil, 0o644))
	}
	must(t, a.panels[1].Reload())
	a.HandleEvent(tcell.NewEventMouse(50, 5, tcell.WheelDown, 0))
	a.HandleEvent(tcell.NewEventMouse(50, 5, tcell.WheelDown, 0))
	a.HandleEvent(tcell.NewEventMouse(50, 5, tcell.WheelUp, 0))
	if a.active != 0 || a.panels[1].Cursor != 1 || a.panels[0].Cursor != 0 {
		t.Fatalf("active %d cursors %d %d", a.active, a.panels[0].Cursor, a.panels[1].Cursor)
	}
}

func TestKeyBarClick(t *testing.T) {
	a, _ := newApp(t)
	clickAt(a, 75, 24) // 10Quit
	if _, ok := a.modals.Top().(*ui.Dialog); !ok {
		t.Fatalf("top %#v", a.modals.Top())
	}
	x, y := findText(t, a, " Yes ")
	clickAt(a, x, y)
	if !a.quit {
		t.Fatal("click on Yes did not quit")
	}
}

// findText returns the cell where text starts on the screen.
func findText(t *testing.T, a *App, text string) (int, int) {
	t.Helper()
	for y := range 25 {
		if row := screenRow(a, y); strings.Contains(row, text) {
			return utf8.RuneCountInString(row[:strings.Index(row, text)]), y
		}
	}
	t.Fatalf("%q not on screen", text)
	return 0, 0
}
