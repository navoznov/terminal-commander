package panel

import (
	"sort"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/navoznov/terminal-commander/internal/fs"
)

// SortEntries puts ".." first, then directories, then files, each group by
// case-insensitive name.
func SortEntries(es []fs.Entry) {
	key := func(e fs.Entry) string { return strings.ToLower(norm.NFC.String(e.Name)) }
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.IsUp != b.IsUp {
			return a.IsUp
		}
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		if ka, kb := key(a), key(b); ka != kb {
			return ka < kb
		}
		return a.Name < b.Name
	})
}
