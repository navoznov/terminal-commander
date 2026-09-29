package term

import (
	"strings"
	"testing"
)

func TestFitPads(t *testing.T) {
	if got := Fit("abc", 5); got != "abc  " {
		t.Fatalf("got %q", got)
	}
}

func TestFitTruncatesWithBrace(t *testing.T) {
	if got := Fit("abcdef", 4); got != "abc}" {
		t.Fatalf("got %q", got)
	}
}

func TestFitZeroWidth(t *testing.T) {
	if got := Fit("abc", 0); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestFitNormalizesNFD(t *testing.T) {
	nfd := "йка" // «йка», й в NFD
	if got := Fit(nfd, 4); got != "йка " {
		t.Fatalf("got %q", got)
	}
	if w := Width(nfd); w != 3 {
		t.Fatalf("width %d", w)
	}
}

func TestFitWideRuneAtBoundary(t *testing.T) {
	if got := Fit("😀😀x", 3); got != "😀}" {
		t.Fatalf("got %q", got)
	}
	if got := Fit("😀😀x", 4); got != "😀 }" {
		t.Fatalf("got %q", got)
	}
}

func TestTail(t *testing.T) {
	if got := Tail("/Users/rhino/projects", 8); got != "projects" {
		t.Fatalf("got %q", got)
	}
	if got := Tail("abc", 10); got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestWrapAtSpace(t *testing.T) {
	got := Wrap("hello world foo", 11)
	if len(got) != 2 || got[0] != "hello world" || got[1] != "foo" {
		t.Fatalf("got %q", got)
	}
}

func TestWrapBreaksLongWord(t *testing.T) {
	got := Wrap("abcdefghij", 4)
	if len(got) != 3 || got[0] != "abcd" || got[1] != "efgh" || got[2] != "ij" {
		t.Fatalf("got %q", got)
	}
}

func TestWrapShort(t *testing.T) {
	if got := Wrap("", 5); len(got) != 1 || got[0] != "" {
		t.Fatalf("got %q", got)
	}
}

func TestWrapLinesFit(t *testing.T) {
	s := "open /Users/nc/" + strings.Repeat("очень-длинная-папка/", 10) + ": permission denied"
	for _, l := range Wrap(s, 60) {
		if Width(l) > 60 {
			t.Fatalf("line too wide: %q", l)
		}
	}
}
