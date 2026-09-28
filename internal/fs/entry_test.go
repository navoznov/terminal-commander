package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func names(es []Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Name)
	}
	return out
}

func find(es []Entry, name string) *Entry {
	for i := range es {
		if es[i].Name == name {
			return &es[i]
		}
	}
	return nil
}

func mkTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, ".hidden"), nil, 0o644))
	must(t, os.Symlink(filepath.Join(dir, "sub"), filepath.Join(dir, "linkdir")))
	must(t, os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken")))
	return dir
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadDirUpFirstAndHidesDotfiles(t *testing.T) {
	es, err := ReadDir(mkTree(t), false)
	must(t, err)
	if !es[0].IsUp || es[0].Name != ".." || !es[0].IsDir {
		t.Fatalf("first entry %+v", es[0])
	}
	if find(es, ".hidden") != nil {
		t.Fatalf("hidden file listed: %v", names(es))
	}
	a := find(es, "a.txt")
	if a == nil || a.IsDir || a.Size != 5 {
		t.Fatalf("a.txt: %+v", a)
	}
}

func TestReadDirShowHidden(t *testing.T) {
	es, err := ReadDir(mkTree(t), true)
	must(t, err)
	if find(es, ".hidden") == nil {
		t.Fatalf("hidden file missing: %v", names(es))
	}
}

func TestReadDirSymlinks(t *testing.T) {
	es, err := ReadDir(mkTree(t), false)
	must(t, err)
	if l := find(es, "linkdir"); l == nil || !l.IsDir || !l.IsLink {
		t.Fatalf("linkdir: %+v", l)
	}
	if b := find(es, "broken"); b == nil || b.IsDir || !b.IsLink {
		t.Fatalf("broken: %+v", b)
	}
}

func TestReadDirRootHasNoUp(t *testing.T) {
	es, err := ReadDir("/", false)
	must(t, err)
	if len(es) > 0 && es[0].IsUp {
		t.Fatal("root must not have ..")
	}
}

func TestReadDirMissing(t *testing.T) {
	_, err := ReadDir(filepath.Join(t.TempDir(), "nope"), false)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}
