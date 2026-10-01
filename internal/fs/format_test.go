package fs

import (
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
	d := time.Date(1994, 5, 31, 6, 22, 0, 0, time.UTC)
	if got := FormatDate(d); got != "5-31-94" {
		t.Fatalf("got %q", got)
	}
	d = time.Date(2005, 12, 3, 0, 0, 0, 0, time.UTC)
	if got := FormatDate(d); got != "12-03-05" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatTime(t *testing.T) {
	cases := map[[2]int]string{
		{6, 22}:  "6:22a",
		{16, 54}: "4:54p",
		{0, 5}:   "12:05a",
		{12, 0}:  "12:00p",
	}
	for hm, want := range cases {
		d := time.Date(2000, 1, 1, hm[0], hm[1], 0, 0, time.UTC)
		if got := FormatTime(d); got != want {
			t.Errorf("%v: got %q want %q", hm, got, want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	if got := FormatSize(66294, 9); got != "66294" {
		t.Fatalf("got %q", got)
	}
	if got := FormatSize(12345678901, 9); got != "12056327K" {
		t.Fatalf("got %q", got)
	}
	if got := FormatSize(1<<50, 9); got != "1048576G" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatUnits(t *testing.T) {
	for n, want := range map[int64]string{
		0:                   "0",
		1023:                "1023",
		1024:                "1.0K",
		117760:              "115K",
		118272:              "116K", // 115.5K rounds up
		4718592:             "4.5M",
		10189:               "10K",  // 9.95K: one decimal would give 10.0
		1048575:             "1.0M", // 1023.999K: rounding carries to the next unit
		1288490189:          "1.2G",
		1 << 62:             "4.0E",
		9223372036854775807: "8.0E",
	} {
		if got := FormatUnits(n); got != want {
			t.Errorf("%d: got %q want %q", n, got, want)
		}
	}
}

func TestFormatThousands(t *testing.T) {
	for n, want := range map[int64]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567"} {
		if got := FormatThousands(n); got != want {
			t.Errorf("%d: got %q want %q", n, got, want)
		}
	}
}

func TestDisplayPath(t *testing.T) {
	home := "/Users/rhino"
	for in, want := range map[string]string{
		"/Users/rhino":          "~",
		"/Users/rhino/projects": "~/projects",
		"/Users/rhinoceros":     "/Users/rhinoceros",
		"/tmp":                  "/tmp",
	} {
		if got := DisplayPath(in, home); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
	if got := DisplayPath("/tmp", ""); got != "/tmp" {
		t.Fatalf("got %q", got)
	}
}
