package app

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navoznov/terminal-commander/internal/ops"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// progressDelay is how long an operation runs before its window shows.
const progressDelay = 300 * time.Millisecond

// op is a file operation running in the background.
type op struct {
	job     *ops.Job
	view    *ui.Progress
	src     *panel.Panel
	timer   *time.Timer
	fileBar bool

	mu     sync.Mutex
	last   ops.Progress
	queued atomic.Bool // a progress update waits in a.calls
}

// post runs f on the UI goroutine.
func (a *App) post(f func()) { a.calls <- f }

// run starts work in the background under a progress window titled title,
// with a bar for the current file if fileBar. When work ends, the names it
// returns are unselected in the active panel and both panels are re-read.
func (a *App) run(title string, fileBar bool, work func(*ops.Job) []string) {
	o := &op{src: a.panels[a.active], fileBar: fileBar}
	bars := 1
	if fileBar {
		bars = 2
	}
	o.view = &ui.Progress{Title: title, Bars: make([]float64, bars), Hidden: true, Cancel: func() { o.job.Cancel() }}
	o.job = &ops.Job{
		Progress: func(p ops.Progress) {
			o.mu.Lock()
			o.last = p
			o.mu.Unlock()
			if o.queued.CompareAndSwap(false, true) {
				a.post(func() { o.queued.Store(false); a.showProgress(o) })
			}
		},
		Conflict: a.askConflict,
		NotEmpty: a.askNotEmpty,
		Fail:     a.askFail,
	}
	a.op = o
	a.modals.Push(o.view)
	o.timer = time.AfterFunc(progressDelay, func() { a.post(func() { o.view.Hidden = false }) })
	go func() {
		done := work(o.job)
		a.post(func() { a.finish(o, done) })
	}()
}

func (a *App) showProgress(o *op) {
	o.mu.Lock()
	p := o.last
	o.mu.Unlock()
	o.view.File = filepath.Base(p.File)
	total := fraction(p.Done, p.Total)
	if o.fileBar {
		o.view.Bars = []float64{fraction(p.FileDone, p.FileSize), total}
	} else {
		o.view.Bars = []float64{total}
	}
}

func fraction(n, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(n) / float64(total)
}

func (a *App) finish(o *op, done []string) {
	o.timer.Stop()
	a.modals.Remove(o.view)
	a.op = nil
	for _, n := range done {
		delete(o.src.Selected, n)
	}
	a.reloadPanels()
}

// ask shows d from the operation's goroutine and waits for the answer:
// answers[i] for button i, Cancel for Esc.
func (a *App) ask(d *ui.Dialog, answers []ops.Answer) ops.Answer {
	ch := make(chan ops.Answer, 1)
	d.Danger = true
	d.Done = func(b int, _ string) {
		if b < 0 {
			ch <- ops.Cancel
		} else {
			ch <- answers[b]
		}
	}
	a.post(func() { a.modals.Push(d) })
	return <-ch
}

func (a *App) askConflict(dst string) ops.Answer {
	return a.ask(&ui.Dialog{
		Title:   "Warning",
		Lines:   append([]string{"File already exists:"}, term.Wrap(dst, 60)...),
		Buttons: []string{"Overwrite", "Skip", "Overwrite all", "Skip all", "Cancel"},
	}, []ops.Answer{ops.Yes, ops.Skip, ops.All, ops.SkipAll, ops.Cancel})
}

func (a *App) askNotEmpty(dir string) ops.Answer {
	return a.ask(&ui.Dialog{
		Title:   "Delete",
		Lines:   []string{"Directory " + filepath.Base(dir) + " is not empty.", "Delete it?"},
		Buttons: []string{"Delete", "All", "Skip", "Cancel"},
	}, []ops.Answer{ops.Yes, ops.All, ops.Skip, ops.Cancel})
}

func (a *App) askFail(err error, retry bool) ops.Answer {
	d := &ui.Dialog{Title: "Error", Lines: term.Wrap(err.Error(), 60)}
	if !retry {
		d.Buttons = []string{"Skip", "Cancel"}
		return a.ask(d, []ops.Answer{ops.Skip, ops.Cancel})
	}
	d.Buttons = []string{"Retry", "Skip", "Cancel"}
	return a.ask(d, []ops.Answer{ops.Yes, ops.Skip, ops.Cancel})
}
