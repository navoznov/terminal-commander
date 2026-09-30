package ops

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// script is a scripted user: each question takes the next answer and is
// recorded in asked. Without answers left it cancels.
type script struct {
	answers []Answer
	asked   []string
}

func (s *script) next(q string) Answer {
	s.asked = append(s.asked, q)
	if len(s.answers) == 0 {
		return Cancel
	}
	a := s.answers[0]
	s.answers = s.answers[1:]
	return a
}

func newJob(s *script) *Job {
	return &Job{
		Progress: func(Progress) {},
		Conflict: func(dst string) Answer { return s.next("conflict " + filepath.Base(dst)) },
		NotEmpty: func(dir string) Answer { return s.next("not empty " + filepath.Base(dir)) },
		Fail: func(err error, retry bool) Answer {
			if !retry {
				return s.next("error " + err.Error())
			}
			return s.next("fail " + err.Error())
		},
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

var stamp = time.Date(1994, 5, 31, 6, 22, 0, 0, time.UTC)

// write creates path with content, mode and the fixed stamp time, making
// parent directories.
func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0o755))
	must(t, os.WriteFile(path, []byte(content), mode))
	must(t, os.Chmod(path, mode))
	must(t, os.Chtimes(path, stamp, stamp))
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	must(t, err)
	return string(b)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
