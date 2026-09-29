package app

import (
	"os"
	"path/filepath"
	"strings"

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
