package term

import "testing"

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
