package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/config"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestNewRestoresSetup(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, ".dot"), nil, 0o644))
	cfg := config.Config{
		Left:       config.Panel{Path: dir, Mode: "full", Sort: "size"},
		Right:      config.Panel{Path: "/", Mode: "bogus", Sort: "time"},
		ShowHidden: true,
		History:    []string{"ls", "pwd"},
	}
	a := New(termtest.NewScreen(t, 80, 25), cfg, "")
	l, r := a.panels[0], a.panels[1]
	if l.Path != dir || l.Mode != panel.Full || l.Sort != panel.SortSize || !l.Focus(".dot") {
		t.Fatalf("left %s mode %v sort %v", l.Path, l.Mode, l.Sort)
	}
	if r.Path != "/" || r.Mode != panel.Brief || r.Sort != panel.SortTime {
		t.Fatalf("right %s mode %v sort %v", r.Path, r.Mode, r.Sort)
	}
	if !a.showHidden || !slices.Equal(a.cmd.History(), cfg.History) {
		t.Fatalf("hidden %v history %q", a.showHidden, a.cmd.History())
	}
}

func TestQuitSavesSetup(t *testing.T) {
	a, dir := newApp(t)
	a.cfgPath = filepath.Join(t.TempDir(), "config.json")
	must(t, a.panels[1].Load(filepath.Join(dir, "sub")))
	a.panels[1].SetMode(panel.Full)
	a.cmd.Text = "ls"
	a.cmd.Commit()
	press(a, tcell.KeyF10, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	want := config.Config{
		Left:    config.Panel{Path: dir, Mode: "brief", Sort: "name"},
		Right:   config.Panel{Path: filepath.Join(dir, "sub"), Mode: "full", Sort: "name"},
		History: []string{"ls"},
	}
	if got := config.Load(a.cfgPath); !a.quit || !slices.Equal(got.History, want.History) || got.Left != want.Left || got.Right != want.Right {
		t.Fatalf("quit %v config %+v", a.quit, got)
	}
}

func TestQuitAfterSaveError(t *testing.T) {
	a, dir := newApp(t)
	a.cfgPath = filepath.Join(dir, ".dot", "config.json") // .dot is a file
	press(a, tcell.KeyF10, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if a.quit || !ok || !d.Danger {
		t.Fatalf("quit %v top %#v", a.quit, a.modals.Top())
	}
	press(a, tcell.KeyEnter, 0, 0)
	if !a.quit {
		t.Fatal("did not quit after the error")
	}
}

func TestSaveSetup(t *testing.T) {
	a, _ := newApp(t)
	a.cfgPath = filepath.Join(t.TempDir(), "config.json")
	press(a, tcell.KeyRune, '.', tcell.ModAlt)
	must(t, a.saveSetup())
	if !config.Load(a.cfgPath).ShowHidden || a.quit {
		t.Fatal("setup not saved")
	}
}
