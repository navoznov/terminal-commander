package viewer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func key(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, 0) }

// open returns a viewer over s drawn on an 80×25 screen.
func open(t *testing.T, s string) (*Viewer, tcell.SimulationScreen) {
	scr := termtest.NewScreen(t, 80, 25)
	v := New("/tmp/file.txt", strings.NewReader(s), int64(len(s)))
	draw(v, scr)
	return v, scr
}

func draw(v *Viewer, scr tcell.SimulationScreen) {
	v.Draw(term.Canvas{Screen: scr}, 80, 25)
}

// press sends keys and redraws after each, as the app does.
func press(v *Viewer, scr tcell.SimulationScreen, keys ...tcell.Key) {
	for _, k := range keys {
		v.HandleKey(key(k))
		draw(v, scr)
	}
}

// numbered returns lines "line 0" … "line n-1".
func numbered(n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	return b.String()
}

// screenRow returns row y of the screen text.
func screenRow(scr tcell.SimulationScreen, y int) string {
	return strings.Split(termtest.Dump(scr), "\n")[y]
}

func TestViewerGolden(t *testing.T) {
	text := "Hello, мир!\n\tindented\r\nbad \xff byte\n" + strings.Repeat("long ", 20) + "\n"
	v, scr := open(t, text)
	termtest.Golden(t, "text", termtest.Dump(scr))
	press(v, scr, tcell.KeyF2)
	termtest.Golden(t, "wrap", termtest.Dump(scr))
	press(v, scr, tcell.KeyF4)
	termtest.Golden(t, "hex", termtest.Dump(scr))
}

func TestScrolling(t *testing.T) {
	v, scr := open(t, numbered(100))
	steps := []struct {
		k     tcell.Key
		first string
	}{
		{tcell.KeyDown, "line 1"},
		{tcell.KeyPgDn, "line 24"},
		{tcell.KeyUp, "line 23"},
		{tcell.KeyPgUp, "line 0"},
		{tcell.KeyEnd, "line 77"},
		{tcell.KeyDown, "line 77"},
		{tcell.KeyPgDn, "line 77"},
		{tcell.KeyHome, "line 0"},
	}
	for i, st := range steps {
		press(v, scr, st.k)
		if got := strings.TrimSpace(screenRow(scr, 1)); got != st.first {
			t.Fatalf("step %d: first row %q, want %q", i, got, st.first)
		}
	}
}

func TestShortFileDoesNotScroll(t *testing.T) {
	v, scr := open(t, "a\nb\n")
	press(v, scr, tcell.KeyDown, tcell.KeyPgDn, tcell.KeyEnd)
	if v.top != 0 {
		t.Fatalf("top %d", v.top)
	}
	if !strings.Contains(screenRow(scr, 0), "4 bytes  100%") {
		t.Fatalf("title %q", screenRow(scr, 0))
	}
}

func TestEmptyFile(t *testing.T) {
	v, scr := open(t, "")
	press(v, scr, tcell.KeyDown, tcell.KeyEnd, tcell.KeyUp, tcell.KeyF4, tcell.KeyEnd)
	if v.top != 0 {
		t.Fatalf("top %d", v.top)
	}
}

func TestHorizontalScroll(t *testing.T) {
	v, scr := open(t, "0123456789\n")
	press(v, scr, tcell.KeyRight, tcell.KeyRight, tcell.KeyRight, tcell.KeyLeft)
	if got := strings.TrimSpace(screenRow(scr, 1)); got != "23456789" {
		t.Fatalf("row %q", got)
	}
	press(v, scr, tcell.KeyF2) // wrapping shows lines from their start
	if got := strings.TrimSpace(screenRow(scr, 1)); got != "0123456789" {
		t.Fatalf("row %q", got)
	}
}

func TestModesKeepPosition(t *testing.T) {
	v, scr := open(t, strings.Repeat("x", 200)+"\n"+numbered(50))
	press(v, scr, tcell.KeyDown, tcell.KeyDown)
	press(v, scr, tcell.KeyF4)
	if v.top != 208 { // "line 1" starts at 208
		t.Fatalf("hex top %d", v.top)
	}
	press(v, scr, tcell.KeyF4)
	if got := strings.TrimSpace(screenRow(scr, 1)); got != "line 1" {
		t.Fatalf("back to text: first row %q", got)
	}
	press(v, scr, tcell.KeyHome, tcell.KeyF2, tcell.KeyDown, tcell.KeyDown, tcell.KeyDown)
	if got := strings.TrimSpace(screenRow(scr, 1)); got != "line 0" {
		t.Fatalf("wrapped: first row %q", got)
	}
}

func TestCloseKeys(t *testing.T) {
	for _, k := range []tcell.Key{tcell.KeyEscape, tcell.KeyF10, tcell.KeyF3} {
		closed := false
		v := New("f", strings.NewReader("x"), 1)
		v.Close = func() { closed = true }
		if !v.HandleKey(key(k)) || !closed {
			t.Errorf("%v did not close the viewer", k)
		}
	}
}

// A file that is one huge line is shown without reading it whole.
func TestHugeLine(t *testing.T) {
	s := strings.Repeat("0123456789", 5_000_000)
	v, scr := open(t, s)
	press(v, scr, tcell.KeyEnd, tcell.KeyUp, tcell.KeyF2, tcell.KeyEnd, tcell.KeyPgUp)
	if v.top <= 0 || v.top >= int64(len(s)) {
		t.Fatalf("top %d", v.top)
	}
}

func TestMouse(t *testing.T) {
	v, scr := open(t, numbered(100))
	v.HandleMouse(ui.Mouse{X: 5, Y: 5, Action: ui.WheelDown})
	v.HandleMouse(ui.Mouse{X: 5, Y: 5, Action: ui.WheelDown})
	v.HandleMouse(ui.Mouse{X: 5, Y: 5, Action: ui.WheelUp})
	draw(v, scr)
	if got := strings.TrimSpace(screenRow(scr, 1)); got != "line 1" {
		t.Fatalf("first row %q", got)
	}
	v.HandleMouse(ui.Mouse{X: 10, Y: 24, Action: ui.Click}) // 2Wrap
	if !v.wrap {
		t.Fatal("key bar click did not wrap")
	}
	closed := false
	v.Close = func() { closed = true }
	if !v.HandleMouse(ui.Mouse{X: 75, Y: 24, Action: ui.Click}) || !closed { // 10Quit
		t.Fatal("Quit click did not close")
	}
}
