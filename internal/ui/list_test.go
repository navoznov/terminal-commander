package ui_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestListPick(t *testing.T) {
	got := -2
	l := &ui.List{Title: "History", Items: []string{"a", "b", "c"}, Cur: 2, Done: func(i int) { got = i }}
	l.Draw(term.Canvas{Screen: background(t, 60, 12)}, 60, 12)
	for _, k := range []tcell.Key{tcell.KeyUp, tcell.KeyUp, tcell.KeyUp, tcell.KeyDown} {
		if l.HandleKey(key(k)) {
			t.Fatal("list closed on a move")
		}
	}
	if !l.HandleKey(key(tcell.KeyEnter)) || got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestListEscAndEmpty(t *testing.T) {
	got := -2
	l := &ui.List{Items: []string{"a"}, Done: func(i int) { got = i }}
	if !l.HandleKey(key(tcell.KeyEscape)) || got != -1 {
		t.Fatalf("Esc gave %d", got)
	}
	got = -2
	empty := &ui.List{Title: "History", Cur: -1, Done: func(i int) { got = i }}
	empty.Draw(term.Canvas{Screen: background(t, 60, 12)}, 60, 12) // must not panic
	empty.HandleKey(key(tcell.KeyDown))
	if !empty.HandleKey(key(tcell.KeyEnter)) || got != -1 {
		t.Fatalf("Enter on empty list gave %d", got)
	}
}

func TestListScrollsToCursor(t *testing.T) {
	var items []string
	for i := 0; i < 30; i++ {
		items = append(items, fmt.Sprintf("item %d", i))
	}
	s := background(t, 60, 12) // 12-6 = 6 visible rows
	l := &ui.List{Title: "History", Items: items, Cur: 29}
	l.Draw(term.Canvas{Screen: s}, 60, 12)
	if l.Top != 24 {
		t.Fatalf("top %d", l.Top)
	}
	lines := strings.Split(termtest.Dump(s), "\n")
	for y, line := range lines[:12] {
		if i := strings.Index(line, "item 29"); i >= 0 {
			x := utf8.RuneCountInString(line[:i])
			if _, _, st, _ := s.GetContent(x, y); st != term.CursorStyle {
				t.Fatal("cursor row not highlighted")
			}
		}
	}
	l.HandleKey(key(tcell.KeyHome))
	l.Draw(term.Canvas{Screen: s}, 60, 12)
	if l.Cur != 0 || l.Top != 0 {
		t.Fatalf("cur %d top %d", l.Cur, l.Top)
	}
}
