// Package keys turns raw terminal key events into the keys the app acts on.
package keys

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

// PrefixTimeout is how long Esc keeps acting as an Alt (Option) prefix.
const PrefixTimeout = time.Second

// Normalizer implements the Esc prefix: the key that follows Esc within
// PrefixTimeout gets the Alt modifier, and Alt+digit becomes F1…F10.
// Esc itself is passed through so it still closes dialogs immediately.
type Normalizer struct {
	armed   bool
	armedAt time.Time
}

func (n *Normalizer) Feed(ev *tcell.EventKey, now time.Time) *tcell.EventKey {
	if ev.Key() == tcell.KeyEscape && ev.Modifiers() == 0 {
		n.armed, n.armedAt = true, now
		return ev
	}
	if n.armed {
		n.armed = false
		if now.Sub(n.armedAt) <= PrefixTimeout {
			ev = tcell.NewEventKey(ev.Key(), ev.Rune(), ev.Modifiers()|tcell.ModAlt)
		}
	}
	return altDigitToF(ev)
}

func altDigitToF(ev *tcell.EventKey) *tcell.EventKey {
	r := ev.Rune()
	if ev.Key() != tcell.KeyRune || ev.Modifiers()&tcell.ModAlt == 0 || r < '0' || r > '9' {
		return ev
	}
	n := int(r - '0')
	if n == 0 {
		n = 10
	}
	return tcell.NewEventKey(tcell.KeyF1+tcell.Key(n-1), 0, ev.Modifiers()&^tcell.ModAlt)
}
