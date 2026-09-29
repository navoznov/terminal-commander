package panel

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/navoznov/terminal-commander/internal/fs"
)

func selectDemo() *Panel {
	p := New()
	p.Entries = []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true},
		{Name: "docs.txt", IsDir: true},
		{Name: "a.txt"},
		{Name: "B.TXT"},
		{Name: "c.go"},
	}
	return p
}

// selected lists the selected names in panel order.
func selected(p *Panel) string {
	var ns []string
	for _, e := range p.Entries {
		if p.Selected[e.Name] {
			ns = append(ns, e.Name)
		}
	}
	return strings.Join(ns, " ")
}

func TestSelectMaskFilesOnlyIgnoringCase(t *testing.T) {
	p := selectDemo()
	if err := p.SelectMask("*.txt", true); err != nil {
		t.Fatal(err)
	}
	if got := selected(p); got != "a.txt B.TXT" {
		t.Fatalf("got %q", got)
	}
	if err := p.SelectMask("B*", false); err != nil {
		t.Fatal(err)
	}
	if got := selected(p); got != "a.txt" {
		t.Fatalf("got %q", got)
	}
}

func TestSelectMaskKeepsManualDirSelection(t *testing.T) {
	p := selectDemo()
	p.Selected["docs.txt"] = true
	if err := p.SelectMask("*", false); err != nil {
		t.Fatal(err)
	}
	if got := selected(p); got != "docs.txt" {
		t.Fatalf("got %q", got)
	}
}

func TestSelectMaskBadPattern(t *testing.T) {
	p := selectDemo()
	p.Selected["c.go"] = true
	err := p.SelectMask("[", true)
	if !errors.Is(err, filepath.ErrBadPattern) {
		t.Fatalf("err %v", err)
	}
	if got := selected(p); got != "c.go" {
		t.Fatalf("selection changed: %q", got)
	}
}

func TestInvertSelection(t *testing.T) {
	p := selectDemo()
	p.Selected["a.txt"] = true
	p.Selected["docs.txt"] = true
	p.InvertSelection()
	if got := selected(p); got != "docs.txt B.TXT c.go" {
		t.Fatalf("got %q", got)
	}
}

func TestSources(t *testing.T) {
	p := selectDemo()
	if got := p.Sources(); got != nil {
		t.Fatalf("cursor on ..: %q", got)
	}
	p.Cursor = 2
	if got := strings.Join(p.Sources(), " "); got != "a.txt" {
		t.Fatalf("cursor: %q", got)
	}
	p.Selected["c.go"] = true
	p.Selected["docs.txt"] = true
	if got := strings.Join(p.Sources(), " "); got != "docs.txt c.go" {
		t.Fatalf("selection: %q", got)
	}
	if got := New().Sources(); got != nil {
		t.Fatalf("empty panel: %q", got)
	}
}
