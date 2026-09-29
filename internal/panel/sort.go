package panel

import (
	"sort"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/navoznov/terminal-commander/internal/fs"
)

type SortMode int

const (
	SortName SortMode = iota
	SortExt
	SortTime
	SortSize
	Unsorted
)

func sortKey(s string) string { return strings.ToLower(norm.NFC.String(s)) }

// ext returns the extension of name without the dot; ".zshrc" has none.
func ext(name string) string {
	if i := strings.LastIndexByte(name, '.'); i > 0 {
		return name[i+1:]
	}
	return ""
}

// SortEntries puts ".." first. Unsorted keeps the rest as is; other modes put
// directories before files and order each group by the mode, then by
// case-insensitive name. Time puts newest first, Size largest first
// (directories are ordered by name).
func SortEntries(es []fs.Entry, mode SortMode) {
	if mode == Unsorted {
		return
	}
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.IsUp != b.IsUp {
			return a.IsUp
		}
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		switch mode {
		case SortExt:
			if ea, eb := sortKey(ext(a.Name)), sortKey(ext(b.Name)); ea != eb {
				return ea < eb
			}
		case SortTime:
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.After(b.ModTime)
			}
		case SortSize:
			if !a.IsDir && a.Size != b.Size {
				return a.Size > b.Size
			}
		}
		if ka, kb := sortKey(a.Name), sortKey(b.Name); ka != kb {
			return ka < kb
		}
		return a.Name < b.Name
	})
}
