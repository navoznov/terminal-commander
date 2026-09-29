package panel

import (
	"strings"
	"testing"
	"time"

	"github.com/navoznov/terminal-commander/internal/fs"
)

func sortDemo() []fs.Entry {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	return []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true},
		{Name: "b.txt", Size: 10, ModTime: t0.Add(3 * time.Hour)},
		{Name: "zdir", IsDir: true, Size: 900, ModTime: t0.Add(5 * time.Hour)},
		{Name: "a.md", Size: 30, ModTime: t0.Add(1 * time.Hour)},
		{Name: "Adir", IsDir: true, Size: 100, ModTime: t0},
		{Name: "c.go", Size: 20, ModTime: t0.Add(2 * time.Hour)},
		{Name: "Makefile", Size: 20, ModTime: t0.Add(2 * time.Hour)},
	}
}

func entryNames(es []fs.Entry) string {
	var ns []string
	for _, e := range es {
		ns = append(ns, e.Name)
	}
	return strings.Join(ns, " ")
}

func TestSortModes(t *testing.T) {
	cases := []struct {
		mode SortMode
		want string
	}{
		{SortName, ".. Adir zdir a.md b.txt c.go Makefile"},
		{SortExt, ".. Adir zdir Makefile c.go a.md b.txt"},
		{SortTime, ".. zdir Adir b.txt c.go Makefile a.md"},
		{SortSize, ".. Adir zdir a.md c.go Makefile b.txt"},
		{Unsorted, ".. b.txt zdir a.md Adir c.go Makefile"},
	}
	for _, c := range cases {
		es := sortDemo()
		SortEntries(es, c.mode)
		if got := entryNames(es); got != c.want {
			t.Errorf("mode %d: got %q want %q", c.mode, got, c.want)
		}
	}
}
