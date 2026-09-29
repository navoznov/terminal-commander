package term_test

import (
	"strings"
	"testing"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

func TestBox(t *testing.T) {
	s := termtest.NewScreen(t, 6, 3)
	term.Canvas{Screen: s}.Box(0, 0, 6, 3, term.PanelStyle)
	lines := strings.Split(termtest.Dump(s), "\n")
	want := []string{"╔════╗", "║    ║", "╚════╝"}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d: got %q want %q", i, lines[i], w)
		}
	}
	if lines[4] != "pppppp" {
		t.Fatalf("styles: %q", lines[4])
	}
}

func TestTextClips(t *testing.T) {
	s := termtest.NewScreen(t, 10, 1)
	c := term.Canvas{Screen: s}
	if n := c.Text(0, 0, "hello", 3, term.PanelStyle); n != 3 {
		t.Fatalf("used %d", n)
	}
	if got := strings.Split(termtest.Dump(s), "\n")[0]; got != "hel       " {
		t.Fatalf("got %q", got)
	}
}

func TestFillAndLines(t *testing.T) {
	s := termtest.NewScreen(t, 4, 3)
	c := term.Canvas{Screen: s}
	c.Fill(0, 0, 4, 3, '.', term.CmdLineStyle)
	c.HLine(0, 1, 4, '─', term.PanelStyle)
	c.VLine(2, 0, 3, '│', term.PanelStyle)
	lines := strings.Split(termtest.Dump(s), "\n")
	want := []string{"..│.", "──│─", "..│."}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d: got %q want %q", i, lines[i], w)
		}
	}
}
