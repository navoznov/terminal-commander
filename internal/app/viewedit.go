package app

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/shell"
	"github.com/navoznov/terminal-commander/internal/viewer"
)

// view opens the file under the cursor in the viewer. Only regular files
// are opened: opening a FIFO would block.
func (a *App) view() {
	p := a.panels[a.active]
	e := p.Current()
	if e == nil || e.IsDir {
		return
	}
	path := filepath.Join(p.Path, e.Name)
	info, err := os.Stat(path)
	if err == nil && !info.Mode().IsRegular() {
		err = errors.New(e.Name + ": not a regular file")
	}
	var f *os.File
	if err == nil {
		f, err = os.Open(path)
	}
	if err != nil {
		a.report(err)
		return
	}
	v := viewer.New(fs.DisplayPath(path, a.home), f, info.Size())
	v.Push = a.modals.Push
	v.Close = func() { f.Close() }
	a.modals.Push(v)
}

// edit opens the file under the cursor in $EDITOR (nano if unset). If the
// editor fails, its message stays on screen until a key is pressed.
func (a *App) edit() {
	p := a.panels[a.active]
	e := p.Current()
	if e == nil || e.IsDir {
		return
	}
	line := cmp.Or(os.Getenv("EDITOR"), "nano") + " " + shell.Quote(e.Name)
	a.outside(func() {
		if a.console.Run(p.Path, line) != nil {
			a.console.Pause()
		}
	})
	a.reloadPanels()
}
