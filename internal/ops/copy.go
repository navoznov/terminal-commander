package ops

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const blockSize = 1 << 20

// Copy copies srcs into the directory dst, or to the new name dst when there
// is one source and dst is not a directory. Directories are copied
// recursively and merged into existing ones; permissions, modification times
// and symlinks are kept. It returns the names of the sources copied
// completely.
func Copy(j *Job, srcs []string, dst string) []string {
	j.countBytes = true
	for _, s := range srcs {
		j.progress.Total += size(s)
	}
	targets, ok := j.targets(srcs, dst, false)
	if !ok {
		return nil
	}
	var done []string
	for i, src := range srcs {
		if j.Canceled() {
			break
		}
		if j.copyTop(src, targets[i]) {
			done = append(done, filepath.Base(src))
		}
	}
	return done
}

// size is the total size of the regular files under path.
func size(path string) int64 {
	var n int64
	filepath.WalkDir(path, func(_ string, d iofs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if fi, err := d.Info(); err == nil {
				n += fi.Size()
			}
		}
		return nil
	})
	return n
}

// targets maps each source to its destination path. With several sources,
// a trailing "/" or an existing directory, dst is a directory, created if
// missing. A single source that is dst itself (maybe in another case of the
// name) is renamed to dst when moving.
func (j *Job) targets(srcs []string, dst string, move bool) ([]string, bool) {
	intoDir := len(srcs) > 1 || strings.HasSuffix(dst, "/")
	dst = filepath.Clean(dst)
	if fi, err := os.Stat(dst); err == nil && fi.IsDir() {
		intoDir = !(move && len(srcs) == 1 && sameFile(srcs[0], dst))
	}
	if !intoDir {
		return []string{dst}, true
	}
	if !j.do(func() error { return os.MkdirAll(dst, 0o755) }) {
		return nil, false
	}
	out := make([]string, len(srcs))
	for i, s := range srcs {
		out[i] = filepath.Join(dst, filepath.Base(s))
	}
	return out, true
}

// sameFile reports whether a and b both exist and are the same file (not
// following symlinks).
func sameFile(a, b string) bool {
	fa, err := os.Lstat(a)
	if err != nil {
		return false
	}
	fb, err := os.Lstat(b)
	return err == nil && os.SameFile(fa, fb)
}

// inside reports whether path is the directory dir or lies under it. The
// file system decides, so a different case of a name or a symlink on the way
// is caught. A dir that is a symlink is never "inside": links are copied as
// links.
func inside(path, dir string) bool {
	di, err := os.Lstat(dir)
	if err != nil || !di.IsDir() {
		return false
	}
	for p := filepath.Clean(path); ; p = filepath.Dir(p) {
		if fi, err := os.Stat(p); err == nil && os.SameFile(fi, di) {
			return true
		}
		if p == filepath.Dir(p) {
			return false
		}
	}
}

// copyTop copies one source after the checks that only make sense at the
// top level.
func (j *Job) copyTop(src, dst string) bool {
	switch {
	case sameFile(src, dst):
		return j.fail(fmt.Errorf("cannot copy %s onto itself", src))
	case inside(dst, src):
		return j.fail(fmt.Errorf("cannot copy %s into itself", src))
	}
	return j.copyItem(src, dst)
}

// copyItem copies src to dst, asking about an existing dst, and reports
// whether everything was copied.
func (j *Job) copyItem(src, dst string) bool {
	var si os.FileInfo
	if !j.do(func() (err error) { si, err = os.Lstat(src); return err }) {
		return false
	}
	if di, err := os.Lstat(dst); err == nil {
		switch {
		case si.IsDir() && di.IsDir():
			return j.copyDir(src, dst, si, true)
		case si.IsDir() || di.IsDir():
			return j.fail(fmt.Errorf("cannot overwrite %s", dst))
		case !j.overwrite(dst):
			return false
		case !j.do(func() error { return os.Remove(dst) }):
			return false
		}
	}
	switch {
	case si.Mode()&os.ModeSymlink != 0:
		return j.do(func() error {
			target, err := os.Readlink(src)
			if err != nil {
				return err
			}
			return os.Symlink(target, dst)
		})
	case si.IsDir():
		return j.copyDir(src, dst, si, false)
	case si.Mode().IsRegular():
		return j.do(func() error { return j.copyFile(src, dst, si) })
	}
	return j.fail(fmt.Errorf("%s: unsupported file type", src))
}

