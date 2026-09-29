package ui_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

type fakeView struct{ onKey func() bool }

func (f *fakeView) Draw(term.Canvas, int, int)     {}
func (f *fakeView) HandleKey(*tcell.EventKey) bool { return f.onKey() }

func key(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, 0) }

func runeKey(r rune) *tcell.EventKey { return tcell.NewEventKey(tcell.KeyRune, r, 0) }

func TestStackRoutesToTop(t *testing.T) {
	var s ui.Stack
	var got []string
	s.Push(&fakeView{onKey: func() bool { got = append(got, "bottom"); return false }})
	s.Push(&fakeView{onKey: func() bool { got = append(got, "top"); return true }})
	s.HandleKey(key(tcell.KeyEnter))
	s.HandleKey(key(tcell.KeyEnter))
	if len(got) != 2 || got[0] != "top" || got[1] != "bottom" || s.Len() != 1 {
		t.Fatalf("got %v, len %d", got, s.Len())
	}
}

func TestStackKeepsViewPushedByFinishingOne(t *testing.T) {
	var s ui.Stack
	next := &fakeView{onKey: func() bool { return false }}
	s.Push(&fakeView{onKey: func() bool { s.Push(next); return true }})
	s.HandleKey(key(tcell.KeyEnter))
	if s.Len() != 1 || s.Top() != next {
		t.Fatalf("len %d, top %v", s.Len(), s.Top())
	}
}

func TestEmptyStack(t *testing.T) {
	var s ui.Stack
	s.HandleKey(key(tcell.KeyEnter)) // must not panic
	if !s.Empty() || s.Top() != nil {
		t.Fatal("stack not empty")
	}
}

func TestStackRemove(t *testing.T) {
	var s ui.Stack
	a := &fakeView{onKey: func() bool { return false }}
	b := &fakeView{onKey: func() bool { return false }}
	s.Push(a)
	s.Push(b)
	s.Remove(a)
	if s.Len() != 1 || s.Top() != b {
		t.Fatalf("len %d top %v", s.Len(), s.Top())
	}
	s.Remove(a) // not in the stack: no-op
	if s.Len() != 1 {
		t.Fatalf("len %d", s.Len())
	}
}
