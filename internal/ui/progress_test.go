package ui_test

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestProgressKeysCancel(t *testing.T) {
	for _, ev := range []*tcell.EventKey{key(tcell.KeyEscape), key(tcell.KeyEnter), runeKey('c'), runeKey('C')} {
		n := 0
		p := &ui.Progress{Cancel: func() { n++ }}
		if p.HandleKey(ev) || n != 1 {
			t.Fatalf("%v: finished or cancels %d", ev.Name(), n)
		}
	}
	n := 0
	p := &ui.Progress{Cancel: func() { n++ }}
	p.HandleKey(runeKey('x'))
	if n != 0 {
		t.Fatal("x cancelled")
	}
}

func TestProgressHiddenIgnoresKeysAndDrawsNothing(t *testing.T) {
	n := 0
	p := &ui.Progress{Title: "Copy", Bars: []float64{0.5}, Hidden: true, Cancel: func() { n++ }}
	p.HandleKey(key(tcell.KeyEscape))
	s := termtest.NewScreen(t, 80, 25)
	p.Draw(term.Canvas{Screen: s}, 80, 25)
	if n != 0 || strings.Contains(termtest.Dump(s), "Copy") {
		t.Fatalf("cancels %d, drawn %v", n, strings.Contains(termtest.Dump(s), "Copy"))
	}
}

func TestProgressGolden(t *testing.T) {
	s := termtest.NewScreen(t, 80, 25)
	p := &ui.Progress{Title: "Copy", File: "autoexec.bat", Bars: []float64{0.5, 0.25}}
	p.Draw(term.Canvas{Screen: s}, 80, 25)
	termtest.Golden(t, "progress", termtest.Dump(s))
}
