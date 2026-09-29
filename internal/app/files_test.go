package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestResolve(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"b", "/a/b"},
		{"b/../c", "/a/c"},
		{"sub/", "/a/sub/"},
		{"/x//y", "/x/y"},
		{"/", "/"},
		{"~", "/home"},
		{"~/x", "/home/x"},
		{"~x", "/a/~x"},
	} {
		if got := resolve("/a", "/home", c.in); got != c.want {
			t.Errorf("resolve(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func typeText(a *App, s string) {
	for _, r := range s {
		press(a, tcell.KeyRune, r, 0)
	}
}

func TestF7MakesDirectoryAndFocusesIt(t *testing.T) {
	a, dir := newApp(t)
	press(a, tcell.KeyF7, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Make directory" || d.Input == nil {
		t.Fatalf("top %#v", a.modals.Top())
	}
	typeText(a, "new/deep")
	press(a, tcell.KeyEnter, 0, 0)
	if fi, err := os.Stat(filepath.Join(dir, "new", "deep")); err != nil || !fi.IsDir() {
		t.Fatalf("not created: %v", err)
	}
	if !a.modals.Empty() || a.panels[0].Current().Name != "new" {
		t.Fatalf("modals %d current %s", a.modals.Len(), a.panels[0].Current().Name)
	}
	if !a.panels[1].Focus("new") {
		t.Fatal("the other panel was not re-read")
	}
}

func TestF7EmptyNameDoesNothing(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF7, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	if !a.modals.Empty() {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestF7ErrorIsReported(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF7, 0, 0)
	typeText(a, "sub/f.txt/x") // f.txt is a file
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || !d.Danger {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestMenuMakeDirectory(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyRight, 0, 0) // Files
	for i := 0; i < 5; i++ {       // Help → View → Edit → Copy → Rename/Move → Make directory
		press(a, tcell.KeyDown, 0, 0)
	}
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || d.Title != "Make directory" {
		t.Fatalf("top %#v", a.modals.Top())
	}
}
