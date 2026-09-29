package ui_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestClicker(t *testing.T) {
	t0 := time.Unix(1000, 0)
	var c ui.Clicker
	steps := []struct {
		x, y    int
		buttons tcell.ButtonMask
		after   time.Duration
		want    ui.MouseAction // 0: no action
	}{
		{1, 1, tcell.Button1, 0, ui.Click},
		{1, 1, tcell.Button1, 10 * time.Millisecond, 0}, // still held (drag)
		{1, 1, 0, 50 * time.Millisecond, 0},             // released
		{1, 1, tcell.Button1, 200 * time.Millisecond, ui.DoubleClick},
		{1, 1, 0, 250 * time.Millisecond, 0},
		{1, 1, tcell.Button1, 300 * time.Millisecond, ui.Click}, // a third click starts over
		{1, 1, 0, 310 * time.Millisecond, 0},
		{2, 1, tcell.Button1, 400 * time.Millisecond, ui.Click}, // another cell
		{2, 1, 0, 410 * time.Millisecond, 0},
		{2, 1, tcell.Button1, 900 * time.Millisecond, ui.Click}, // too late
		{2, 1, 0, 910 * time.Millisecond, 0},
		{5, 6, tcell.WheelUp, time.Second, ui.WheelUp},
		{5, 6, tcell.WheelDown, time.Second, ui.WheelDown},
		{5, 6, tcell.Button2, time.Second, 0},
	}
	for i, st := range steps {
		m, ok := c.Feed(tcell.NewEventMouse(st.x, st.y, st.buttons, 0), t0.Add(st.after))
		if ok != (st.want != 0) || m.Action != st.want || (ok && (m.X != st.x || m.Y != st.y)) {
			t.Fatalf("step %d: got %+v %v, want action %d", i, m, ok, st.want)
		}
	}
}

// find returns the cell where s starts on the screen.
func find(t *testing.T, s tcell.SimulationScreen, text string) (int, int) {
	t.Helper()
	for y, line := range strings.Split(termtest.Dump(s), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return len([]rune(line[:i])), y
		}
	}
	t.Fatalf("%q not on screen", text)
	return 0, 0
}

func click(x, y int) ui.Mouse { return ui.Mouse{X: x, Y: y, Action: ui.Click} }

func TestDialogClickButton(t *testing.T) {
	r := result{button: -2}
	s := background(t, 50, 10)
	d := dialog(&r, "Yes", "No")
	d.Draw(term.Canvas{Screen: s}, 50, 10)
	if d.HandleMouse(click(0, 0)) || r.button != -2 {
		t.Fatalf("click outside: button %d", r.button)
	}
	x, y := find(t, s, " No ")
	if !d.HandleMouse(click(x+2, y)) || r.button != 1 {
		t.Fatalf("button %d", r.button)
	}
}

func TestMenuMouse(t *testing.T) {
	var log []string
	s := background(t, 60, 12)
	m := ui.NewMenuBar(testMenus(&log), 0)
	m.Draw(term.Canvas{Screen: s}, 60, 12)
	x, y := find(t, s, "Right")
	if m.HandleMouse(click(x, y)) || m.Cur != 2 {
		t.Fatalf("title click: cur %d", m.Cur)
	}
	s = background(t, 60, 12)
	m.Draw(term.Canvas{Screen: s}, 60, 12)
	x, y = find(t, s, "Full")
	if !m.HandleMouse(click(x, y)) || strings.Join(log, ",") != "rfull" {
		t.Fatalf("item click: log %v", log)
	}
	m = ui.NewMenuBar(testMenus(&log), 0)
	s = background(t, 60, 12)
	m.Draw(term.Canvas{Screen: s}, 60, 12)
	x, y = find(t, s, "Re-read")
	if m.HandleMouse(click(x, y-1)) { // the separator above it
		t.Fatal("separator closed the menu")
	}
	if !m.HandleMouse(click(50, 10)) || len(log) != 1 {
		t.Fatalf("outside click: log %v", log)
	}
}

func TestProgressClickCancel(t *testing.T) {
	n := 0
	s := background(t, 60, 12)
	p := &ui.Progress{Title: "Copy", Bars: []float64{0.5}, Cancel: func() { n++ }}
	p.Draw(term.Canvas{Screen: s}, 60, 12)
	x, y := find(t, s, "Cancel")
	if p.HandleMouse(click(x, y)) || n != 1 {
		t.Fatalf("cancels %d", n)
	}
}

func TestStackHandleMouse(t *testing.T) {
	var r result
	var st ui.Stack
	s := background(t, 50, 10)
	st.Push(&ui.TextView{Lines: []string{"a"}}) // takes no mouse
	st.HandleMouse(click(1, 1))
	d := dialog(&r, "OK")
	st.Push(d)
	st.Draw(term.Canvas{Screen: s}, 50, 10)
	x, y := find(t, s, " OK ")
	st.HandleMouse(click(x, y))
	if st.Len() != 1 || r.button != 0 {
		t.Fatalf("len %d button %d", st.Len(), r.button)
	}
}
