package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveFilesAndEmptyDirsWithoutAsking(t *testing.T) {
	dir := t.TempDir()
	a, e := filepath.Join(dir, "a"), filepath.Join(dir, "empty")
	write(t, a, "a", 0o644)
	must(t, os.Mkdir(e, 0o755))
	var s script
	var last Progress
	j := newJob(&s)
	j.Progress = func(p Progress) { last = p }
	done := Remove(j, []string{a, e})
	if strings.Join(done, " ") != "a empty" || len(s.asked) != 0 || exists(a) || exists(e) {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if last.Done != 2 || last.Total != 2 {
		t.Fatalf("progress %+v", last)
	}
}

// fullDirs makes two non-empty directories d1 and d2.
func fullDirs(t *testing.T) []string {
	dir := t.TempDir()
	var ds []string
	for _, n := range []string{"d1", "d2"} {
		write(t, filepath.Join(dir, n, "x"), "x", 0o644)
		ds = append(ds, filepath.Join(dir, n))
	}
	return ds
}

func TestRemoveAsksForNonEmptyDirs(t *testing.T) {
	ds := fullDirs(t)
	s := script{answers: []Answer{Skip, Yes}}
	done := Remove(newJob(&s), ds)
	if strings.Join(done, " ") != "d2" || strings.Join(s.asked, ",") != "not empty d1,not empty d2" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(ds[0]) || exists(ds[1]) {
		t.Fatal("wrong directory deleted")
	}
}

func TestRemoveAllAsksOnce(t *testing.T) {
	ds := fullDirs(t)
	s := script{answers: []Answer{All}}
	if done := Remove(newJob(&s), ds); len(done) != 2 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
}

func TestRemoveCancelStops(t *testing.T) {
	ds := fullDirs(t)
	var s script // Cancel
	j := newJob(&s)
	if done := Remove(j, ds); len(done) != 0 || len(s.asked) != 1 || !j.Canceled() {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(ds[0]) || !exists(ds[1]) {
		t.Fatal("deleted after cancel")
	}
}

func TestRemoveSymlinkToDirKeepsTarget(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "d", "x"), "x", 0o644)
	link := filepath.Join(dir, "link")
	must(t, os.Symlink(filepath.Join(dir, "d"), link))
	var s script
	if done := Remove(newJob(&s), []string{link}); len(done) != 1 || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(link) || !exists(filepath.Join(dir, "d", "x")) {
		t.Fatal("wrong thing deleted")
	}
}

func TestTrash(t *testing.T) {
	if _, err := os.Stat(TrashCmd); err != nil {
		t.Skip("no " + TrashCmd)
	}
	a := filepath.Join(t.TempDir(), "tc-trash-test.txt")
	write(t, a, "a", 0o644)
	var s script
	if done := Trash(newJob(&s), []string{a}); len(done) != 1 || len(s.asked) != 0 || exists(a) {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
}

func TestTrashMissingFileAsks(t *testing.T) {
	s := script{answers: []Answer{Skip}}
	missing := filepath.Join(t.TempDir(), "missing")
	if done := Trash(newJob(&s), []string{missing}); len(done) != 0 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !strings.Contains(s.asked[0], "no such file") {
		t.Fatalf("asked %q", s.asked)
	}
}
