package panel

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// FindPrefix puts the cursor on the first entry whose name starts with
// prefix, ignoring case and Unicode normalization, and reports whether
// there is one. ".." is never found.
func (p *Panel) FindPrefix(prefix string) bool {
	prefix = strings.ToLower(norm.NFC.String(prefix))
	for i, e := range p.Entries {
		if !e.IsUp && strings.HasPrefix(strings.ToLower(norm.NFC.String(e.Name)), prefix) {
			p.Cursor = i
			p.ensureVisible()
			return true
		}
	}
	return false
}
