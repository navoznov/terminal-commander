package ui_test

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func testMenus(log *[]string) []ui.Menu {
	act := func(s string) func() { return func() { *log = append(*log, s) } }
	return []ui.Menu{
		{Title: "Left", Items: []ui.Item{
			{Label: "Brief", Checked: true, Action: act("brief")},
			{Label: "Full", Action: act("full")},
			{},
			{Label: "Re-read", Key: "Control-R", Action: act("reread")},
		}},
		{Title: "Files", Items: []ui.Item{{Label: "Help", Key: "F1", Action: act("help")}}},
		{Title: "Right", Items: []ui.Item{
			{Label: "Brief", Action: act("rbrief")},
			{},
			{Label: "Full", Action: act("rfull")},
		}},
	}
}

func TestMenuUpDownSkipSeparatorsAndWrap(t *testing.T) {
	var log []string
	m := ui.NewMenuBar(testMenus(&log), 0)
	for _, want := range []int{1, 3, 0} {
		m.HandleKey(key(tcell.KeyDown))
		if m.Sel != want {
			t.Fatalf("down: sel %d want %d", m.Sel, want)
		}
	}
	m.HandleKey(key(tcell.KeyUp))
	if m.Sel != 3 {
		t.Fatalf("up: sel %d", m.Sel)
	}
	m.HandleKey(key(tcell.KeyHome))
	m.HandleKey(key(tcell.KeyEnd))
	if m.Sel != 3 {
		t.Fatalf("end: sel %d", m.Sel)
	}
}

func TestMenuLeftRightSwitchMenus(t *testing.T) {
	var log []string
	m := ui.NewMenuBar(testMenus(&log), 0)
	m.HandleKey(key(tcell.KeyDown))
	for _, want := range []int{1, 2, 0} {
		m.HandleKey(key(tcell.KeyRight))
		if m.Cur != want || m.Sel != 0 {
			t.Fatalf("right: cur %d sel %d want %d", m.Cur, m.Sel, want)
		}
	}
	m.HandleKey(key(tcell.KeyLeft))
	if m.Cur != 2 {
		t.Fatalf("left: cur %d", m.Cur)
	}
}

func TestMenuEnterRunsAction(t *testing.T) {
	var log []string
	m := ui.NewMenuBar(testMenus(&log), 2)
	m.HandleKey(key(tcell.KeyDown))
	if !m.HandleKey(key(tcell.KeyEnter)) || len(log) != 1 || log[0] != "rfull" {
		t.Fatalf("log %v", log)
	}
}

func TestMenuEscClosesWithoutAction(t *testing.T) {
	var log []string
	for _, k := range []tcell.Key{tcell.KeyEscape, tcell.KeyF9, tcell.KeyF10} {
		m := ui.NewMenuBar(testMenus(&log), 0)
		if !m.HandleKey(key(k)) {
			t.Fatalf("%v did not close", k)
		}
	}
	if len(log) != 0 {
		t.Fatalf("log %v", log)
	}
}

func TestMenuGolden(t *testing.T) {
	var log []string
	s := background(t, 50, 10)
	m := ui.NewMenuBar(testMenus(&log), 0)
	m.HandleKey(key(tcell.KeyDown))
	m.Draw(term.Canvas{Screen: s}, 50, 10)
	termtest.Golden(t, "menu", termtest.Dump(s))
}

func TestMenuDropdownStaysOnScreen(t *testing.T) {
	var log []string
	s := background(t, 24, 10)
	ui.NewMenuBar(testMenus(&log), 2).Draw(term.Canvas{Screen: s}, 24, 10)
	row := []rune(strings.Split(termtest.Dump(s), "\n")[1])
	if !strings.ContainsRune(string(row), '╗') {
		t.Fatalf("dropdown cut off: %q", string(row))
	}
}
