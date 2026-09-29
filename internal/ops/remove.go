package ops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// TrashCmd moves a file to the Trash so that Finder can put it back
// (macOS 14+).
var TrashCmd = "/usr/bin/trash"

// Remove deletes paths permanently. A non-empty directory is deleted only
// after NotEmpty agrees. It returns the names of the deleted paths.
func Remove(j *Job, paths []string) []string {
	return j.each(paths, j.removeOne)
}

// Trash moves paths to the Trash, one TrashCmd run per path, and returns the
// names of the moved paths.
func Trash(j *Job, paths []string) []string {
	return j.each(paths, func(p string) bool {
		return j.do(func() error {
			if _, err := os.Lstat(p); err != nil {
				return err
			}
			if err := exec.Command(TrashCmd, p).Run(); err != nil {
				return fmt.Errorf("cannot move %s to the Trash: %w", p, err)
			}
			return nil
		})
	})
}

// each runs f on every path, counting paths as progress, and returns the
// names of the paths f succeeded on.
func (j *Job) each(paths []string, f func(string) bool) []string {
	j.progress.Total = int64(len(paths))
	var done []string
	for _, p := range paths {
		if j.Canceled() {
			break
		}
		j.progress.File = p
		j.Progress(j.progress)
		if f(p) {
			done = append(done, filepath.Base(p))
		}
		j.progress.Done++
		j.Progress(j.progress)
	}
	return done
}

func (j *Job) removeOne(path string) bool {
	if fi, err := os.Lstat(path); err == nil && fi.IsDir() && !emptyDir(path) && !j.deleteDir(path) {
		return false
	}
	return j.do(func() error { return os.RemoveAll(path) })
}

func emptyDir(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	names, _ := f.Readdirnames(1)
	return len(names) == 0
}

// deleteDir asks whether to delete the non-empty directory path,
// remembering "All".
func (j *Job) deleteDir(path string) bool {
	if j.deleteAll {
		return true
	}
	switch j.NotEmpty(path) {
	case Yes:
		return true
	case All:
		j.deleteAll = true
		return true
	case Skip:
		return false
	}
	j.Cancel()
	return false
}
