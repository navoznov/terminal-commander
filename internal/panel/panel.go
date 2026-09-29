// Package panel holds the state of one file panel and draws it.
package panel

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/navoznov/terminal-commander/internal/fs"
)

type Mode int

const (
	Brief Mode = iota
	Full
)

const briefCols = 3

type Panel struct {
	Path       string
	Entries    []fs.Entry
	Cursor     int
	Top        int
	Mode       Mode
	ShowHidden bool
	Sort       SortMode
	Selected   map[string]bool

	rows int // visible list rows, set by the layout
}

func New() *Panel {
	return &Panel{Selected: map[string]bool{}, rows: 1}
}

func (p *Panel) read(path string) ([]fs.Entry, error) {
	es, err := fs.ReadDir(path, p.ShowHidden)
	if err != nil {
		return nil, err
	}
	SortEntries(es, p.Sort)
	return es, nil
}

// Load switches the panel to path, resetting cursor and selection. On error
// the panel is left unchanged.
func (p *Panel) Load(path string) error {
	path = filepath.Clean(path)
	es, err := p.read(path)
	if err != nil {
		return err
	}
	p.Path, p.Entries = path, es
	p.Selected = map[string]bool{}
	p.Cursor, p.Top = 0, 0
	return nil
}

// Reload re-reads the directory keeping the cursor and selection by name.
// If the directory is gone, the panel climbs to the nearest existing parent.
func (p *Panel) Reload() error {
	dir := p.Path
	for dir != "/" {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			break
		}
		dir = filepath.Dir(dir)
	}
	if dir != p.Path {
		return p.Load(dir)
	}
	name := ""
	if e := p.Current(); e != nil {
		name = e.Name
	}
	es, err := p.read(dir)
	if err != nil {
		return err
	}
	p.Entries = es
	kept := map[string]bool{}
	for _, e := range es {
		if p.Selected[e.Name] {
			kept[e.Name] = true
		}
	}
	p.Selected = kept
	if !p.Focus(name) {
		p.Move(0)
	}
	return nil
}

// Current returns the entry under the cursor, or nil for an empty panel.
func (p *Panel) Current() *fs.Entry {
	if p.Cursor < 0 || p.Cursor >= len(p.Entries) {
		return nil
	}
	return &p.Entries[p.Cursor]
}

// Focus puts the cursor on name and reports whether it was found.
func (p *Panel) Focus(name string) bool {
	for i, e := range p.Entries {
		if e.Name == name {
			p.Cursor = i
			p.ensureVisible()
			return true
		}
	}
	return false
}

// SetRows sets the number of visible list rows (from the layout).
func (p *Panel) SetRows(n int) {
	n = max(n, 1)
	if n == p.rows {
		return
	}
	p.rows = n
	p.Top = 0
	p.ensureVisible()
}

func (p *Panel) SetMode(m Mode) {
	p.Mode = m
	p.Top = 0
	p.ensureVisible()
}

func (p *Panel) capacity() int {
	if p.Mode == Brief {
		return p.rows * briefCols
	}
	return p.rows
}

func (p *Panel) ensureVisible() {
	capacity := p.capacity()
	if p.Mode == Brief {
		// Scroll a whole column at a time, like NC.
		if p.Cursor < p.Top {
			p.Top = p.Cursor / p.rows * p.rows
		}
		if p.Cursor >= p.Top+capacity {
			p.Top = (p.Cursor/p.rows - (briefCols - 1)) * p.rows
		}
	} else {
		if p.Cursor < p.Top {
			p.Top = p.Cursor
		}
		if p.Cursor >= p.Top+capacity {
			p.Top = p.Cursor - capacity + 1
		}
	}
	// Don't leave empty space at the end when the list got shorter.
	overflow := max(len(p.Entries)-capacity, 0)
	if p.Mode == Brief {
		overflow = (overflow + p.rows - 1) / p.rows * p.rows
	}
	p.Top = max(min(p.Top, overflow), 0)
}

// Move shifts the cursor by delta, clamped to the list.
func (p *Panel) Move(delta int) {
	p.Cursor = min(max(p.Cursor+delta, 0), max(len(p.Entries)-1, 0))
	p.ensureVisible()
}

func (p *Panel) Left() {
	if p.Mode == Brief {
		p.Move(-p.rows)
	} else {
		p.PageUp()
	}
}

func (p *Panel) Right() {
	if p.Mode == Brief {
		p.Move(p.rows)
	} else {
		p.PageDown()
	}
}

func (p *Panel) PageUp()   { p.Move(-p.capacity()) }
func (p *Panel) PageDown() { p.Move(p.capacity()) }
func (p *Panel) Home()     { p.Move(-len(p.Entries)) }
func (p *Panel) End()      { p.Move(len(p.Entries)) }

// Enter opens the directory under the cursor. It reports false when the
// cursor is not on a directory.
func (p *Panel) Enter() (bool, error) {
	e := p.Current()
	if e == nil || !e.IsDir {
		return false, nil
	}
	if e.IsUp {
		return true, p.Up()
	}
	return true, p.Load(filepath.Join(p.Path, e.Name))
}

// Up goes to the parent directory and puts the cursor on the one we left.
func (p *Panel) Up() error {
	if p.Path == "/" {
		return nil
	}
	child := filepath.Base(p.Path)
	if err := p.Load(filepath.Dir(p.Path)); err != nil {
		return err
	}
	p.Focus(child)
	return nil
}

// ToggleSelect flips selection of the entry under the cursor (never "..")
// and moves down.
func (p *Panel) ToggleSelect() {
	e := p.Current()
	if e == nil {
		return
	}
	if !e.IsUp {
		if p.Selected[e.Name] {
			delete(p.Selected, e.Name)
		} else {
			p.Selected[e.Name] = true
		}
	}
	p.Move(1)
}

// SelectionStats returns the number of selected entries and the total size
// of the selected files.
func (p *Panel) SelectionStats() (count int, bytes int64) {
	for _, e := range p.Entries {
		if p.Selected[e.Name] {
			count++
			if !e.IsDir {
				bytes += e.Size
			}
		}
	}
	return count, bytes
}

func (p *Panel) SetShowHidden(show bool) error {
	p.ShowHidden = show
	return p.Reload()
}

// SetSort changes the sort mode and re-reads the directory, keeping the
// cursor and selection.
func (p *Panel) SetSort(m SortMode) error {
	p.Sort = m
	return p.Reload()
}
