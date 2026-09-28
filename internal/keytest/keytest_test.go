package keytest

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestDescribe(t *testing.T) {
	raw := tcell.NewEventKey(tcell.KeyRune, '5', tcell.ModAlt)
	norm := tcell.NewEventKey(tcell.KeyF5, 0, 0)
	got := Describe(raw, norm)
	if !strings.Contains(got, "Alt+Rune[5]") || !strings.HasSuffix(got, "-> F5") {
		t.Fatalf("got %q", got)
	}
}
