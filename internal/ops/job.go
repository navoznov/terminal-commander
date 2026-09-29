// Package ops copies, moves and deletes files. An operation runs on the
// caller's goroutine, reports progress and asks the user through its Job,
// and checks for cancellation between files and between copy blocks.
package ops

import "sync/atomic"

// Answer is the user's decision on a question from an operation.
type Answer int

const (
	Cancel  Answer = iota
	Yes            // overwrite, delete, retry
	Skip           // leave this file alone
	All            // overwrite all, delete all
	SkipAll        // skip all conflicts
)

// Progress is a snapshot of a running operation. Done and Total count bytes
// for Copy and sources for the other operations.
type Progress struct {
	File               string // path being processed
	FileDone, FileSize int64  // bytes of the current file
	Done, Total        int64
}

// Job carries an operation's callbacks and its cancel flag. The callbacks
// run on the operation's goroutine; the questions block it until answered.
type Job struct {
	Progress func(Progress)
	// Conflict asks about an existing destination file: Yes (overwrite),
	// Skip, All (overwrite all), SkipAll or Cancel.
	Conflict func(dst string) Answer
	// NotEmpty asks whether to delete a non-empty directory: Yes, All, Skip
	// or Cancel.
	NotEmpty func(dir string) Answer
	// Fail reports an error: Yes (retry), Skip or Cancel.
	Fail func(err error) Answer

	canceled     atomic.Bool
	progress     Progress
	countBytes   bool // Done counts bytes, not sources
	overwriteAll bool
	skipAll      bool
	deleteAll    bool
}

func (j *Job) Cancel()        { j.canceled.Store(true) }
func (j *Job) Canceled() bool { return j.canceled.Load() }

// do runs f until it succeeds, or the user skips or cancels, and reports
// whether f succeeded. Errors after a cancel are not reported.
func (j *Job) do(f func() error) bool {
	for !j.Canceled() {
		err := f()
		if err == nil {
			return true
		}
		if j.Canceled() {
			return false
		}
		switch j.Fail(err) {
		case Yes:
			continue
		case Skip:
			return false
		default:
			j.Cancel()
		}
	}
	return false
}

// fail reports err like do and returns false unless a retry succeeds, which
// for a fixed error it never does.
func (j *Job) fail(err error) bool {
	return j.do(func() error { return err })
}

// overwrite asks whether to replace the existing dst, remembering the "all"
// answers.
func (j *Job) overwrite(dst string) bool {
	switch {
	case j.overwriteAll:
		return true
	case j.skipAll:
		return false
	}
	switch j.Conflict(dst) {
	case Yes:
		return true
	case All:
		j.overwriteAll = true
		return true
	case Skip:
		return false
	case SkipAll:
		j.skipAll = true
		return false
	}
	j.Cancel()
	return false
}
