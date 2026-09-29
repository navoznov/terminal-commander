package keys

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

var t0 = time.Unix(1000, 0)

func key(k tcell.Key, r rune, m tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(k, r, m)
}

func TestEscIsPassedThrough(t *testing.T) {
	var n Normalizer
	ev := n.Feed(key(tcell.KeyEscape, 0, 0), t0)
	if ev.Key() != tcell.KeyEscape {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestEscDigitBecomesFKey(t *testing.T) {
	var n Normalizer
	n.Feed(key(tcell.KeyEscape, 0, 0), t0)
	ev := n.Feed(key(tcell.KeyRune, '5', 0), t0.Add(300*time.Millisecond))
	if ev.Key() != tcell.KeyF5 || ev.Modifiers() != 0 {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestEscZeroIsF10(t *testing.T) {
	var n Normalizer
	n.Feed(key(tcell.KeyEscape, 0, 0), t0)
	if ev := n.Feed(key(tcell.KeyRune, '0', 0), t0); ev.Key() != tcell.KeyF10 {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestEscTimeout(t *testing.T) {
	var n Normalizer
	n.Feed(key(tcell.KeyEscape, 0, 0), t0)
	ev := n.Feed(key(tcell.KeyRune, '5', 0), t0.Add(1500*time.Millisecond))
	if ev.Key() != tcell.KeyRune || ev.Rune() != '5' || ev.Modifiers() != 0 {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestPrefixIsUsedOnce(t *testing.T) {
	var n Normalizer
	n.Feed(key(tcell.KeyEscape, 0, 0), t0)
	n.Feed(key(tcell.KeyRune, '5', 0), t0)
	ev := n.Feed(key(tcell.KeyRune, '5', 0), t0)
	if ev.Key() != tcell.KeyRune || ev.Rune() != '5' {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestAltDigitBecomesFKey(t *testing.T) {
	var n Normalizer
	if ev := n.Feed(key(tcell.KeyRune, '3', tcell.ModAlt), t0); ev.Key() != tcell.KeyF3 || ev.Modifiers() != 0 {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestEscDotBecomesAltDot(t *testing.T) {
	var n Normalizer
	n.Feed(key(tcell.KeyEscape, 0, 0), t0)
	ev := n.Feed(key(tcell.KeyRune, '.', 0), t0)
	if ev.Key() != tcell.KeyRune || ev.Rune() != '.' || ev.Modifiers() != tcell.ModAlt {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestEscF1BecomesAltF1(t *testing.T) {
	var n Normalizer
	n.Feed(key(tcell.KeyEscape, 0, 0), t0)
	ev := n.Feed(key(tcell.KeyF1, 0, 0), t0)
	if ev.Key() != tcell.KeyF1 || ev.Modifiers() != tcell.ModAlt {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestPlainKeysUnchanged(t *testing.T) {
	var n Normalizer
	ev := n.Feed(key(tcell.KeyRune, 'a', 0), t0)
	if ev.Rune() != 'a' || ev.Modifiers() != 0 {
		t.Fatalf("got %s", ev.Name())
	}
}

func TestDisarm(t *testing.T) {
	var n Normalizer
	now := time.Now()
	n.Feed(tcell.NewEventKey(tcell.KeyEscape, 0, 0), now)
	n.Disarm()
	ev := n.Feed(tcell.NewEventKey(tcell.KeyRune, '5', 0), now)
	if ev.Key() != tcell.KeyRune || ev.Modifiers() != 0 {
		t.Fatalf("got %v %v", ev.Key(), ev.Modifiers())
	}
}