// copyDir copies the contents of src into dst, creating dst unless it
// exists. A new dst gets the mode and time of src after its contents are in.
func (j *Job) copyDir(src, dst string, si os.FileInfo, exists bool) bool {
	if !exists && !j.do(func() error { return os.Mkdir(dst, 0o700) }) {
		return false
	}
	var names []string
	if !j.do(func() (err error) { names, err = readNames(src); return err }) {
		return false
	}
	ok := true
	for _, n := range names {
		if j.Canceled() {
			return false
		}
		ok = j.copyItem(filepath.Join(src, n), filepath.Join(dst, n)) && ok
	}
	if !exists {
		ok = j.do(func() error {
			if err := os.Chmod(dst, si.Mode().Perm()); err != nil {
				return err
			}
			return os.Chtimes(dst, si.ModTime(), si.ModTime())
		}) && ok
	}
	return ok && !j.Canceled()
}

func readNames(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

var errCanceled = errors.New("canceled")

// copyFile copies a regular file in blocks, reporting progress. A copy that
// fails or is canceled is removed and its bytes are taken off the total.
func (j *Job) copyFile(src, dst string, si os.FileInfo) (err error) {
	start := j.progress.Done
	j.progress.File, j.progress.FileDone, j.progress.FileSize = src, 0, si.Size()
	j.Progress(j.progress)
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if si, err = in.Stat(); err != nil { // fresh: the user may have fixed it before Retry
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(dst)
			if j.countBytes {
				j.progress.Done = start
			}
		}
	}()
	buf := make([]byte, blockSize)
	for {
		if j.Canceled() {
			return errCanceled
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			j.progress.FileDone += int64(n)
			if j.countBytes {
				j.progress.Done += int64(n)
			}
			j.Progress(j.progress)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if err := out.Chmod(si.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(dst, si.ModTime(), si.ModTime())
}

var rename = os.Rename

// Move moves srcs like Copy. Within one volume it renames; across volumes it
// copies and deletes the source only after a complete copy. It returns the
// names of the sources moved.
func Move(j *Job, srcs []string, dst string) []string {
	j.progress.Total = int64(len(srcs))
	targets, ok := j.targets(srcs, dst, true)
	if !ok {
		return nil
	}
	var done []string
	for i, src := range srcs {
		if j.Canceled() {
			break
		}
		j.progress.File, j.progress.FileDone, j.progress.FileSize = src, 0, 0
		j.Progress(j.progress)
		if j.moveItem(src, targets[i]) {
			done = append(done, filepath.Base(src))
		}
		j.progress.Done++
		j.Progress(j.progress)
	}
	return done
}

func (j *Job) moveItem(src, dst string) bool {
	if sameFile(src, dst) { // the same path, or a new case of the name
		return j.do(func() error { return rename(src, dst) })
	}
	var si os.FileInfo
	if !j.do(func() (err error) { si, err = os.Lstat(src); return err }) {
		return false
	}
	if inside(dst, src) {
		return j.fail(fmt.Errorf("cannot move %s into itself", src))
	}
	if di, err := os.Lstat(dst); err == nil {
		switch {
		case si.IsDir() && di.IsDir():
			return j.fail(fmt.Errorf("%s already exists", dst))
		case si.IsDir() || di.IsDir():
			return j.fail(fmt.Errorf("cannot overwrite %s", dst))
		case !j.overwrite(dst):
			return false
		}
	}
	crossed := false
	if !j.do(func() error {
		err := rename(src, dst)
		crossed = errors.Is(err, syscall.EXDEV)
		if crossed {
			return nil
		}
		return err
	}) {
		return false
	}
	if !crossed {
		return true
	}
	if _, err := os.Lstat(dst); err == nil && !j.do(func() error { return os.Remove(dst) }) {
		return false // the overwrite was agreed above
	}
	return j.copyItem(src, dst) && j.do(func() error { return os.RemoveAll(src) })
}
