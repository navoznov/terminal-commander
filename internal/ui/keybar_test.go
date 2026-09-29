package ui_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestKeyBarKey(t *testing.T) {
	for _, w := range []int{80, 83, 101} {
		for i := 0; i < 10; i++ {
			for _, x := range []int{i * w / 10, (i+1)*w/10 - 1} {
				if got := ui.KeyBarKey(x, w); got != tcell.KeyF1+tcell.Key(i) {
					t.Errorf("w %d x %d: got %v, want F%d", w, x, got, i+1)
				}
			}
		}
	}
}
