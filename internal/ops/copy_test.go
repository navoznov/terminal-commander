package ops

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFileIntoDirKeepsModeAndTime(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	write(t, src, "hello", 0o640)
	dst := filepath.Join(dir, "out")
	must(t, os.Mkdir(dst, 0o755))
	var s script
	j := newJob(&s)
	var last Progress
	j.Progress = func(p Progress) { last = p }
	done := Copy(j, []string{src}, dst)
	if strings.Join(done, " ") != "a.txt" || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	out := filepath.Join(dst, "a.txt")
	fi, err := os.Stat(out)
	must(t, err)
	if read(t, out) != "hello" || fi.Mode().Perm() != 0o640 || !fi.ModTime().Equal(stamp) {
		t.Fatalf("mode %v time %v", fi.Mode(), fi.ModTime())
	}
	if last.Total != 5 || last.Done != 5 || last.FileSize != 5 || last.FileDone != 5 {
		t.Fatalf("progress %+v", last)
	}
	if !exists(src) {
		t.Fatal("source removed")
	}
}

func TestCopyToNewName(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	write(t, src, "x", 0o644)
	var s script
	Copy(newJob(&s), []string{src}, filepath.Join(dir, "b.txt"))
	if read(t, filepath.Join(dir, "b.txt")) != "x" {
		t.Fatal("not copied")
	}
}

func TestCopyTreeWithSymlink(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	write(t, filepath.Join(d, "x.txt"), "x", 0o644)
	write(t, filepath.Join(d, "sub", "y.txt"), "y", 0o600)
	must(t, os.Symlink("x.txt", filepath.Join(d, "link")))
	must(t, os.Chmod(d, 0o750))
	must(t, os.Chtimes(d, stamp, stamp))
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	if done := Copy(newJob(&s), []string{d}, out); strings.Join(done, " ") != "d" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	c := filepath.Join(out, "d")
	if read(t, filepath.Join(c, "x.txt")) != "x" || read(t, filepath.Join(c, "sub", "y.txt")) != "y" {
		t.Fatal("files not copied")
	}
	if target, err := os.Readlink(filepath.Join(c, "link")); err != nil || target != "x.txt" {
		t.Fatalf("link %q %v", target, err)
	}
	fi, err := os.Stat(c)
	must(t, err)
	if fi.Mode().Perm() != 0o750 || !fi.ModTime().Equal(stamp) {
		t.Fatalf("dir mode %v time %v", fi.Mode(), fi.ModTime())
	}
}

func TestCopyManyCreatesDestination(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, a, "a", 0o644)
	write(t, b, "b", 0o644)
	dst := filepath.Join(dir, "new", "place")
	var s script
	if done := Copy(newJob(&s), []string{a, b}, dst); len(done) != 2 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "a" || read(t, filepath.Join(dst, "b")) != "b" {
		t.Fatal("not copied into the new directory")
	}
}

func TestCopyTrailingSlashMeansDirectory(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	write(t, a, "a", 0o644)
	var s script
	Copy(newJob(&s), []string{a}, filepath.Join(dir, "new")+"/")
	if read(t, filepath.Join(dir, "new", "a")) != "a" {
		t.Fatal("not copied into new/")
	}
}

// conflictDir makes src/a, src/b ("new") and dst/a, dst/b ("old").
func conflictDir(t *testing.T) (srcs []string, dst string) {
	dir := t.TempDir()
	dst = filepath.Join(dir, "dst")
	for _, n := range []string{"a", "b"} {
		write(t, filepath.Join(dir, "src", n), "new", 0o644)
		write(t, filepath.Join(dst, n), "old", 0o644)
		srcs = append(srcs, filepath.Join(dir, "src", n))
	}
	return srcs, dst
}

func TestCopyConflictSkipThenOverwrite(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{Skip, Yes}}
	done := Copy(newJob(&s), srcs, dst)
	if strings.Join(done, " ") != "b" || strings.Join(s.asked, ",") != "conflict a,conflict b" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "old" || read(t, filepath.Join(dst, "b")) != "new" {
		t.Fatal("wrong files overwritten")
	}
}

func TestCopyOverwriteAllAsksOnce(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{All}}
	if done := Copy(newJob(&s), srcs, dst); len(done) != 2 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "new" || read(t, filepath.Join(dst, "b")) != "new" {
		t.Fatal("not overwritten")
	}
}

func TestCopySkipAllAsksOnce(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{SkipAll}}
	if done := Copy(newJob(&s), srcs, dst); len(done) != 0 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "b")) != "old" {
		t.Fatal("overwritten")
	}
}

