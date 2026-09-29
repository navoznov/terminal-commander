package app

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

func TestTypingGoesToCommandLine(t *testing.T) {
	a, dir := newApp(t)
	typeText(a, "ls -la")
	press(a, tcell.KeyBackspace, 0, 0)
	if a.cmd.Text != "ls -l" || a.panels[0].Path != dir {
		t.Fatalf("text %q path %s", a.cmd.Text, a.panels[0].Path)
	}
}

func TestSpacePlusMinusStarGoToNonEmptyLine(t *testing.T) {
	a, _ := newApp(t)
	typeText(a, "x +-* ")
	if a.cmd.Text != "x +-* " || len(a.panels[0].Selected) != 0 || !a.modals.Empty() {
		t.Fatalf("text %q selected %v modals %d", a.cmd.Text, a.panels[0].Selected, a.modals.Len())
	}
}

func TestEscClearsLineWithoutAltPrefix(t *testing.T) {
	a, _ := newApp(t)
	typeText(a, "ls")
	press(a, tcell.KeyEscape, 0, 0)
	if a.cmd.Text != "" {
		t.Fatalf("text %q after Esc", a.cmd.Text)
	}
	press(a, tcell.KeyRune, '.', 0)
	if a.cmd.Text != "." || a.showHidden {
		t.Fatalf("text %q hidden %v", a.cmd.Text, a.showHidden)
	}
}

func TestArrowsMovePanelWhileTyping(t *testing.T) {
	a, _ := newApp(t)
	typeText(a, "ls")
	before := a.panels[0].Cursor
	press(a, tcell.KeyDown, 0, 0)
	if a.panels[0].Cursor == before || a.cmd.Text != "ls" {
		t.Fatalf("cursor %d text %q", a.panels[0].Cursor, a.cmd.Text)
	}
}

// cmdRow returns the command line row (y = 23 on the 80×25 test screen).
func cmdRow(a *App) string {
	return strings.Split(termtest.Dump(a.screen.(tcell.SimulationScreen)), "\n")[23]
}

func TestCommandLineShowsText(t *testing.T) {
	a, dir := newApp(t)
	a.home = dir
	typeText(a, "ls")
	if row := cmdRow(a); !strings.HasPrefix(row, "~>ls ") {
		t.Fatalf("row %q", row)
	}
	if x, y, _ := a.screen.(tcell.SimulationScreen).GetCursor(); x != 4 || y != 23 {
		t.Fatalf("cursor at %d,%d", x, y)
	}
}

func TestLongCommandLineShowsTail(t *testing.T) {
	a, dir := newApp(t)
	a.home = dir
	long := strings.Repeat("abcdefghij", 10)
	typeText(a, long)
	if row := cmdRow(a); row != long[len(long)-79:]+" " {
		t.Fatalf("row %q", row)
	}
	if x, _, _ := a.screen.(tcell.SimulationScreen).GetCursor(); x != 79 {
		t.Fatalf("cursor x %d", x)
	}
}
