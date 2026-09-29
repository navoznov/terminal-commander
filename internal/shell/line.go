// Package shell is the command line: its text and history, the built-in cd,
// and running commands in the terminal.
package shell

import "strings"

// HistoryLimit is how many commands the history keeps.
const HistoryLimit = 500

// Line is the command line: the text being typed and the commands run
// before. The cursor is always at the end, as in NC.
type Line struct {
	Text    string
	history []string
	pos     int // history entry shown by Prev/Next; len(history) when none
}

func (l *Line) Insert(s string) { l.Text += s }

func (l *Line) Backspace() {
	if rs := []rune(l.Text); len(rs) > 0 {
		l.Text = string(rs[:len(rs)-1])
	}
}

// Clear empties the line and stops browsing the history.
func (l *Line) Clear() {
	l.Text = ""
	l.pos = len(l.history)
}

// Prev shows the previous (older) command.
func (l *Line) Prev() {
	if l.pos > 0 {
		l.pos--
		l.Text = l.history[l.pos]
	}
}

// Next shows the next (newer) command; after the newest one the line is
// empty.
func (l *Line) Next() {
	if l.pos >= len(l.history) {
		return
	}
	l.pos++
	if l.pos == len(l.history) {
		l.Text = ""
	} else {
		l.Text = l.history[l.pos]
	}
}

// Commit clears the line and returns its text without surrounding spaces,
// adding it to the history unless it is empty or repeats the last command.
func (l *Line) Commit() string {
	s := strings.TrimSpace(l.Text)
	if s != "" && (len(l.history) == 0 || l.history[len(l.history)-1] != s) {
		l.history = append(l.history, s)
		if len(l.history) > HistoryLimit {
			l.history = l.history[len(l.history)-HistoryLimit:]
		}
	}
	l.Clear()
	return s
}

// History returns the commands, oldest first.
func (l *Line) History() []string { return l.history }

// SetHistory replaces the history, keeping the last HistoryLimit commands.
func (l *Line) SetHistory(h []string) {
	l.history = h[max(len(h)-HistoryLimit, 0):]
	l.pos = len(l.history)
}
