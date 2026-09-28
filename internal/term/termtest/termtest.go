// Package termtest provides a simulated screen and golden-file helpers.
package termtest

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

var update = flag.Bool("update", false, "rewrite golden files")

// NewScreen returns an initialized simulation screen of w×h cells.
func NewScreen(t *testing.T, w, h int) tcell.SimulationScreen {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	s.SetSize(w, h)
	t.Cleanup(s.Fini)
	return s
}

// Dump shows the screen and returns its characters, a blank line, and a grid
// of term.StyleCode letters for every cell.
func Dump(s tcell.SimulationScreen) string {
	s.Show()
	cells, w, h := s.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := ' '
			if rs := cells[y*w+x].Runes; len(rs) > 0 {
				r = rs[0]
			}
			b.WriteRune(r)
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.WriteRune(term.StyleCode(cells[y*w+x].Style))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Golden compares got with testdata/<name>.golden; with -update it rewrites
// the file instead.
func Golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test with -update to create it)", err)
	}
	if string(want) != got {
		t.Errorf("screen differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
