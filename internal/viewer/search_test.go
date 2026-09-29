package viewer

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestFold(t *testing.T) {
	in := "AbC ПРИВЕТ Ёж \xff İx"
	got := fold([]byte(in))
	if len(got) != len(in) {
		t.Fatalf("length %d, want %d", len(got), len(in))
	}
	if want := "abc привет ёж \xff İx"; string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFind(t *testing.T) {
	l := newLayout("one Two three two TWO", 80, false, false)
	for _, tt := range []struct {
		pat  string
		from int64
		want int64
	}{
		{"two", 0, 4}, {"two", 5, 14}, {"TWO", 15, 18}, {"four", 0, -1}, {"", 0, -1},
	} {
		if got := l.find(tt.pat, tt.from); got != tt.want {
			t.Errorf("find(%q, %d) = %d, want %d", tt.pat, tt.from, got, tt.want)
		}
	}
}

func TestFindFold(t *testing.T) {
	l := newLayout("Ну, ПрИвЕт, мир", 80, false, false)
	if got := l.find("привет", 0); got != int64(len("Ну, ")) {
		t.Fatalf("got %d", got)
	}
}

func TestFindAcrossChunks(t *testing.T) {
	s := strings.Repeat("a", searchChunk-3) + "Найди" + strings.Repeat("b", 10)
	l := newLayout(s, 80, false, false)
	if got := l.find("НАЙДИ", 0); got != searchChunk-3 {
		t.Fatalf("got %d", got)
	}
}

// viewerWithStack opens a viewer whose windows go on a stack, like in the
// app.
func viewerWithStack(t *testing.T, s string) (*Viewer, *ui.Stack, func(k tcell.Key, r rune, m tcell.ModMask)) {
	v, scr := open(t, s)
	st := &ui.Stack{}
	st.Push(v)
	v.Push = st.Push
	press := func(k tcell.Key, r rune, m tcell.ModMask) {
		st.HandleKey(tcell.NewEventKey(k, r, m))
		st.Draw(term.Canvas{Screen: scr}, 80, 25)
	}
	return v, st, press
}

func typeIn(press func(tcell.Key, rune, tcell.ModMask), s string) {
	for _, r := range s {
		press(tcell.KeyRune, r, 0)
	}
}

func TestSearchKeys(t *testing.T) {
	v, st, press := viewerWithStack(t, numbered(100)+"Needle 1\n"+numbered(3)+"needle 2\n")
	press(tcell.KeyF7, 0, 0)
	if _, ok := st.Top().(*ui.Dialog); !ok {
		t.Fatalf("top %#v", st.Top())
	}
	typeIn(press, "NEEDLE")
	press(tcell.KeyEnter, 0, 0)
	first := v.match
	if st.Top() != v || v.top != first || v.matchEnd != first+6 {
		t.Fatalf("top %d match %d-%d", v.top, v.match, v.matchEnd)
	}
	press(tcell.KeyRune, 'n', 0)
	if v.match <= first || v.top != v.rowStart(v.match) {
		t.Fatalf("n: match %d top %d", v.match, v.top)
	}
	press(tcell.KeyF7, 0, tcell.ModShift)
	d, ok := st.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != "String not found" {
		t.Fatalf("top %#v", st.Top())
	}
	press(tcell.KeyEnter, 0, 0)
	if st.Top() != v {
		t.Fatal("dialog not closed")
	}
}

func TestSearchScrollsRight(t *testing.T) {
	v, _, press := viewerWithStack(t, strings.Repeat("x", 300)+"found\n")
	v.pattern = "found"
	press(tcell.KeyF19, 0, 0)
	if v.col == 0 || v.col > 300 || 300-v.col >= 80 {
		t.Fatalf("col %d", v.col)
	}
}

func TestMatchHighlighted(t *testing.T) {
	v, scr := open(t, "abc def\n")
	v.pattern = "DEF"
	v.search(0)
	draw(v, scr)
	for x := 0; x < 8; x++ {
		_, _, st, _ := scr.GetContent(x, 1)
		if want := x >= 4 && x < 7; (st == term.CursorStyle) != want {
			t.Errorf("x %d highlighted %v", x, !want)
		}
	}
}
