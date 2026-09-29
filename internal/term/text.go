package term

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/text/unicode/norm"
)

// Width returns the number of terminal columns s occupies.
func Width(s string) int {
	return runewidth.StringWidth(norm.NFC.String(s))
}

// Fit normalizes s to NFC and makes it exactly w columns wide: padded with
// spaces, or cut with '}' in the last column when it does not fit, as NC does.
func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = norm.NFC.String(s)
	if sw := runewidth.StringWidth(s); sw <= w {
		return s + strings.Repeat(" ", w-sw)
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if used+rw > w-1 {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	b.WriteString(strings.Repeat(" ", w-1-used))
	b.WriteRune('}')
	return b.String()
}

// Tail returns the longest suffix of s (NFC) that fits in w columns.
func Tail(s string, w int) string {
	rs := []rune(norm.NFC.String(s))
	used := 0
	i := len(rs)
	for i > 0 {
		rw := runewidth.RuneWidth(rs[i-1])
		if used+rw > w {
			break
		}
		used += rw
		i--
	}
	return string(rs[i:])
}
