package panel

import "path/filepath"

// SelectMask selects (on) or unselects the files whose names match the shell
// pattern mask, ignoring case. Directories are left alone, as in NC.
func (p *Panel) SelectMask(mask string, on bool) error {
	mask = sortKey(mask)
	if _, err := filepath.Match(mask, ""); err != nil {
		return err
	}
	for _, e := range p.Entries {
		if e.IsDir {
			continue
		}
		if ok, _ := filepath.Match(mask, sortKey(e.Name)); !ok {
			continue
		}
		if on {
			p.Selected[e.Name] = true
		} else {
			delete(p.Selected, e.Name)
		}
	}
	return nil
}

// InvertSelection flips the selection of every file (not directories).
func (p *Panel) InvertSelection() {
	for _, e := range p.Entries {
		if e.IsDir {
			continue
		}
		if p.Selected[e.Name] {
			delete(p.Selected, e.Name)
		} else {
			p.Selected[e.Name] = true
		}
	}
}
