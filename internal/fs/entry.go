// Package fs reads directories and formats file data the way NC shows it.
package fs

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry is one line of a file panel.
type Entry struct {
	Name    string
	IsDir   bool
	IsUp    bool // the ".." entry
	IsLink  bool
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
}

// ReadDir lists path. ".." comes first unless path is "/". Dotfiles are
// skipped unless showHidden. Symlinks are described by their targets; broken
// links look like files. Entries after ".." are not sorted.
func ReadDir(path string, showHidden bool) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	des, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	var out []Entry
	if path != "/" {
		up := Entry{Name: "..", IsDir: true, IsUp: true}
		if info, err := os.Stat(filepath.Join(path, "..")); err == nil {
			up.ModTime = info.ModTime()
		}
		out = append(out, up)
	}
	for _, de := range des {
		name := de.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		info, err := os.Lstat(full)
		if err != nil {
			continue // vanished after ReadDir
		}
		e := Entry{Name: name}
		if info.Mode()&os.ModeSymlink != 0 {
			e.IsLink = true
			if target, err := os.Stat(full); err == nil {
				info = target
			}
		}
		e.IsDir = info.IsDir()
		e.Size = info.Size()
		e.ModTime = info.ModTime()
		e.Mode = info.Mode()
		out = append(out, e)
	}
	return out, nil
}
