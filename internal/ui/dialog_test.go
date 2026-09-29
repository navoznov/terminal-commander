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

type result struct {
	button int
	text   string
	calls  int
}

func dialog(r *result, buttons ...string) *ui.Dialog {
	return &ui.Dialog{
		Title:   "Quit",
		Lines:   []string{"Do you want to quit?"},
		Buttons: buttons,
		Done:    func(b int, text string) { r.button, r.text = b, text; r.calls++ },
	}
}

func TestDialogEnterPressesFocused(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	d.HandleKey(key(tcell.KeyRight))
	if !d.HandleKey(key(tcell.KeyEnter)) || r.button != 1 || r.calls != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestDialogFocusWraps(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	d.HandleKey(key(tcell.KeyLeft))
	if d.Focus != 1 {
		t.Fatalf("focus %d", d.Focus)
	}
	d.HandleKey(key(tcell.KeyTab))
	if d.Focus != 0 {
		t.Fatalf("focus %d", d.Focus)
	}
}

func TestDialogEscCancels(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	if !d.HandleKey(key(tcell.KeyEscape)) || r.button != -1 {
		t.Fatalf("%+v", r)
	}
}

func TestDialogLetterPressesButton(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	if d.HandleKey(runeKey('x')) || r.calls != 0 {
		t.Fatal("x pressed a button")
	}
	if !d.HandleKey(runeKey('n')) || r.button != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestDialogInput(t *testing.T) {
	var r result
	d := dialog(&r, "OK", "Cancel")
	d.Input = ui.NewInput("*")
	for _, c := range "ab" {
		d.HandleKey(runeKey(c)) // first letter replaces "*"
	}
	d.HandleKey(key(tcell.KeyBackspace))
	d.HandleKey(runeKey('n')) // goes to the field, not to a button
	if !d.HandleKey(key(tcell.KeyEnter)) || r.button != 0 || r.text != "an" {
		t.Fatalf("%+v", r)
	}
}

func TestInputBackspaceEditsFreshText(t *testing.T) {
	in := ui.NewInput("abc")
	in.HandleKey(key(tcell.KeyBackspace))
	in.HandleKey(runeKey('d'))
	if in.Text != "abd" {
		t.Fatalf("got %q", in.Text)
	}
}

func TestInputIgnoresAltRunes(t *testing.T) {
	in := ui.NewInput("")
	if in.HandleKey(tcell.NewEventKey(tcell.KeyRune, '.', tcell.ModAlt)) || in.Text != "" {
		t.Fatalf("got %q", in.Text)
	}
}

func background(t *testing.T, w, h int) tcell.SimulationScreen {
	s := termtest.NewScreen(t, w, h)
	term.Canvas{Screen: s}.Fill(0, 0, w, h, '.', term.PanelStyle)
	return s
}

func TestDialogGolden(t *testing.T) {
	var r result
	s := background(t, 50, 10)
	dialog(&r, "Yes", "No").Draw(term.Canvas{Screen: s}, 50, 10)
	termtest.Golden(t, "dialog", termtest.Dump(s))
}

func TestDialogInputGolden(t *testing.T) {
	s := background(t, 60, 10)
	d := &ui.Dialog{Title: "Select", Input: ui.NewInput("*.go"), Buttons: []string{"OK", "Cancel"}}
	d.Draw(term.Canvas{Screen: s}, 60, 10)
	termtest.Golden(t, "dialog_input", termtest.Dump(s))
}

func TestDangerDialogIsRed(t *testing.T) {
	var r result
	s := background(t, 50, 10)
	d := dialog(&r, "OK")
	d.Danger = true
	d.Draw(term.Canvas{Screen: s}, 50, 10)
	styles := strings.SplitN(termtest.Dump(s), "\n\n", 2)[1]
	if !strings.Contains(styles, "r") || strings.Contains(styles, "g") {
		t.Fatalf("not red:\n%s", styles)
	}
}

func TestDialogOverCentersInSpan(t *testing.T) {
	var r result
	s := background(t, 80, 10)
	d := dialog(&r, "Yes", "No")
	d.Over = func(w int) (int, int) { return w / 2, w - w/2 }
	d.Draw(term.Canvas{Screen: s}, 80, 10)
	row := []rune(strings.Split(termtest.Dump(s), "\n")[3])
	// Window is 28 wide, centered in columns 40..79: x = 46, frame from 48.
	if row[48] != '╔' || row[47] != ' ' {
		t.Fatalf("row %q", string(row))
	}
}

func TestDialogOverStaysOnScreen(t *testing.T) {
	s := background(t, 80, 10)
	d := &ui.Dialog{
		Title:   "Drive",
		Lines:   []string{"Choose right drive:"},
		Buttons: []string{"/", "~", "Time Machine", "SanDisk Extr"},
		Over:    func(w int) (int, int) { return w / 2, w - w/2 },
	}
	d.Draw(term.Canvas{Screen: s}, 80, 10)
	out := termtest.Dump(s)
	row := []rune(strings.Split(out, "\n")[3])
	if i := strings.IndexRune(string(row), '╗'); i < 0 || len([]rune(string(row)[:i])) > 77 {
		t.Fatalf("frame cut off: %q", string(row))
	}
	if !strings.Contains(out, "SanDisk Extr") {
		t.Fatalf("button missing:\n%s", out)
	}
}

func TestDialogWrapsButtonRows(t *testing.T) {
	var buttons []string
	for i := 0; i < 12; i++ {
		buttons = append(buttons, fmt.Sprintf("Volume %05d", i))
	}
	s := background(t, 80, 14)
	d := &ui.Dialog{Title: "Drive", Buttons: buttons}
	d.Draw(term.Canvas{Screen: s}, 80, 14)
	out := termtest.Dump(s)
	for _, b := range buttons {
		if !strings.Contains(out, b) {
			t.Fatalf("button %q not shown:\n%s", b, out)
		}
	}
	for _, line := range strings.Split(strings.SplitN(out, "\n\n", 2)[0], "\n") {
		if r := strings.LastIndex(line, "║"); r >= 0 && strings.Contains(line[r:], "Volume") {
			t.Fatalf("button outside the frame: %q", line)
		}
	}
}
