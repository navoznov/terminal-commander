package term

import (
	"strings"
	"unicode/utf8"

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

// Wrap splits s (NFC) into lines of at most w columns, breaking at the last
// space that fits or, inside a long word, anywhere.
func Wrap(s string, w int) []string {
	s = norm.NFC.String(s)
	w = max(w, 1)
	var lines []string
	for runewidth.StringWidth(s) > w {
		cut, used := 0, 0
		for i, r := range s {
			rw := runewidth.RuneWidth(r)
			if used+rw > w {
				break
			}
			used += rw
			cut = i + utf8.RuneLen(r)
		}
		if cut == 0 {
			_, cut = utf8.DecodeRuneInString(s)
		}
		if s[cut] != ' ' {
			if sp := strings.LastIndexByte(s[:cut], ' '); sp > 0 {
				cut = sp
			}
		}
		lines = append(lines, strings.TrimRight(s[:cut], " "))
		s = strings.TrimLeft(s[cut:], " ")
	}
	return append(lines, s)
}