func TestCopyCancelOnConflictStops(t *testing.T) {
	srcs, dst := conflictDir(t)
	var s script // no answers: Cancel
	j := newJob(&s)
	if done := Copy(j, srcs, dst); len(done) != 0 || len(s.asked) != 1 || !j.Canceled() {
		t.Fatalf("done %q asked %q canceled %v", done, s.asked, j.Canceled())
	}
}

func TestCopyMergesDirectories(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "d", "x"), "x", 0o644)
	write(t, filepath.Join(dir, "out", "d", "old"), "old", 0o644)
	var s script
	if done := Copy(newJob(&s), []string{filepath.Join(dir, "d")}, filepath.Join(dir, "out")); len(done) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(filepath.Join(dir, "out", "d", "x")) || !exists(filepath.Join(dir, "out", "d", "old")) {
		t.Fatal("not merged")
	}
}

func TestCopyFileOverDirFails(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "src", "a"), "a", 0o644)
	must(t, os.MkdirAll(filepath.Join(dir, "dst", "a"), 0o755))
	s := script{answers: []Answer{Skip}}
	Copy(newJob(&s), []string{filepath.Join(dir, "src", "a")}, filepath.Join(dir, "dst"))
	if len(s.asked) != 1 || !strings.Contains(s.asked[0], "cannot overwrite") {
		t.Fatalf("asked %q", s.asked)
	}
}

func TestCopyDirIntoItself(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	must(t, os.MkdirAll(filepath.Join(d, "sub"), 0o755))
	s := script{answers: []Answer{Skip}}
	done := Copy(newJob(&s), []string{d}, filepath.Join(d, "sub"))
	if len(done) != 0 || len(s.asked) != 1 || !strings.HasPrefix(s.asked[0], "error ") || !strings.Contains(s.asked[0], "into itself") {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(filepath.Join(d, "sub", "d")) {
		t.Fatal("copied into itself")
	}
}

func TestCopyOntoItself(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	write(t, a, "a", 0o644)
	s := script{answers: []Answer{Skip}}
	Copy(newJob(&s), []string{a}, dir)
	if len(s.asked) != 1 || !strings.Contains(s.asked[0], "onto itself") || read(t, a) != "a" {
		t.Fatalf("asked %q", s.asked)
	}
}

func TestInsideFollowsSymlinks(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	must(t, os.MkdirAll(filepath.Join(d, "sub"), 0o755))
	must(t, os.Symlink(d, filepath.Join(dir, "link")))
	if !inside(filepath.Join(dir, "link", "sub", "new"), d) {
		t.Fatal("path through a symlink not detected")
	}
	if !inside(d, d) {
		t.Fatal("the directory itself not detected")
	}
	if inside(filepath.Join(dir, "other"), d) {
		t.Fatal("sibling detected")
	}
	if inside(filepath.Join(d, "sub"), filepath.Join(dir, "link")) {
		t.Fatal("a symlink is copied as a link, never into itself")
	}
}

func TestCopyCancelRemovesPartialFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "big")
	must(t, os.WriteFile(src, bytes.Repeat([]byte{'x'}, 3*blockSize), 0o644))
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	j := newJob(&s)
	j.Progress = func(p Progress) {
		if p.FileDone >= blockSize {
			j.Cancel()
		}
	}
	if done := Copy(j, []string{src}, out); len(done) != 0 || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(filepath.Join(out, "big")) {
		t.Fatal("partial file left")
	}
}

func TestCopyErrorRetry(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "locked")
	write(t, src, "x", 0o000)
	t.Cleanup(func() { os.Chmod(src, 0o644) })
	var s script
	j := newJob(&s)
	fails := 0
	j.Fail = func(err error, retry bool) Answer {
		if !retry {
			t.Errorf("no retry for %v", err)
		}
		fails++
		must(t, os.Chmod(src, 0o644)) // the user fixed it and pressed Retry
		return Yes
	}
	dst := filepath.Join(dir, "copy")
	if done := Copy(j, []string{src}, dst); len(done) != 1 || fails != 1 || read(t, dst) != "x" {
		t.Fatalf("done %q fails %d", done, fails)
	}
}

func TestCopyCancelDuringScan(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	for _, n := range []string{"x", "y", "z"} {
		write(t, filepath.Join(d, n), n, 0o644)
	}
	var s script
	j := newJob(&s)
	var first *Progress
	j.Progress = func(p Progress) {
		if first == nil {
			first = &p
			j.Cancel()
		}
	}
	out := filepath.Join(dir, "out")
	if done := Copy(j, []string{d}, out); len(done) != 0 || exists(out) {
		t.Fatalf("done %q, out exists %v", done, exists(out))
	}
	if first == nil || first.File != d || first.FileSize != 0 {
		t.Fatalf("first progress %+v, want the scan of %s", first, d)
	}
	if n := j.size(d); n != 0 {
		t.Fatalf("scan after cancel counted %d bytes", n)
	}
}
