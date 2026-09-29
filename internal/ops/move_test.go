package ops

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestMoveRenames(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	write(t, a, "a", 0o644)
	var s script
	var last Progress
	j := newJob(&s)
	j.Progress = func(p Progress) { last = p }
	done := Move(j, []string{a}, filepath.Join(dir, "b"))
	if strings.Join(done, " ") != "a" || exists(a) || read(t, filepath.Join(dir, "b")) != "a" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if last.Done != 1 || last.Total != 1 {
		t.Fatalf("progress %+v", last)
	}
}

func TestMoveManyIntoDir(t *testing.T) {
	dir := t.TempDir()
	a, d := filepath.Join(dir, "a"), filepath.Join(dir, "d")
	write(t, a, "a", 0o644)
	write(t, filepath.Join(d, "x"), "x", 0o644)
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	if done := Move(newJob(&s), []string{a, d}, out); len(done) != 2 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(a) || exists(d) || read(t, filepath.Join(out, "d", "x")) != "x" {
		t.Fatal("not moved")
	}
}

// crossDevice makes rename fail with EXDEV for the rest of the test.
func crossDevice(t *testing.T) {
	rename = func(a, b string) error { return &os.LinkError{Op: "rename", Old: a, New: b, Err: syscall.EXDEV} }
	t.Cleanup(func() { rename = os.Rename })
}

func TestMoveAcrossVolumesCopiesAndDeletes(t *testing.T) {
	crossDevice(t)
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	write(t, filepath.Join(d, "sub", "x"), "x", 0o640)
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	if done := Move(newJob(&s), []string{d}, out); len(done) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(d) || read(t, filepath.Join(out, "d", "sub", "x")) != "x" {
		t.Fatal("not moved")
	}
}

func TestMoveAcrossVolumesOverwrites(t *testing.T) {
	crossDevice(t)
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{Yes, Skip}}
	done := Move(newJob(&s), srcs, dst)
	if strings.Join(done, " ") != "a" || len(s.asked) != 2 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "new" || exists(srcs[0]) || !exists(srcs[1]) {
		t.Fatal("wrong result")
	}
}

func TestMoveAcrossVolumesCanceledKeepsSource(t *testing.T) {
	crossDevice(t)
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	write(t, filepath.Join(d, "x"), "x", 0o644)
	write(t, filepath.Join(d, "y"), "y", 0o644)
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	j := newJob(&s)
	j.Progress = func(p Progress) {
		if p.FileSize > 0 {
			j.Cancel() // during the first file
		}
	}
	if done := Move(j, []string{d}, out); len(done) != 0 {
		t.Fatalf("done %q", done)
	}
	if !exists(filepath.Join(d, "x")) || !exists(filepath.Join(d, "y")) {
		t.Fatal("source deleted after a canceled copy")
	}
}

func TestMoveConflict(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{Skip, Yes}}
	done := Move(newJob(&s), srcs, dst)
	if strings.Join(done, " ") != "b" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "old" || !exists(srcs[0]) {
		t.Fatal("skipped file touched")
	}
	if read(t, filepath.Join(dst, "b")) != "new" || exists(srcs[1]) {
		t.Fatal("b not moved")
	}
}

func TestMoveOntoExistingDirFails(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "d", "x"), "x", 0o644)
	must(t, os.MkdirAll(filepath.Join(dir, "out", "d"), 0o755))
	s := script{answers: []Answer{Skip}}
	Move(newJob(&s), []string{filepath.Join(dir, "d")}, filepath.Join(dir, "out"))
	if len(s.asked) != 1 || !strings.Contains(s.asked[0], "already exists") {
		t.Fatalf("asked %q", s.asked)
	}
	if !exists(filepath.Join(dir, "d", "x")) {
		t.Fatal("source lost")
	}
}

func TestMoveDirIntoItself(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	must(t, os.MkdirAll(filepath.Join(d, "sub"), 0o755))
	s := script{answers: []Answer{Skip}}
	done := Move(newJob(&s), []string{d}, filepath.Join(d, "sub"))
	if len(done) != 0 || len(s.asked) != 1 || !strings.Contains(s.asked[0], "into itself") {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(filepath.Join(d, "sub")) {
		t.Fatal("source changed")
	}
}

func TestMoveChangesCaseOnly(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "foo")
	write(t, filepath.Join(d, "x"), "x", 0o644)
	if !exists(filepath.Join(dir, "FOO")) {
		t.Skip("case-sensitive file system")
	}
	var s script
	if done := Move(newJob(&s), []string{d}, filepath.Join(dir, "Foo")); len(done) != 1 || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	names, err := readNames(dir)
	must(t, err)
	if strings.Join(names, " ") != "Foo" {
		t.Fatalf("names %q", names)
	}
}
