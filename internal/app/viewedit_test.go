package app

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ui"
	"github.com/navoznov/terminal-commander/internal/viewer"
)

func TestF3ViewsFile(t *testing.T) {
	a, _ := newApp(t)
	addFile(t, a, "a.txt", "hello viewer\n", 0o644)
	press(a, tcell.KeyF3, 0, 0)
	if _, ok := a.modals.Top().(*viewer.Viewer); !ok {
		t.Fatalf("top %#v", a.modals.Top())
	}
	if row := screenRow(a, 1); !strings.HasPrefix(row, "hello viewer") {
		t.Fatalf("row %q", row)
	}
	press(a, tcell.KeyEscape, 0, 0)
	if !a.modals.Empty() {
		t.Fatal("viewer not closed")
	}
}

func TestMenuViewOpensViewer(t *testing.T) {
	a, _ := newApp(t)
	addFile(t, a, "a.txt", "x", 0o644)
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyRight, 0, 0) // Files
	press(a, tcell.KeyDown, 0, 0)  // View
	press(a, tcell.KeyEnter, 0, 0)
	if _, ok := a.modals.Top().(*viewer.Viewer); !ok {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestF3OnDirectoryDoesNothing(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF3, 0, 0)
	press(a, tcell.KeyF4, 0, 0)
	if !a.modals.Empty() {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestF3OnFifoShowsError(t *testing.T) {
	a, dir := newApp(t)
	must(t, syscall.Mkfifo(filepath.Join(dir, "pipe"), 0o644))
	must(t, a.panels[0].Reload())
	a.panels[0].Focus("pipe")
	press(a, tcell.KeyF3, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || !d.Danger {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestF4RunsEditor(t *testing.T) {
	a, dir := newApp(t)
	out := quietConsole(t, a)
	log := filepath.Join(dir, "edited")
	editor := filepath.Join(t.TempDir(), "ed")
	must(t, os.WriteFile(editor, []byte("#!/bin/sh\necho \"$PWD $1\" > '"+log+"'\n"), 0o755))
	t.Setenv("EDITOR", editor)
	addFile(t, a, "my file.txt", "", 0o644)
	press(a, tcell.KeyF4, 0, 0)
	got, err := os.ReadFile(log)
	must(t, err)
	if want := dir + " my file.txt\n"; string(got) != want {
		t.Fatalf("editor got %q, want %q", got, want)
	}
	if !a.panels[0].Focus("edited") || out.Len() != 0 {
		t.Fatalf("panels not re-read or output %q", out.String())
	}
}

func TestF4EditorFailureWaits(t *testing.T) {
	a, _ := newApp(t)
	out := quietConsole(t, a)
	t.Setenv("EDITOR", "/nonexistent/editor")
	addFile(t, a, "a.txt", "", 0o644)
	press(a, tcell.KeyF4, 0, 0)
	if !strings.Contains(out.String(), "Press any key") {
		t.Fatalf("output %q", out.String())
	}
}
