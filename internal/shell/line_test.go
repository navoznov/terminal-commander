package shell

import (
	"slices"
	"strconv"
	"testing"
)

func TestLineEditing(t *testing.T) {
	var l Line
	l.Backspace() // empty line: nothing happens
	l.Insert("ls -")
	l.Insert("ля")
	l.Backspace()
	if l.Text != "ls -л" {
		t.Fatalf("text %q", l.Text)
	}
	l.Clear()
	if l.Text != "" {
		t.Fatalf("text %q after Clear", l.Text)
	}
}

func TestCommitAddsToHistory(t *testing.T) {
	var l Line
	for _, s := range []string{" ls ", "ls", "pwd", "  ", "ls"} {
		l.Text = s
		l.Commit()
	}
	if want := []string{"ls", "pwd", "ls"}; !slices.Equal(l.History(), want) {
		t.Fatalf("history %q, want %q", l.History(), want)
	}
	if l.Text != "" {
		t.Fatalf("text %q after Commit", l.Text)
	}
}

func TestCommitReturnsTrimmedText(t *testing.T) {
	l := Line{Text: "  make test "}
	if got := l.Commit(); got != "make test" {
		t.Fatalf("got %q", got)
	}
}

func TestPrevNext(t *testing.T) {
	var l Line
	l.Prev()
	l.Next()
	if l.Text != "" {
		t.Fatalf("empty history gave %q", l.Text)
	}
	for _, s := range []string{"a", "b", "c"} {
		l.Text = s
		l.Commit()
	}
	steps := []struct {
		f    func()
		want string
	}{
		{l.Prev, "c"}, {l.Prev, "b"}, {l.Prev, "a"}, {l.Prev, "a"},
		{l.Next, "b"}, {l.Next, "c"}, {l.Next, ""}, {l.Next, ""},
		{l.Prev, "c"},
	}
	for i, st := range steps {
		st.f()
		if l.Text != st.want {
			t.Fatalf("step %d: text %q, want %q", i, l.Text, st.want)
		}
	}
}

func TestCommitEndsBrowsing(t *testing.T) {
	var l Line
	for _, s := range []string{"a", "b"} {
		l.Text = s
		l.Commit()
	}
	l.Prev()
	l.Prev() // "a"
	l.Commit()
	l.Prev()
	if l.Text != "a" || !slices.Equal(l.History(), []string{"a", "b", "a"}) {
		t.Fatalf("text %q history %q", l.Text, l.History())
	}
}

func TestHistoryLimit(t *testing.T) {
	var l Line
	for i := 0; i < HistoryLimit+10; i++ {
		l.Text = strconv.Itoa(i)
		l.Commit()
	}
	h := l.History()
	if len(h) != HistoryLimit || h[0] != "10" || h[len(h)-1] != strconv.Itoa(HistoryLimit+9) {
		t.Fatalf("len %d first %q last %q", len(h), h[0], h[len(h)-1])
	}
}
