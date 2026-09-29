package panel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/navoznov/terminal-commander/internal/fs"
)

// fake builds a panel over fixed entries; names ending in "/" are dirs.
func fake(rows int, names ...string) *Panel {
	p := New()
	p.Path = "/demo"
	for _, n := range names {
		e := fs.Entry{Name: strings.TrimSuffix(n, "/"), Size: 10}
		e.IsDir = strings.HasSuffix(n, "/")
		e.IsUp = n == "../"
		p.Entries = append(p.Entries, e)
	}
	p.SetRows(rows)
	return p
}

func seq(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, string(rune('a'+i)))
	}
	return out
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSortEntries(t *testing.T) {
	es := []fs.Entry{
		{Name: "b.txt"}, {Name: "Zdir", IsDir: true}, {Name: "A.txt"},
		{Name: "..", IsDir: true, IsUp: true}, {Name: "adir", IsDir: true},
	}
	SortEntries(es)
	var got []string
	for _, e := range es {
		got = append(got, e.Name)
	}
	if strings.Join(got, ",") != "..,adir,Zdir,A.txt,b.txt" {
		t.Fatalf("got %v", got)
	}
}

func TestMoveClamps(t *testing.T) {
	p := fake(3, seq(5)...)
	p.Move(-1)
	if p.Cursor != 0 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Move(100)
	if p.Cursor != 4 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Home()
	if p.Cursor != 0 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.End()
	if p.Cursor != 4 {
		t.Fatalf("cursor %d", p.Cursor)
	}
}

func TestBriefScrollsByColumn(t *testing.T) {
	p := fake(3, seq(12)...) // capacity 9
	p.Move(9)
	if p.Top != 3 {
		t.Fatalf("top %d", p.Top)
	}
	p.Home()
	if p.Top != 0 {
		t.Fatalf("top %d", p.Top)
	}
}

func TestBriefLeftRightJumpColumns(t *testing.T) {
	p := fake(3, seq(8)...)
	p.Right()
	if p.Cursor != 3 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Right()
	p.Right()
	if p.Cursor != 7 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Left()
	if p.Cursor != 4 {
		t.Fatalf("cursor %d", p.Cursor)
	}
}

func TestFullScrollsByLine(t *testing.T) {
	p := fake(3, seq(8)...)
	p.SetMode(Full)
	p.Move(5)
	if p.Top != 3 {
		t.Fatalf("top %d", p.Top)
	}
	p.PageUp()
	if p.Cursor != 2 || p.Top != 2 {
		t.Fatalf("cursor %d top %d", p.Cursor, p.Top)
	}
}

func TestSetRowsKeepsCursorVisible(t *testing.T) {
	p := fake(10, seq(20)...)
	p.Move(19)
	p.SetRows(2)
	if p.Cursor < p.Top || p.Cursor >= p.Top+6 {
		t.Fatalf("cursor %d top %d", p.Cursor, p.Top)
	}
}

func TestToggleSelectSkipsUpAndMovesDown(t *testing.T) {
	p := fake(5, "../", "a", "b")
	p.ToggleSelect()
	if len(p.Selected) != 0 || p.Cursor != 1 {
		t.Fatalf("selected %v cursor %d", p.Selected, p.Cursor)
	}
	p.ToggleSelect()
	p.ToggleSelect()
	if n, bytes := p.SelectionStats(); n != 2 || bytes != 20 {
		t.Fatalf("n %d bytes %d", n, bytes)
	}
	p.Home()
	p.Move(1)
	p.ToggleSelect()
	if n, _ := p.SelectionStats(); n != 1 {
		t.Fatalf("n %d", n)
	}
}

func TestEmptyPanel(t *testing.T) {
	p := fake(3)
	p.Move(1)
	p.End()
	p.Right()
	p.ToggleSelect()
	if p.Current() != nil {
		t.Fatal("current must be nil")
	}
	if ok, err := p.Enter(); ok || err != nil {
		t.Fatalf("enter: %v %v", ok, err)
	}
}

func TestEnterOnFileDoesNothing(t *testing.T) {
	p := fake(3, "a")
	if ok, _ := p.Enter(); ok {
		t.Fatal("entered a file")
	}
}

func TestEnterAndUpFocusesChild(t *testing.T) {
	dir := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755))
	must(t, os.Mkdir(filepath.Join(dir, "a", "a0"), 0o755))
	p := New()
	p.SetRows(10)
	must(t, p.Load(filepath.Join(dir, "a")))
	p.Focus("b")
	ok, err := p.Enter()
	must(t, err)
	if !ok || p.Path != filepath.Join(dir, "a", "b") {
		t.Fatalf("path %s", p.Path)
	}
	must(t, p.Up())
	if p.Path != filepath.Join(dir, "a") || p.Current().Name != "b" {
		t.Fatalf("path %s current %+v", p.Path, p.Current())
	}
}

func TestUpAtRootIsNoop(t *testing.T) {
	p := New()
	must(t, p.Load("/"))
	must(t, p.Up())
	if p.Path != "/" {
		t.Fatalf("path %s", p.Path)
	}
}

func TestReloadKeepsCursorOnSameName(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "m"), nil, 0o644))
	p := New()
	p.SetRows(10)
	must(t, p.Load(dir))
	p.Focus("m")
	p.Selected["m"] = true
	must(t, os.WriteFile(filepath.Join(dir, "a"), nil, 0o644))
	must(t, p.Reload())
	if p.Current().Name != "m" || !p.Selected["m"] {
		t.Fatalf("current %+v selected %v", p.Current(), p.Selected)
	}
}

func TestReloadClimbsWhenDirRemoved(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(dir, "x", "y")
	must(t, os.MkdirAll(gone, 0o755))
	p := New()
	must(t, p.Load(gone))
	must(t, os.RemoveAll(filepath.Join(dir, "x")))
	must(t, p.Reload())
	if p.Path != dir {
		t.Fatalf("path %s want %s", p.Path, dir)
	}
}

func TestSetShowHidden(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, ".dot"), nil, 0o644))
	p := New()
	must(t, p.Load(dir))
	if p.Focus(".dot") {
		t.Fatal(".dot visible")
	}
	must(t, p.SetShowHidden(true))
	if !p.Focus(".dot") {
		t.Fatal(".dot hidden")
	}
}

func TestShrinkingListClampsTop(t *testing.T) {
	for _, mode := range []Mode{Full, Brief} {
		dir := t.TempDir()
		for i := 0; i < 31; i++ {
			must(t, os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%02d", i)), nil, 0o644))
		}
		p := New()
		p.SetRows(10)
		must(t, p.Load(dir))
		p.SetMode(mode)
		p.End()
		for i := 5; i < 31; i++ {
			must(t, os.Remove(filepath.Join(dir, fmt.Sprintf("f%02d", i))))
		}
		must(t, p.Reload())
		if p.Top != 0 {
			t.Fatalf("mode %v: %d entries, cursor %d, top %d — entries hidden above", mode, len(p.Entries), p.Cursor, p.Top)
		}
	}
}
