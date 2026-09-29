package panel

import (
	"testing"

	"github.com/navoznov/terminal-commander/internal/fs"
)

func TestFindPrefix(t *testing.T) {
	p := New()
	p.Entries = []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true},
		{Name: "alpha", IsDir: true},
		{Name: "Beta.txt"},
		{Name: "bravo"},
		{Name: "Йод"}, // "Йод" in NFD, as some macOS apps name files
	}
	p.Cursor = 3
	tests := []struct {
		prefix string
		ok     bool
		cursor int
	}{
		{"b", true, 2},
		{"BR", true, 3},
		{"x", false, 3}, // the cursor stays
		{".", false, 3}, // ".." is not found
		{"йо", true, 4}, // any case, NFC or NFD
		{"ALPHA", true, 1},
	}
	for _, tt := range tests {
		if ok := p.FindPrefix(tt.prefix); ok != tt.ok || p.Cursor != tt.cursor {
			t.Errorf("FindPrefix(%q) = %v, cursor %d; want %v, %d", tt.prefix, ok, p.Cursor, tt.ok, tt.cursor)
		}
	}
}
