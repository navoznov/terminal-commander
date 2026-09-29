package app

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/shell"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
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

// quietConsole makes commands run with /bin/sh and write into the returned
// buffer instead of the terminal, without waiting for a key.
func quietConsole(t *testing.T, a *App) *bytes.Buffer {
	t.Setenv("SHELL", "/bin/sh")
	out := &bytes.Buffer{}
	a.console = shell.Console{Out: out}
	return out
}

// run types line and presses Enter.
func run(a *App, line string) {
	typeText(a, line)
	press(a, tcell.KeyEnter, 0, 0)
}

func TestEnterRunsCommand(t *testing.T) {
	a, dir := newApp(t)
	a.home = dir
	out := quietConsole(t, a)
	run(a, "touch made.txt")
	if _, err := os.Stat(filepath.Join(dir, "made.txt")); err != nil {
		t.Fatal(err)
	}
	if !a.panels[0].Focus("made.txt") || !a.panels[1].Focus("made.txt") {
		t.Fatal("panels not re-read")
	}
	if want := "~>touch made.txt\nPress any key to continue...\n"; out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
	if a.cmd.Text != "" || !slices.Equal(a.cmd.History(), []string{"touch made.txt"}) {
		t.Fatalf("text %q history %q", a.cmd.Text, a.cmd.History())
	}
}

func TestCommandRunsInActivePanelDir(t *testing.T) {
	a, dir := newApp(t)
	out := quietConsole(t, a)
	must(t, a.panels[1].Load(filepath.Join(dir, "sub")))
	press(a, tcell.KeyTab, 0, 0)
	run(a, "pwd -P")
	real, err := filepath.EvalSymlinks(filepath.Join(dir, "sub"))
	must(t, err)
	if !strings.Contains(out.String(), "\n"+real+"\n") {
		t.Fatalf("output %q", out.String())
	}
}

func TestBlankLineEntersDirectory(t *testing.T) {
	a, dir := newApp(t)
	typeText(a, "x")
	press(a, tcell.KeyBackspace, 0, 0)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyEnter, 0, 0)
	if a.panels[0].Path != filepath.Join(dir, "sub") {
		t.Fatalf("path %s", a.panels[0].Path)
	}
}

func TestCd(t *testing.T) {
	a, dir := newApp(t)
	out := quietConsole(t, a)
	a.home = filepath.Join(dir, "sub")
	must(t, os.Mkdir(filepath.Join(dir, "My Dir"), 0o755))
	p := a.panels[0]
	steps := []struct{ line, want string }{
		{"cd sub", "sub"},
		{"cd ..", ""},
		{`cd "My Dir"`, "My Dir"},
		{"cd -", ""},
		{"cd -", "My Dir"},
		{"cd", "sub"},
		{"cd ~/", "sub"},
		{"cd " + dir, ""},
	}
	for _, st := range steps {
		run(a, st.line)
		if want := filepath.Join(dir, st.want); p.Path != want {
			t.Fatalf("%q: path %s, want %s", st.line, p.Path, want)
		}
	}
	if out.Len() != 0 {
		t.Fatalf("cd reached the shell: %q", out.String())
	}
	if h := a.cmd.History(); h[0] != "cd sub" {
		t.Fatalf("history %q", h)
	}
}

func TestCdErrorKeepsPanel(t *testing.T) {
	a, dir := newApp(t)
	quietConsole(t, a)
	run(a, "cd nope")
	d, ok := a.modals.Top().(*ui.Dialog)
	if a.panels[0].Path != dir || !ok || !d.Danger {
		t.Fatalf("path %s top %#v", a.panels[0].Path, a.modals.Top())
	}
}

func TestCdChainGoesToShell(t *testing.T) {
	a, dir := newApp(t)
	out := quietConsole(t, a)
	run(a, "cd sub && pwd -P")
	if a.panels[0].Path != dir || !strings.Contains(out.String(), "/sub\n") {
		t.Fatalf("path %s output %q", a.panels[0].Path, out.String())
	}
}

func TestHistoryKeys(t *testing.T) {
	a, _ := newApp(t)
	quietConsole(t, a)
	run(a, "cd sub")
	run(a, "cd ..")
	steps := []struct {
		k    tcell.Key
		want string
	}{
		{tcell.KeyCtrlE, "cd .."},
		{tcell.KeyCtrlE, "cd sub"},
		{tcell.KeyCtrlX, "cd .."},
		{tcell.KeyCtrlX, ""},
	}
	for i, st := range steps {
		press(a, st.k, 0, tcell.ModCtrl)
		if a.cmd.Text != st.want {
			t.Fatalf("step %d: text %q, want %q", i, a.cmd.Text, st.want)
		}
	}
}
