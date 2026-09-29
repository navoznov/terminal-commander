package ui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func helpView(t *testing.T) (*ui.TextView, tcell.SimulationScreen) {
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	s := background(t, 60, 12)
	v := &ui.TextView{Title: "Help", Lines: lines}
	v.Draw(term.Canvas{Screen: s}, 60, 12) // 12-6 = 6 visible rows
	return v, s
}

func TestTextViewScrollClamps(t *testing.T) {
	v, _ := helpView(t)
	steps := []struct {
		k    tcell.Key
		want int
	}{
		{tcell.KeyUp, 0},
		{tcell.KeyDown, 1},
		{tcell.KeyPgDn, 7},
		{tcell.KeyEnd, 24},
		{tcell.KeyDown, 24},
		{tcell.KeyPgUp, 18},
		{tcell.KeyHome, 0},
	}
	for _, st := range steps {
		if v.HandleKey(key(st.k)) {
			t.Fatalf("%v closed the view", st.k)
		}
		if v.Top != st.want {
			t.Fatalf("after %v: top %d want %d", st.k, v.Top, st.want)
		}
	}
}

func TestTextViewCloses(t *testing.T) {
	for _, k := range []tcell.Key{tcell.KeyEscape, tcell.KeyEnter, tcell.KeyF1, tcell.KeyF10} {
		v, _ := helpView(t)
		if !v.HandleKey(key(k)) {
			t.Fatalf("%v did not close", k)
		}
	}
}

func TestTextViewDraw(t *testing.T) {
	v, s := helpView(t)
	v.HandleKey(key(tcell.KeyDown))
	v.Draw(term.Canvas{Screen: s}, 60, 12)
	out := termtest.Dump(s)
	if !strings.Contains(out, " Help ") || !strings.Contains(out, "line 6") || strings.Contains(out, "line 0 ") || strings.Contains(out, "line 7") {
		t.Fatalf("unexpected view:\n%s", out)
	}
}
