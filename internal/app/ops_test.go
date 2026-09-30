package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ops"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// settle runs the calls posted by the running operation until it ends or
// shows a question dialog.
func settle(t *testing.T, a *App) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for a.op != nil {
		if _, ok := a.modals.Top().(*ui.Dialog); ok {
			return
		}
		select {
		case f := <-a.calls:
			f()
			a.Draw()
		case <-deadline:
			t.Fatal("operation did not finish")
		}
	}
}

// fakeTrash makes F8 delete with rm instead of filling the real Trash.
func fakeTrash(t *testing.T) {
	ops.TrashCmd = "/bin/rm"
	t.Cleanup(func() { ops.TrashCmd = "/usr/bin/trash" })
}

func TestSubject(t *testing.T) {
	if got := subject([]string{"/a/x.txt"}); got != `"x.txt"` {
		t.Fatalf("one: %s", got)
	}
	if got := subject([]string{"/a/x", "/a/y"}); got != "2 files" {
		t.Fatalf("two: %s", got)
	}
}

func TestF8MovesSelectedToTrash(t *testing.T) {
	fakeTrash(t)
	a, dir := newApp(t)
	for _, n := range []string{"a", "b", "c"} {
		must(t, os.WriteFile(filepath.Join(dir, n), nil, 0o644))
	}
	p := a.panels[0]
	must(t, p.Reload())
	p.Selected["a"], p.Selected["b"] = true, true
	press(a, tcell.KeyF8, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != "Move 2 files to Trash?" || d.Danger {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	if !a.modals.Empty() || len(p.Selected) != 0 {
		t.Fatalf("modals %d selected %v", a.modals.Len(), p.Selected)
	}
	if p.Focus("a") || p.Focus("b") || !p.Focus("c") {
		t.Fatal("wrong files deleted or panel not re-read")
	}
}

func TestF8OnUpDirDoesNothing(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Home() // ".."
	press(a, tcell.KeyF8, 0, 0)
	if !a.modals.Empty() {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestShiftF8AsksAboutNonEmptyDir(t *testing.T) {
	for _, k := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyF8, 0, tcell.ModShift),
		tcell.NewEventKey(tcell.KeyF20, 0, 0),
	} {
		a, dir := newApp(t)
		a.panels[0].Focus("sub")
		a.HandleEvent(k)
		a.Draw()
		d, ok := a.modals.Top().(*ui.Dialog)
		if !ok || d.Lines[0] != `Delete "sub" permanently?` || !d.Danger {
			t.Fatalf("%s: top %#v", k.Name(), a.modals.Top())
		}
		press(a, tcell.KeyEnter, 0, 0)
		settle(t, a)
		d, ok = a.modals.Top().(*ui.Dialog)
		if !ok || d.Lines[0] != "Directory sub is not empty." {
			t.Fatalf("%s: top %#v", k.Name(), a.modals.Top())
		}
		press(a, tcell.KeyRune, 'd', 0)
		settle(t, a)
		if !a.modals.Empty() {
			t.Fatalf("%s: top %#v", k.Name(), a.modals.Top())
		}
		if _, err := os.Stat(filepath.Join(dir, "sub")); !os.IsNotExist(err) {
			t.Fatalf("%s: sub still there: %v", k.Name(), err)
		}
	}
}

func TestShiftF8CancelInQuestionKeepsDir(t *testing.T) {
	a, dir := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF8, 0, tcell.ModShift)
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	press(a, tcell.KeyEscape, 0, 0) // Esc in the question = Cancel
	settle(t, a)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "f.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestOperationErrorAsksRetrySkipCancel(t *testing.T) {
	a, _ := newApp(t)
	a.run("Test", false, func(j *ops.Job) []string {
		if j.Fail(os.ErrPermission, true) == ops.Skip {
			return []string{"skipped"}
		}
		return nil
	})
	settle(t, a)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Error" || !d.Danger || strings.Join(d.Buttons, ",") != "Retry,Skip,Cancel" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, 's', 0)
	settle(t, a)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
}

func TestFixedErrorAsksSkipCancel(t *testing.T) {
	a, _ := newApp(t)
	got := make(chan ops.Answer, 1)
	a.run("Test", false, func(j *ops.Job) []string {
		got <- j.Fail(os.ErrExist, false)
		return nil
	})
	settle(t, a)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || strings.Join(d.Buttons, ",") != "Skip,Cancel" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, 's', 0)
	if ans := <-got; ans != ops.Skip {
		t.Fatalf("answer %v", ans)
	}
	settle(t, a)
}

