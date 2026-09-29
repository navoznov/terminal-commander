package viewer

import (
	"slices"
	"strings"
	"testing"
)

func newLayout(s string, width int, wrap, hex bool) *layout {
	return &layout{r: strings.NewReader(s), size: int64(len(s)), width: width, wrap: wrap, hex: hex}
}

// starts walks the rows with nextRow from 0 and returns their starts.
func starts(l *layout) []int64 {
	var ss []int64
	for s := int64(0); s < l.size; s = l.nextRow(s) {
		ss = append(ss, s)
	}
	return ss
}

func TestReadLine(t *testing.T) {
	l := newLayout("ab\r\ncd\n\nlast", 80, false, false)
	for _, tt := range []struct {
		start int64
		line  string
		next  int64
	}{
		{0, "ab", 4}, {4, "cd", 7}, {7, "", 8}, {8, "last", 12},
	} {
		line, next := l.readLine(tt.start)
		if string(line) != tt.line || next != tt.next {
			t.Errorf("readLine(%d) = %q, %d", tt.start, line, next)
		}
	}
}

func TestLongLineIsCut(t *testing.T) {
	s := strings.Repeat("x", 2*maxLine+10)
	l := newLayout(s, 80, false, false)
	if got, want := starts(l), []int64{0, maxLine, 2 * maxLine}; !slices.Equal(got, want) {
		t.Fatalf("starts %v, want %v", got, want)
	}
	if got := l.lineStart(2*maxLine + 5); got != 2*maxLine {
		t.Fatalf("lineStart %d", got)
	}
	// A newline right after a full piece does not make an empty line.
	l = newLayout(strings.Repeat("y", maxLine)+"\nz", 80, false, false)
	if got, want := starts(l), []int64{0, maxLine + 1}; !slices.Equal(got, want) {
		t.Fatalf("starts %v, want %v", got, want)
	}
}

func TestLineStart(t *testing.T) {
	l := newLayout("ab\ncd\n\nef", 80, false, false)
	for off, want := range []int64{0, 0, 0, 3, 3, 3, 6, 7, 7} {
		if got := l.lineStart(int64(off)); got != want {
			t.Errorf("lineStart(%d) = %d, want %d", off, got, want)
		}
	}
}

func TestCells(t *testing.T) {
	cs := cells([]byte("a\tб\xff\x01界"))
	var b strings.Builder
	for _, c := range cs {
		b.WriteRune(c.r)
	}
	if got := b.String(); got != "a       б··界" {
		t.Fatalf("got %q", got)
	}
	if last := cs[len(cs)-1]; last.w != 2 || last.off != 6 {
		t.Fatalf("last cell %+v", last)
	}
}

func TestWrapRows(t *testing.T) {
	l := newLayout("abcdefgh\n\n界界界\nxy", 3, true, false)
	// "abc" "def" "gh" | "" | "界" "界" "界" | "xy"
	want := []int64{0, 3, 6, 9, 10, 13, 16, 20}
	if got := starts(l); !slices.Equal(got, want) {
		t.Fatalf("starts %v, want %v", got, want)
	}
	for i, s := range want[1:] {
		if got := l.prevRow(s); got != want[i] {
			t.Errorf("prevRow(%d) = %d, want %d", s, got, want[i])
		}
	}
	if got := l.rowStart(7); got != 6 {
		t.Fatalf("rowStart(7) = %d", got)
	}
}

func TestWrapCutsTab(t *testing.T) {
	cs := cells([]byte("abcdef\tg"))
	// "abcdef" + 2 tab spaces; w = 7 cuts the tab, "g" starts the next row
	cuts := wrapCuts(cs, 7)
	if len(cuts) != 2 || cs[cuts[1]].r != 'g' {
		t.Fatalf("cuts %v", cuts)
	}
}

func TestHexRows(t *testing.T) {
	l := newLayout(strings.Repeat("z", 40), 80, false, true)
	if got, want := starts(l), []int64{0, 16, 32}; !slices.Equal(got, want) {
		t.Fatalf("starts %v", got)
	}
	if l.rowStart(33) != 32 || l.prevRow(16) != 0 {
		t.Fatal("hex rowStart/prevRow")
	}
}

func TestRowsFrom(t *testing.T) {
	l := newLayout("a\nb\nc\nd\n", 80, false, false)
	rows, next := l.rowsFrom(2, 2)
	if len(rows) != 2 || rows[0].start != 2 || rows[1].start != 4 || next != 6 {
		t.Fatalf("rows %+v next %d", rows, next)
	}
	if rows, next := l.rowsFrom(6, 5); len(rows) != 1 || next != 8 {
		t.Fatalf("rows %+v next %d", rows, next)
	}
}
