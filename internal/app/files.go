package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navoznov/terminal-commander/internal/ops"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// resolve turns a typed path into an absolute one: "~" is the home directory
// and relative paths start at dir. A trailing "/" is kept: it tells ops that
// the destination is a directory.
func resolve(dir, home, s string) string {
	p := s
	switch {
	case s == "~":
		p = home
	case strings.HasPrefix(s, "~/"):
		p = home + s[1:]
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	p = filepath.Clean(p)
	if strings.HasSuffix(s, "/") && p != "/" {
		p += "/"
	}
	return p
}

// mkdir asks for a name and creates the directory with its parents, then
// puts the cursor on it.
func (a *App) mkdir() {
	p := a.panels[a.active]
	a.modals.Push(&ui.Dialog{
		Title:   "Make directory",
		Lines:   []string{"Create the directory"},
		Input:   ui.NewInput(""),
		Buttons: []string{"OK", "Cancel"},
		Done: func(b int, name string) {
			if b != 0 || name == "" {
				return
			}
			path := resolve(p.Path, a.home, name)
			if err := os.MkdirAll(path, 0o755); err != nil {
				a.report(err)
				return
			}
			a.reloadPanels()
			if rel, err := filepath.Rel(p.Path, filepath.Clean(path)); err == nil && !strings.HasPrefix(rel, "..") {
				p.Focus(strings.Split(rel, string(filepath.Separator))[0])
			}
		},
	})
}

func (a *App) reloadPanels() {
	for _, p := range a.panels {
		a.report(p.Reload())
	}
}

// sources returns the paths the active panel's operation works on.
func (a *App) sources() []string {
	p := a.panels[a.active]
	var paths []string
	for _, n := range p.Sources() {
		paths = append(paths, filepath.Join(p.Path, n))
	}
	return paths
}

// subject names the sources in a dialog: `"name"` or `N files`.
func subject(paths []string) string {
	if len(paths) == 1 {
		return `"` + filepath.Base(paths[0]) + `"`
	}
	return fmt.Sprintf("%d files", len(paths))
}

// trash asks and moves the sources to the Trash.
func (a *App) trash() {
	srcs := a.sources()
	if srcs == nil {
		return
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Delete",
		Lines:   []string{"Move " + subject(srcs) + " to Trash?"},
		Buttons: []string{"Delete", "Cancel"},
		Done: func(b int, _ string) {
			if b == 0 {
				a.run("Delete", false, func(j *ops.Job) []string { return ops.Trash(j, srcs) })
			}
		},
	})
}

// remove asks and deletes the sources permanently.
func (a *App) remove() {
	srcs := a.sources()
	if srcs == nil {
		return
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Delete",
		Lines:   []string{"Delete " + subject(srcs) + " permanently?"},
		Buttons: []string{"Delete", "Cancel"},
		Danger:  true,
		Done: func(b int, _ string) {
			if b == 0 {
				a.run("Delete", false, func(j *ops.Job) []string { return ops.Remove(j, srcs) })
			}
		},
	})
}

// transfer asks for a destination and copies (F5) or moves (F6) the
// sources there; the other panel's directory is the default.
func (a *App) transfer(move bool) {
	srcs := a.sources()
	if srcs == nil {
		return
	}
	title, prompt, button, do := "Copy", "Copy ", "Copy", ops.Copy
	if move {
		title, prompt, button, do = "Rename/Move", "Rename or move ", "Move", ops.Move
	}
	base := a.panels[a.active].Path
	a.modals.Push(&ui.Dialog{
		Title:   title,
		Lines:   []string{prompt + subject(srcs) + " to:"},
		Input:   ui.NewInput(a.panels[1-a.active].Path),
		Buttons: []string{button, "Cancel"},
		Done: func(b int, text string) {
			if b != 0 || text == "" {
				return
			}
			dst := resolve(base, a.home, text)
			a.run(title, true, func(j *ops.Job) []string { return do(j, srcs, dst) })
		},
	})
}