func TestProgressAppearsAndCancels(t *testing.T) {
	a, _ := newApp(t)
	stopped := make(chan struct{})
	a.run("Test", true, func(j *ops.Job) []string {
		j.Progress(ops.Progress{File: "/x/big", FileDone: 1, FileSize: 4, Done: 1, Total: 2})
		for !j.Canceled() {
			time.Sleep(time.Millisecond)
		}
		close(stopped)
		return nil
	})
	v := a.modals.Top().(*ui.Progress)
	p := a.panels[0].Cursor
	press(a, tcell.KeyEscape, 0, 0) // hidden: ignored
	press(a, tcell.KeyDown, 0, 0)   // never reaches the panel
	if a.panels[0].Cursor != p {
		t.Fatal("key reached the panel")
	}
	deadline := time.After(5 * time.Second)
	for v.Hidden {
		select {
		case f := <-a.calls:
			f()
		case <-deadline:
			t.Fatal("progress window never showed")
		}
	}
	if v.File != "big" || len(v.Bars) != 2 || v.Bars[0] != 0.25 || v.Bars[1] != 0.5 {
		t.Fatalf("view %+v", v)
	}
	select {
	case <-stopped:
		t.Fatal("canceled by a key pressed while hidden")
	default:
	}
	a.Draw()
	press(a, tcell.KeyEscape, 0, 0)
	settle(t, a)
	<-stopped
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
}

func TestMenuDeleteItems(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyRight, 0, 0) // Files
	for i := 0; i < 7; i++ {       // Help → … → Delete → Delete permanently
		press(a, tcell.KeyDown, 0, 0)
	}
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || !d.Danger || !strings.HasSuffix(d.Lines[0], "permanently?") {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

// twoDirs puts the right panel into a new directory "dst" next to "sub".
func twoDirs(t *testing.T) (*App, string, string) {
	a, dir := newApp(t)
	dst := filepath.Join(dir, "dst")
	must(t, os.Mkdir(dst, 0o755))
	must(t, a.panels[1].Load(dst))
	must(t, a.panels[0].Reload())
	return a, dir, dst
}

func TestF5CopiesToOtherPanel(t *testing.T) {
	a, dir, dst := twoDirs(t)
	must(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644))
	must(t, a.panels[0].Reload())
	a.panels[0].Focus("a.txt")
	press(a, tcell.KeyF5, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Copy" || d.Lines[0] != `Copy "a.txt" to:` || d.Input.Text != dst || d.Buttons[0] != "Copy" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	if b, err := os.ReadFile(filepath.Join(dst, "a.txt")); err != nil || string(b) != "a" {
		t.Fatalf("copy: %q %v", b, err)
	}
	if !a.modals.Empty() || !a.panels[1].Focus("a.txt") {
		t.Fatal("dialog left open or other panel not re-read")
	}
}

func TestF6RenamesRelativeToPanel(t *testing.T) {
	a, dir := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF6, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != `Rename or move "sub" to:` || d.Buttons[0] != "Move" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	typeText(a, "renamed")
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	if _, err := os.Stat(filepath.Join(dir, "renamed", "f.txt")); err != nil {
		t.Fatal(err)
	}
	if a.panels[0].Focus("sub") {
		t.Fatal("panel not re-read")
	}
}

func TestCopyConflictDialogSkip(t *testing.T) {
	a, dir, dst := twoDirs(t)
	for _, n := range []string{"a", "b"} {
		must(t, os.WriteFile(filepath.Join(dir, n), []byte("new"), 0o644))
	}
	must(t, os.WriteFile(filepath.Join(dst, "a"), []byte("old"), 0o644))
	p := a.panels[0]
	must(t, p.Reload())
	p.Selected["a"], p.Selected["b"] = true, true
	press(a, tcell.KeyF5, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Warning" || strings.Join(d.Buttons, ",") != "Overwrite,Skip,Overwrite all,Skip all,Cancel" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, 's', 0)
	settle(t, a)
	if b, _ := os.ReadFile(filepath.Join(dst, "a")); string(b) != "old" {
		t.Fatal("skipped file overwritten")
	}
	if !p.Selected["a"] || p.Selected["b"] {
		t.Fatalf("selected %v: skipped must stay, copied must go", p.Selected)
	}
}

func TestCopyIntoItselfShowsError(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF5, 0, 0)
	typeText(a, "sub/in")
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Error" || !strings.Contains(strings.Join(d.Lines, " "), "into itself") {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, 'c', 0)
	settle(t, a)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
}

func TestF5CancelDoesNothing(t *testing.T) {
	a, _, dst := twoDirs(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF5, 0, 0)
	press(a, tcell.KeyEscape, 0, 0)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
	if _, err := os.Stat(filepath.Join(dst, "sub")); !os.IsNotExist(err) {
		t.Fatal("copied after Cancel")
	}
}

func TestMenuCopyAndMove(t *testing.T) {
	for i, title := range []string{"Copy", "Rename/Move"} {
		a, _ := newApp(t)
		a.panels[0].Focus("sub")
		press(a, tcell.KeyF9, 0, 0)
		press(a, tcell.KeyRight, 0, 0) // Files
		for k := 0; k < 3+i; k++ {     // Help → View → Edit → Copy (→ Rename/Move)
			press(a, tcell.KeyDown, 0, 0)
		}
		press(a, tcell.KeyEnter, 0, 0)
		if d, ok := a.modals.Top().(*ui.Dialog); !ok || d.Title != title {
			t.Fatalf("%s: top %#v", title, a.modals.Top())
		}
	}
}
