package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func press(a *App, k tcell.Key, r rune, m tcell.ModMask) {
	a.HandleEvent(tcell.NewEventKey(k, r, m))
	a.Draw()
}

func newApp(t *testing.T) (*App, string) {
	dir := t.TempDir()
	must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "sub", "f.txt"), nil, 0o644))
	must(t, os.WriteFile(filepath.Join(dir, ".dot"), nil, 0o644))
	s := termtest.NewScreen(t, 80, 25)
	a := New(s, dir, dir)
	a.Draw()
	return a, dir
}

func TestTabSwitchesPanel(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyTab, 0, 0)
	if a.active != 1 {
		t.Fatalf("active %d", a.active)
	}
}

func TestEnterAndBackspace(t *testing.T) {
	a, dir := newApp(t)
	p := a.panels[0]
	p.Focus("sub")
	press(a, tcell.KeyEnter, 0, 0)
	if p.Path != filepath.Join(dir, "sub") {
		t.Fatalf("path %s", p.Path)
	}
	press(a, tcell.KeyBackspace, 0, 0)
	if p.Path != dir || p.Current().Name != "sub" {
		t.Fatalf("path %s current %+v", p.Path, p.Current())
	}
}

func TestCtrlPgUpGoesUp(t *testing.T) {
	a, dir := newApp(t)
	must(t, a.panels[0].Load(filepath.Join(dir, "sub")))
	press(a, tcell.KeyPgUp, 0, tcell.ModCtrl)
	if a.panels[0].Path != dir {
		t.Fatalf("path %s", a.panels[0].Path)
	}
}

func TestAltDotTogglesHiddenInBothPanels(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyRune, '.', tcell.ModAlt)
	for i, p := range a.panels {
		if !p.Focus(".dot") {
			t.Fatalf("panel %d: .dot not shown", i)
		}
	}
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyRune, '.', 0)
	if a.panels[0].Focus(".dot") {
		t.Fatal("Esc . did not hide .dot")
	}
}

func TestSpaceSelects(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyRune, ' ', 0)
	if !a.panels[0].Selected["sub"] {
		t.Fatal("sub not selected")
	}
}

func TestCtrlUSwapsPanels(t *testing.T) {
	a, dir := newApp(t)
	must(t, a.panels[0].Load(filepath.Join(dir, "sub")))
	press(a, tcell.KeyCtrlU, 0, 0)
	if a.panels[1].Path != filepath.Join(dir, "sub") || a.active != 1 {
		t.Fatalf("right %s active %d", a.panels[1].Path, a.active)
	}
}

func TestEscZeroQuits(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyRune, '0', 0)
	if !a.quit {
		t.Fatal("Esc 0 did not quit")
	}
}

func TestEnterUnreadableDirShowsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	a, dir := newApp(t)
	locked := filepath.Join(dir, "locked")
	must(t, os.Mkdir(locked, 0o000))
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	p := a.panels[0]
	must(t, p.Reload())
	p.Focus("locked")
	press(a, tcell.KeyEnter, 0, 0)
	if p.Path != dir || a.errMsg == "" {
		t.Fatalf("path %s err %q", p.Path, a.errMsg)
	}
	if !strings.Contains(a.errMsg, "permission denied") {
		t.Fatalf("err %q", a.errMsg)
	}
	if !strings.Contains(termtest.Dump(a.screen.(tcell.SimulationScreen)), "open /") {
		t.Fatal("error not shown in the command line")
	}
	press(a, tcell.KeyDown, 0, 0)
	if a.errMsg != "" {
		t.Fatal("error not cleared by next key")
	}
}

func TestSmallWindow(t *testing.T) {
	a, _ := newApp(t)
	s := a.screen.(tcell.SimulationScreen)
	s.SetSize(40, 10)
	a.HandleEvent(tcell.NewEventResize(40, 10))
	a.Draw()
	if !strings.Contains(termtest.Dump(s), "Window too small") {
		t.Fatal("no small-window message")
	}
	press(a, tcell.KeyDown, 0, 0) // must not panic
}

func TestScreenGolden(t *testing.T) {
	a, _ := newApp(t)
	a.home = "/Users/nc"
	day := time.Date(1994, 5, 31, 6, 22, 0, 0, time.Local)
	left, right := a.panels[0], a.panels[1]
	left.Path, right.Path = "/Users/nc", "/Users/nc/GAMES"
	left.Entries = []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true, ModTime: day},
		{Name: "DOS", IsDir: true, ModTime: day},
		{Name: "GAMES", IsDir: true, ModTime: day},
		{Name: "autoexec.bat", Size: 13, ModTime: day},
		{Name: "config.sys", Size: 13, ModTime: day},
	}
	right.Entries = []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true, ModTime: day},
		{Name: "CIV", IsDir: true, ModTime: day},
		{Name: "tetris.exe", Size: 42001, ModTime: day},
	}
	left.Cursor, right.Cursor = 4, 1
	right.SetMode(panel.Full)
	a.Draw()
	termtest.Golden(t, "screen", termtest.Dump(a.screen.(tcell.SimulationScreen)))
}

func TestCtrlTTogglesMode(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyCtrlT, 0, 0)
	if a.panels[0].Mode != panel.Full || a.panels[1].Mode != panel.Brief {
		t.Fatalf("modes %v %v", a.panels[0].Mode, a.panels[1].Mode)
	}
	press(a, tcell.KeyCtrlT, 0, 0)
	if a.panels[0].Mode != panel.Brief {
		t.Fatalf("mode %v", a.panels[0].Mode)
	}
}

func TestEscYuTogglesHiddenOnRussianLayout(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyRune, 'ю', 0)
	if !a.panels[0].Focus(".dot") {
		t.Fatal("Esc ю did not show .dot")
	}
}
