package fs

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FormatDate formats t as NC does: 5-31-94.
func FormatDate(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d", int(t.Month()), t.Day(), t.Year()%100)
}

// FormatTime formats t as NC does: 6:22a, 4:54p.
func FormatTime(t time.Time) string {
	h, suffix := t.Hour(), "a"
	if h >= 12 {
		suffix = "p"
	}
	h %= 12
	if h == 0 {
		h = 12
	}
	return fmt.Sprintf("%d:%02d%s", h, t.Minute(), suffix)
}

// FormatSize formats n bytes in at most width columns, switching to K, M, G,
// T units when the plain number does not fit.
func FormatSize(n int64, width int) string {
	s := strconv.FormatInt(n, 10)
	for _, unit := range []string{"K", "M", "G", "T"} {
		if len(s) <= width {
			return s
		}
		n /= 1024
		s = strconv.FormatInt(n, 10) + unit
	}
	return s
}

// FormatUnits formats n bytes in the largest 1024-based unit that keeps the
// number under 1024, as ls -h does: 1023, 115K, 4.5M, 12G. Below 10 units
// it shows one decimal.
func FormatUnits(n int64) string {
	if n < 1024 {
		return strconv.FormatInt(n, 10)
	}
	v := float64(n)
	units := []string{"K", "M", "G", "T", "P", "E"}
	for i, unit := range units {
		v /= 1024
		if tenths := math.Round(v * 10); tenths < 100 {
			return strconv.FormatFloat(tenths/10, 'f', 1, 64) + unit
		}
		if r := math.Round(v); r < 1024 || i == len(units)-1 {
			return strconv.FormatFloat(r, 'f', 0, 64) + unit
		}
	}
	panic("unreachable")
}

// FormatThousands formats n with comma separators: 1,234,567.
func FormatThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// DisplayPath abbreviates the home directory to ~.
func DisplayPath(path, home string) string {
	if home == "" || home == "/" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}
