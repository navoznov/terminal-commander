# Terminal Commander — этапы 1–2: каркас и панели. План реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Запускаемый `tc`: экран в цветах NC, две файловые панели (краткий
и полный режимы) с навигацией, выделением, скрытыми файлами, мини-статусом,
строкой клавиш, Esc-префиксом и режимом `tc --keytest`.

**Architecture:** Свой слой отрисовки поверх голого tcell (`internal/term`),
чистые модели без UI (`internal/fs`, `internal/keys`, модель в
`internal/panel`), склейка в `internal/app`. Внешний вид проверяется
golden-тестами на `tcell.SimulationScreen`.

**Tech Stack:** Go 1.27, `github.com/gdamore/tcell/v2` (v2.13.x),
`github.com/mattn/go-runewidth`, `golang.org/x/text/unicode/norm`.

**Spec:** `docs/superpowers/specs/2026-09-28-terminal-commander-design.md`

Этапы 3–6 спеки (диалоги и меню, операции с файлами, командная строка,
просмотрщик/конфиг/мышь) — отдельные планы после этого.

## Global Constraints

- Модуль: `github.com/navoznov/terminal-commander`; бинарник `tc`; вход `cmd/tc`.
- Палитра EGA (RGB): black `#000000`, blue `#0000AA`, cyan `#00AAAA`, red `#AA0000`, lightgray `#AAAAAA`, brightcyan `#55FFFF`, yellow `#FFFF55`, white `#FFFFFF`. Цвета задаются только RGB; понижение до 256 цветов делает tcell (он учитывает `COLORTERM` и terminfo).
- `internal/fs` и `internal/keys` не рисуют ничего; `internal/panel` не знает о другой панели.
- Минимальный размер экрана 80×24; меньше — надпись `Window too small`.
- Строка клавиш: `1Help 2Menu 3View 4Edit 5Copy 6RenMov 7Mkdir 8Delete 9PullDn 10Quit`.
- Дата `M-DD-YY` (`5-31-94`), время `h:mma|p` (`6:22a`), обрезка имён символом `}`.
- Скрытые файлы по умолчанию скрыты; переключение Alt-. и `Esc` `.`.
- Имена файлов перед выводом нормализуются в NFC (macOS хранит многие имена в NFD).
- Коммиты — Conventional Commits, на английском, **без** строк `Co-Authored-By` и пометок «Generated with Claude Code».
- Работа в ветке `feature/v1`; в `main` — только через PR.

## Review Focus

1. **Имена в NFD** (кириллица из Finder: `й` = `и` + U+0306) — отображаются одним символом, колонки не съезжают. Тест: `TestFitNormalizesNFD` (Task 1), `TestBriefNFDName` (Task 5).
2. **Каталог без прав на чтение** — Enter не ломает панель: она остаётся на месте, ошибка показывается в строке команд. Тест: `TestEnterUnreadableDirShowsError` (Task 6).
3. **Текущий каталог удалён извне** — Control-R поднимает панель к ближайшему существующему родителю. Тест: `TestReloadClimbsWhenDirRemoved` (Task 4).
4. **Изменение размера окна / очень маленькое окно** — нет паники, курсор остаётся видимым. Тесты: `TestSetRowsKeepsCursorVisible` (Task 4), `TestSmallWindow` (Task 6).
5. **Пустой список** (пустой `/`-подобный каталог без `..`) — Enter, Space, стрелки, мини-статус не паникуют. Тест: `TestEmptyPanel` (Task 4), `TestDrawEmptyPanel` (Task 5).

---

## Карта файлов

```
go.mod, go.sum, Makefile, .gitignore
cmd/tc/main.go                    флаги, запуск app или keytest, recover
internal/term/palette.go          цвета EGA, стили ролей, StyleCode для golden
internal/term/text.go             Width, Fit, Tail (NFC + runewidth)
internal/term/canvas.go           Canvas: Put, Text, Fill, HLine, VLine, Box
internal/term/termtest/termtest.go  NewScreen, Dump, Golden (-update)
internal/keys/keys.go             Normalizer (Esc-префикс, Alt+цифра → F)
internal/keytest/keytest.go       режим --keytest
internal/fs/entry.go              Entry, ReadDir
internal/fs/format.go             FormatDate/Time/Size/Thousands, DisplayPath
internal/panel/panel.go           модель панели: загрузка, курсор, выделение
internal/panel/sort.go            SortEntries
internal/panel/render.go          Draw (краткий/полный режим, мини-статус)
internal/app/app.go               App: состояние, Run, HandleEvent, клавиши
internal/app/draw.go              раскладка, командная строка, строка клавиш
```

---

### Task 1: Каркас модуля и слой отрисовки `term`

**Files:**
- Create: `go.mod`, `.gitignore`, `Makefile`
- Create: `internal/term/palette.go`, `internal/term/text.go`, `internal/term/canvas.go`
- Create: `internal/term/termtest/termtest.go`
- Test: `internal/term/text_test.go`, `internal/term/canvas_test.go`

**Interfaces:**
- Consumes: —
- Produces:
  - цвета `term.Black, Blue, Cyan, Red, LightGray, BrightCyan, Yellow, White tcell.Color`
  - стили `term.PanelStyle, HeaderStyle, CursorStyle, SelectedStyle, SelectedCursorStyle, ActiveTitleStyle, CmdLineStyle, KeyNumStyle, KeyLabelStyle, ErrorStyle tcell.Style`
  - `term.StyleCode(st tcell.Style) rune`
  - `term.Width(s string) int`, `term.Fit(s string, w int) string`, `term.Tail(s string, w int) string`
  - `type term.Canvas struct{ Screen tcell.Screen }` с методами `Put(x, y int, r rune, st tcell.Style)`, `Text(x, y int, s string, max int, st tcell.Style) int`, `Fill(x, y, w, h int, r rune, st tcell.Style)`, `HLine(x, y, w int, r rune, st tcell.Style)`, `VLine(x, y, h int, r rune, st tcell.Style)`, `Box(x, y, w, h int, st tcell.Style)`
  - `termtest.NewScreen(t *testing.T, w, h int) tcell.SimulationScreen`, `termtest.Dump(s tcell.SimulationScreen) string`, `termtest.Golden(t *testing.T, name, got string)`

- [ ] **Step 1: Инициализировать модуль и зависимости**

```bash
cd /Users/rhino/projects/terminal-commander
go mod init github.com/navoznov/terminal-commander
go get github.com/gdamore/tcell/v2@v2.13 github.com/mattn/go-runewidth@latest golang.org/x/text@latest
```

Создать `.gitignore`:

```
/tc
```

Создать `Makefile` (отступы — табы):

```make
.PHONY: build test install

build:
	go build -o tc ./cmd/tc

test:
	go test ./...

install: build
	mkdir -p $(HOME)/bin
	cp tc $(HOME)/bin/tc
```

- [ ] **Step 2: Написать падающие тесты для `text.go`**

`internal/term/text_test.go`:

```go
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
```

- [ ] **Step 3: Запустить — должен упасть**

Run: `go test ./internal/term/`
Expected: FAIL — `undefined: Fit` и т. п.

- [ ] **Step 4: Реализовать `text.go`**

`internal/term/text.go`:

```go
package term

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/text/unicode/norm"
)

// Width returns the number of terminal columns s occupies.
func Width(s string) int {
	return runewidth.StringWidth(norm.NFC.String(s))
}

// Fit normalizes s to NFC and makes it exactly w columns wide: padded with
// spaces, or cut with '}' in the last column when it does not fit, as NC does.
func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = norm.NFC.String(s)
	if sw := runewidth.StringWidth(s); sw <= w {
		return s + strings.Repeat(" ", w-sw)
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if used+rw > w-1 {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	b.WriteString(strings.Repeat(" ", w-1-used))
	b.WriteRune('}')
	return b.String()
}

// Tail returns the longest suffix of s (NFC) that fits in w columns.
func Tail(s string, w int) string {
	rs := []rune(norm.NFC.String(s))
	used := 0
	i := len(rs)
	for i > 0 {
		rw := runewidth.RuneWidth(rs[i-1])
		if used+rw > w {
			break
		}
		used += rw
		i--
	}
	return string(rs[i:])
}
```

- [ ] **Step 5: Запустить — должен пройти**

Run: `go test ./internal/term/`
Expected: PASS

- [ ] **Step 6: Реализовать палитру и `termtest`**

`internal/term/palette.go`:

```go
package term

import "github.com/gdamore/tcell/v2"

// EGA colors used by Norton Commander. tcell downgrades them to the nearest
// palette colors on terminals without true color.
var (
	Black      = tcell.NewRGBColor(0x00, 0x00, 0x00)
	Blue       = tcell.NewRGBColor(0x00, 0x00, 0xAA)
	Cyan       = tcell.NewRGBColor(0x00, 0xAA, 0xAA)
	Red        = tcell.NewRGBColor(0xAA, 0x00, 0x00)
	LightGray  = tcell.NewRGBColor(0xAA, 0xAA, 0xAA)
	BrightCyan = tcell.NewRGBColor(0x55, 0xFF, 0xFF)
	Yellow     = tcell.NewRGBColor(0xFF, 0xFF, 0x55)
	White      = tcell.NewRGBColor(0xFF, 0xFF, 0xFF)
)

func style(fg, bg tcell.Color) tcell.Style {
	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

// Color roles.
var (
	PanelStyle          = style(BrightCyan, Blue)
	HeaderStyle         = style(Yellow, Blue)
	CursorStyle         = style(Black, Cyan)
	SelectedStyle       = style(Yellow, Blue)
	SelectedCursorStyle = style(Yellow, Cyan)
	ActiveTitleStyle    = style(Black, Cyan)
	CmdLineStyle        = style(LightGray, Black)
	KeyNumStyle         = style(LightGray, Black)
	KeyLabelStyle       = style(Black, Cyan)
	ErrorStyle          = style(White, Red)
)

var styleCodes = map[[2]tcell.Color]rune{}

func init() {
	for _, sc := range []struct {
		st   tcell.Style
		code rune
	}{
		{PanelStyle, 'p'},
		{HeaderStyle, 'y'},       // also SelectedStyle
		{CursorStyle, 'c'},       // also ActiveTitleStyle, KeyLabelStyle
		{SelectedCursorStyle, 'Y'},
		{CmdLineStyle, 'k'},      // also KeyNumStyle
		{ErrorStyle, 'r'},
		{tcell.StyleDefault, '.'},
	} {
		fg, bg, _ := sc.st.Decompose()
		styleCodes[[2]tcell.Color{fg, bg}] = sc.code
	}
}

// StyleCode returns a one-letter code of the color role of st, used by
// golden screen dumps. Unknown combinations give '?'.
func StyleCode(st tcell.Style) rune {
	fg, bg, _ := st.Decompose()
	if c, ok := styleCodes[[2]tcell.Color{fg, bg}]; ok {
		return c
	}
	return '?'
}
```

`internal/term/termtest/termtest.go`:

```go
// Package termtest provides a simulated screen and golden-file helpers.
package termtest

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

var update = flag.Bool("update", false, "rewrite golden files")

// NewScreen returns an initialized simulation screen of w×h cells.
func NewScreen(t *testing.T, w, h int) tcell.SimulationScreen {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	s.SetSize(w, h)
	t.Cleanup(s.Fini)
	return s
}

// Dump shows the screen and returns its characters, a blank line, and a grid
// of term.StyleCode letters for every cell.
func Dump(s tcell.SimulationScreen) string {
	s.Show()
	cells, w, h := s.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := ' '
			if rs := cells[y*w+x].Runes; len(rs) > 0 {
				r = rs[0]
			}
			b.WriteRune(r)
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.WriteRune(term.StyleCode(cells[y*w+x].Style))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Golden compares got with testdata/<name>.golden; with -update it rewrites
// the file instead.
func Golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test with -update to create it)", err)
	}
	if string(want) != got {
		t.Errorf("screen differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
```

- [ ] **Step 7: Написать падающие тесты для `Canvas`**

`internal/term/canvas_test.go`:

```go
package term_test

import (
	"strings"
	"testing"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

func TestBox(t *testing.T) {
	s := termtest.NewScreen(t, 6, 3)
	term.Canvas{Screen: s}.Box(0, 0, 6, 3, term.PanelStyle)
	lines := strings.Split(termtest.Dump(s), "\n")
	want := []string{"╔════╗", "║    ║", "╚════╝"}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d: got %q want %q", i, lines[i], w)
		}
	}
	if lines[4] != "pppppp" {
		t.Fatalf("styles: %q", lines[4])
	}
}

func TestTextClips(t *testing.T) {
	s := termtest.NewScreen(t, 10, 1)
	c := term.Canvas{Screen: s}
	if n := c.Text(0, 0, "hello", 3, term.PanelStyle); n != 3 {
		t.Fatalf("used %d", n)
	}
	if got := strings.Split(termtest.Dump(s), "\n")[0]; got != "hel       " {
		t.Fatalf("got %q", got)
	}
}

func TestFillAndLines(t *testing.T) {
	s := termtest.NewScreen(t, 4, 3)
	c := term.Canvas{Screen: s}
	c.Fill(0, 0, 4, 3, '.', term.CmdLineStyle)
	c.HLine(0, 1, 4, '─', term.PanelStyle)
	c.VLine(2, 0, 3, '│', term.PanelStyle)
	lines := strings.Split(termtest.Dump(s), "\n")
	want := []string{"..│.", "──│─", "..│."}
	for i, w := range want {
		if lines[i] != w {
			t.Fatalf("line %d: got %q want %q", i, lines[i], w)
		}
	}
}
```

- [ ] **Step 8: Запустить — должен упасть**

Run: `go test ./internal/term/...`
Expected: FAIL — `undefined: term.Canvas`

- [ ] **Step 9: Реализовать `canvas.go`**

`internal/term/canvas.go`:

```go
package term

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"golang.org/x/text/unicode/norm"
)

// Canvas draws NC-style primitives on a tcell screen.
type Canvas struct {
	Screen tcell.Screen
}

func (c Canvas) Put(x, y int, r rune, st tcell.Style) {
	c.Screen.SetContent(x, y, r, nil, st)
}

// Text draws s (normalized to NFC) at (x, y) using at most max columns and
// returns the number of columns drawn. Zero-width runes are skipped.
func (c Canvas) Text(x, y int, s string, max int, st tcell.Style) int {
	used := 0
	for _, r := range norm.NFC.String(s) {
		rw := runewidth.RuneWidth(r)
		if rw == 0 {
			continue
		}
		if used+rw > max {
			break
		}
		c.Screen.SetContent(x+used, y, r, nil, st)
		used += rw
	}
	return used
}

func (c Canvas) Fill(x, y, w, h int, r rune, st tcell.Style) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c.Put(x+i, y+j, r, st)
		}
	}
}

func (c Canvas) HLine(x, y, w int, r rune, st tcell.Style) {
	c.Fill(x, y, w, 1, r, st)
}

func (c Canvas) VLine(x, y, h int, r rune, st tcell.Style) {
	c.Fill(x, y, 1, h, r, st)
}

// Box draws a double-line frame.
func (c Canvas) Box(x, y, w, h int, st tcell.Style) {
	if w < 2 || h < 2 {
		return
	}
	c.HLine(x+1, y, w-2, '═', st)
	c.HLine(x+1, y+h-1, w-2, '═', st)
	c.VLine(x, y+1, h-2, '║', st)
	c.VLine(x+w-1, y+1, h-2, '║', st)
	c.Put(x, y, '╔', st)
	c.Put(x+w-1, y, '╗', st)
	c.Put(x, y+h-1, '╚', st)
	c.Put(x+w-1, y+h-1, '╝', st)
}
```

- [ ] **Step 10: Запустить все тесты**

Run: `go mod tidy && go test ./internal/term/...`
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add go.mod go.sum .gitignore Makefile internal/term
git commit -m "feat: add module skeleton and NC drawing layer"
```

---

### Task 2: Нормализатор клавиш (Esc-префикс)

**Files:**
- Create: `internal/keys/keys.go`
- Test: `internal/keys/keys_test.go`

**Interfaces:**
- Consumes: —
- Produces:
  - `const keys.PrefixTimeout = time.Second`
  - `type keys.Normalizer struct` (нулевое значение готово к работе)
  - `func (n *keys.Normalizer) Feed(ev *tcell.EventKey, now time.Time) *tcell.EventKey`
    — Esc возвращается как есть и взводит префикс; следующая клавиша
    в пределах `PrefixTimeout` получает `ModAlt`; Alt+`1`…`9`,`0` → F1…F10 без модификаторов.

- [ ] **Step 1: Написать падающие тесты**

`internal/keys/keys_test.go`:

```go
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
```

- [ ] **Step 2: Запустить — должен упасть**

Run: `go test ./internal/keys/`
Expected: FAIL — `undefined: Normalizer`

- [ ] **Step 3: Реализовать**

`internal/keys/keys.go`:

```go
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
```

- [ ] **Step 4: Запустить — должен пройти**

Run: `go test ./internal/keys/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/keys
git commit -m "feat: add key normalizer with Esc prefix"
```

---

### Task 3: Файловая система: записи каталога и форматирование

**Files:**
- Create: `internal/fs/entry.go`, `internal/fs/format.go`
- Test: `internal/fs/entry_test.go`, `internal/fs/format_test.go`

**Interfaces:**
- Consumes: —
- Produces:
  - `type fs.Entry struct { Name string; IsDir, IsUp, IsLink bool; Size int64; ModTime time.Time; Mode os.FileMode }`
  - `func fs.ReadDir(path string, showHidden bool) ([]fs.Entry, error)` — `..` первой (кроме `/`), без сортировки остального
  - `func fs.FormatDate(t time.Time) string` → `5-31-94`
  - `func fs.FormatTime(t time.Time) string` → `6:22a`
  - `func fs.FormatSize(n int64, width int) string` → число, при нехватке ширины `K`/`M`/`G`/`T`
  - `func fs.FormatThousands(n int64) string` → `1,234,567`
  - `func fs.DisplayPath(path, home string) string` → `~/projects`

- [ ] **Step 1: Написать падающие тесты форматирования**

`internal/fs/format_test.go`:

```go
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
```

- [ ] **Step 2: Запустить — должен упасть**

Run: `go test ./internal/fs/`
Expected: FAIL — `undefined: FormatDate`

- [ ] **Step 3: Реализовать `format.go`**

`internal/fs/format.go`:

```go
package fs

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatDate formats t as NC does: 5-31-94.
func FormatDate(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d", int(t.Month()), t.Day(), t.Year()%100)
}

// FormatTime formats t as NC does: 6:22a, 4:54p.
func FormatTime(t time.Time) string {
	h, suffix := t.Hour(), "a"
	if h >= 12 {
		suffix = "p"
	}
	h %= 12
	if h == 0 {
		h = 12
	}
	return fmt.Sprintf("%d:%02d%s", h, t.Minute(), suffix)
}

// FormatSize formats n bytes in at most width columns, switching to K, M, G,
// T units when the plain number does not fit.
func FormatSize(n int64, width int) string {
	s := strconv.FormatInt(n, 10)
	for _, unit := range []string{"K", "M", "G", "T"} {
		if len(s) <= width {
			return s
		}
		n /= 1024
		s = strconv.FormatInt(n, 10) + unit
	}
	return s
}

// FormatThousands formats n with comma separators: 1,234,567.
func FormatThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// DisplayPath abbreviates the home directory to ~.
func DisplayPath(path, home string) string {
	if home == "" || home == "/" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}
```

- [ ] **Step 4: Запустить — должен пройти**

Run: `go test ./internal/fs/`
Expected: PASS

- [ ] **Step 5: Написать падающие тесты `ReadDir`**

`internal/fs/entry_test.go`:

```go
package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func names(es []Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Name)
	}
	return out
}

func find(es []Entry, name string) *Entry {
	for i := range es {
		if es[i].Name == name {
			return &es[i]
		}
	}
	return nil
}

func mkTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, ".hidden"), nil, 0o644))
	must(t, os.Symlink(filepath.Join(dir, "sub"), filepath.Join(dir, "linkdir")))
	must(t, os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken")))
	return dir
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadDirUpFirstAndHidesDotfiles(t *testing.T) {
	es, err := ReadDir(mkTree(t), false)
	must(t, err)
	if !es[0].IsUp || es[0].Name != ".." || !es[0].IsDir {
		t.Fatalf("first entry %+v", es[0])
	}
	if find(es, ".hidden") != nil {
		t.Fatalf("hidden file listed: %v", names(es))
	}
	a := find(es, "a.txt")
	if a == nil || a.IsDir || a.Size != 5 {
		t.Fatalf("a.txt: %+v", a)
	}
}

func TestReadDirShowHidden(t *testing.T) {
	es, err := ReadDir(mkTree(t), true)
	must(t, err)
	if find(es, ".hidden") == nil {
		t.Fatalf("hidden file missing: %v", names(es))
	}
}

func TestReadDirSymlinks(t *testing.T) {
	es, err := ReadDir(mkTree(t), false)
	must(t, err)
	if l := find(es, "linkdir"); l == nil || !l.IsDir || !l.IsLink {
		t.Fatalf("linkdir: %+v", l)
	}
	if b := find(es, "broken"); b == nil || b.IsDir || !b.IsLink {
		t.Fatalf("broken: %+v", b)
	}
}

func TestReadDirRootHasNoUp(t *testing.T) {
	es, err := ReadDir("/", false)
	must(t, err)
	if len(es) > 0 && es[0].IsUp {
		t.Fatal("root must not have ..")
	}
}

func TestReadDirMissing(t *testing.T) {
	_, err := ReadDir(filepath.Join(t.TempDir(), "nope"), false)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 6: Запустить — должен упасть**

Run: `go test ./internal/fs/`
Expected: FAIL — `undefined: ReadDir`

- [ ] **Step 7: Реализовать `entry.go`**

`internal/fs/entry.go`:

```go
// Package fs reads directories and formats file data the way NC shows it.
package fs

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry is one line of a file panel.
type Entry struct {
	Name    string
	IsDir   bool
	IsUp    bool // the ".." entry
	IsLink  bool
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
}

// ReadDir lists path. ".." comes first unless path is "/". Dotfiles are
// skipped unless showHidden. Symlinks are described by their targets; broken
// links look like files. Entries after ".." are not sorted.
func ReadDir(path string, showHidden bool) ([]Entry, error) {
	des, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var out []Entry
	if path != "/" {
		up := Entry{Name: "..", IsDir: true, IsUp: true}
		if info, err := os.Stat(filepath.Join(path, "..")); err == nil {
			up.ModTime = info.ModTime()
		}
		out = append(out, up)
	}
	for _, de := range des {
		name := de.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		info, err := os.Lstat(full)
		if err != nil {
			continue // vanished after ReadDir
		}
		e := Entry{Name: name}
		if info.Mode()&os.ModeSymlink != 0 {
			e.IsLink = true
			if target, err := os.Stat(full); err == nil {
				info = target
			}
		}
		e.IsDir = info.IsDir()
		e.Size = info.Size()
		e.ModTime = info.ModTime()
		e.Mode = info.Mode()
		out = append(out, e)
	}
	return out, nil
}
```

- [ ] **Step 8: Запустить — должен пройти**

Run: `go test ./internal/fs/`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/fs
git commit -m "feat: add directory listing and NC-style formatting"
```

---

### Task 4: Модель панели

**Files:**
- Create: `internal/panel/panel.go`, `internal/panel/sort.go`
- Test: `internal/panel/panel_test.go`

**Interfaces:**
- Consumes: `fs.Entry`, `fs.ReadDir` (Task 3)
- Produces:
  - `type panel.Mode int` с константами `panel.Brief`, `panel.Full`
  - `type panel.Panel struct { Path string; Entries []fs.Entry; Cursor, Top int; Mode Mode; ShowHidden bool; Selected map[string]bool }` (+ неэкспортируемое `rows int`)
  - `panel.New() *Panel`
  - методы: `Load(path string) error`, `Reload() error`, `Current() *fs.Entry`, `Focus(name string) bool`, `SetRows(n int)`, `SetMode(m Mode)`, `Move(delta int)`, `Left()`, `Right()`, `PageUp()`, `PageDown()`, `Home()`, `End()`, `Enter() (bool, error)`, `Up() error`, `ToggleSelect()`, `SelectionStats() (count int, bytes int64)`, `SetShowHidden(show bool) error`
  - `func panel.SortEntries(es []fs.Entry)`

- [ ] **Step 1: Написать падающие тесты**

`internal/panel/panel_test.go`:

```go
package panel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/navoznov/terminal-commander/internal/fs"
)

// fake builds a panel over fixed entries; names ending in "/" are dirs.
func fake(rows int, names ...string) *Panel {
	p := New()
	p.Path = "/demo"
	for _, n := range names {
		e := fs.Entry{Name: strings.TrimSuffix(n, "/"), Size: 10}
		e.IsDir = strings.HasSuffix(n, "/")
		e.IsUp = n == "../"
		p.Entries = append(p.Entries, e)
	}
	p.SetRows(rows)
	return p
}

func seq(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, string(rune('a'+i)))
	}
	return out
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSortEntries(t *testing.T) {
	es := []fs.Entry{
		{Name: "b.txt"}, {Name: "Zdir", IsDir: true}, {Name: "A.txt"},
		{Name: "..", IsDir: true, IsUp: true}, {Name: "adir", IsDir: true},
	}
	SortEntries(es)
	var got []string
	for _, e := range es {
		got = append(got, e.Name)
	}
	if strings.Join(got, ",") != "..,adir,Zdir,A.txt,b.txt" {
		t.Fatalf("got %v", got)
	}
}

func TestMoveClamps(t *testing.T) {
	p := fake(3, seq(5)...)
	p.Move(-1)
	if p.Cursor != 0 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Move(100)
	if p.Cursor != 4 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Home()
	if p.Cursor != 0 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.End()
	if p.Cursor != 4 {
		t.Fatalf("cursor %d", p.Cursor)
	}
}

func TestBriefScrollsByColumn(t *testing.T) {
	p := fake(3, seq(12)...) // capacity 9
	p.Move(9)
	if p.Top != 3 {
		t.Fatalf("top %d", p.Top)
	}
	p.Home()
	if p.Top != 0 {
		t.Fatalf("top %d", p.Top)
	}
}

func TestBriefLeftRightJumpColumns(t *testing.T) {
	p := fake(3, seq(8)...)
	p.Right()
	if p.Cursor != 3 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Right()
	p.Right()
	if p.Cursor != 7 {
		t.Fatalf("cursor %d", p.Cursor)
	}
	p.Left()
	if p.Cursor != 4 {
		t.Fatalf("cursor %d", p.Cursor)
	}
}

func TestFullScrollsByLine(t *testing.T) {
	p := fake(3, seq(8)...)
	p.SetMode(Full)
	p.Move(5)
	if p.Top != 3 {
		t.Fatalf("top %d", p.Top)
	}
	p.PageUp()
	if p.Cursor != 2 || p.Top != 2 {
		t.Fatalf("cursor %d top %d", p.Cursor, p.Top)
	}
}

func TestSetRowsKeepsCursorVisible(t *testing.T) {
	p := fake(10, seq(20)...)
	p.Move(19)
	p.SetRows(2)
	if p.Cursor < p.Top || p.Cursor >= p.Top+6 {
		t.Fatalf("cursor %d top %d", p.Cursor, p.Top)
	}
}

func TestToggleSelectSkipsUpAndMovesDown(t *testing.T) {
	p := fake(5, "../", "a", "b")
	p.ToggleSelect()
	if len(p.Selected) != 0 || p.Cursor != 1 {
		t.Fatalf("selected %v cursor %d", p.Selected, p.Cursor)
	}
	p.ToggleSelect()
	p.ToggleSelect()
	if n, bytes := p.SelectionStats(); n != 2 || bytes != 20 {
		t.Fatalf("n %d bytes %d", n, bytes)
	}
	p.Home()
	p.Move(1)
	p.ToggleSelect()
	if n, _ := p.SelectionStats(); n != 1 {
		t.Fatalf("n %d", n)
	}
}

func TestEmptyPanel(t *testing.T) {
	p := fake(3)
	p.Move(1)
	p.End()
	p.Right()
	p.ToggleSelect()
	if p.Current() != nil {
		t.Fatal("current must be nil")
	}
	if ok, err := p.Enter(); ok || err != nil {
		t.Fatalf("enter: %v %v", ok, err)
	}
}

func TestEnterOnFileDoesNothing(t *testing.T) {
	p := fake(3, "a")
	if ok, _ := p.Enter(); ok {
		t.Fatal("entered a file")
	}
}

func TestEnterAndUpFocusesChild(t *testing.T) {
	dir := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755))
	must(t, os.Mkdir(filepath.Join(dir, "a", "a0"), 0o755))
	p := New()
	p.SetRows(10)
	must(t, p.Load(filepath.Join(dir, "a")))
	p.Focus("b")
	ok, err := p.Enter()
	must(t, err)
	if !ok || p.Path != filepath.Join(dir, "a", "b") {
		t.Fatalf("path %s", p.Path)
	}
	must(t, p.Up())
	if p.Path != filepath.Join(dir, "a") || p.Current().Name != "b" {
		t.Fatalf("path %s current %+v", p.Path, p.Current())
	}
}

func TestUpAtRootIsNoop(t *testing.T) {
	p := New()
	must(t, p.Load("/"))
	must(t, p.Up())
	if p.Path != "/" {
		t.Fatalf("path %s", p.Path)
	}
}

func TestReloadKeepsCursorOnSameName(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "m"), nil, 0o644))
	p := New()
	p.SetRows(10)
	must(t, p.Load(dir))
	p.Focus("m")
	p.Selected["m"] = true
	must(t, os.WriteFile(filepath.Join(dir, "a"), nil, 0o644))
	must(t, p.Reload())
	if p.Current().Name != "m" || !p.Selected["m"] {
		t.Fatalf("current %+v selected %v", p.Current(), p.Selected)
	}
}

func TestReloadClimbsWhenDirRemoved(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(dir, "x", "y")
	must(t, os.MkdirAll(gone, 0o755))
	p := New()
	must(t, p.Load(gone))
	must(t, os.RemoveAll(filepath.Join(dir, "x")))
	must(t, p.Reload())
	if p.Path != dir {
		t.Fatalf("path %s want %s", p.Path, dir)
	}
}

func TestSetShowHidden(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, ".dot"), nil, 0o644))
	p := New()
	must(t, p.Load(dir))
	if p.Focus(".dot") {
		t.Fatal(".dot visible")
	}
	must(t, p.SetShowHidden(true))
	if !p.Focus(".dot") {
		t.Fatal(".dot hidden")
	}
}
```

- [ ] **Step 2: Запустить — должен упасть**

Run: `go test ./internal/panel/`
Expected: FAIL — `undefined: New`

- [ ] **Step 3: Реализовать `sort.go`**

`internal/panel/sort.go`:

```go
package panel

import (
	"sort"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/navoznov/terminal-commander/internal/fs"
)

// SortEntries puts ".." first, then directories, then files, each group by
// case-insensitive name.
func SortEntries(es []fs.Entry) {
	key := func(e fs.Entry) string { return strings.ToLower(norm.NFC.String(e.Name)) }
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.IsUp != b.IsUp {
			return a.IsUp
		}
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		if ka, kb := key(a), key(b); ka != kb {
			return ka < kb
		}
		return a.Name < b.Name
	})
}
```

- [ ] **Step 4: Реализовать `panel.go`**

`internal/panel/panel.go`:

```go
// Package panel holds the state of one file panel and draws it.
package panel

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/navoznov/terminal-commander/internal/fs"
)

type Mode int

const (
	Brief Mode = iota
	Full
)

const briefCols = 3

type Panel struct {
	Path       string
	Entries    []fs.Entry
	Cursor     int
	Top        int
	Mode       Mode
	ShowHidden bool
	Selected   map[string]bool

	rows int // visible list rows, set by the layout
}

func New() *Panel {
	return &Panel{Selected: map[string]bool{}, rows: 1}
}

func (p *Panel) read(path string) ([]fs.Entry, error) {
	es, err := fs.ReadDir(path, p.ShowHidden)
	if err != nil {
		return nil, err
	}
	SortEntries(es)
	return es, nil
}

// Load switches the panel to path, resetting cursor and selection. On error
// the panel is left unchanged.
func (p *Panel) Load(path string) error {
	path = filepath.Clean(path)
	es, err := p.read(path)
	if err != nil {
		return err
	}
	p.Path, p.Entries = path, es
	p.Selected = map[string]bool{}
	p.Cursor, p.Top = 0, 0
	return nil
}

// Reload re-reads the directory keeping the cursor and selection by name.
// If the directory is gone, the panel climbs to the nearest existing parent.
func (p *Panel) Reload() error {
	dir := p.Path
	for dir != "/" {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			break
		}
		dir = filepath.Dir(dir)
	}
	if dir != p.Path {
		return p.Load(dir)
	}
	name := ""
	if e := p.Current(); e != nil {
		name = e.Name
	}
	es, err := p.read(dir)
	if err != nil {
		return err
	}
	p.Entries = es
	kept := map[string]bool{}
	for _, e := range es {
		if p.Selected[e.Name] {
			kept[e.Name] = true
		}
	}
	p.Selected = kept
	if !p.Focus(name) {
		p.Move(0)
	}
	return nil
}

// Current returns the entry under the cursor, or nil for an empty panel.
func (p *Panel) Current() *fs.Entry {
	if p.Cursor < 0 || p.Cursor >= len(p.Entries) {
		return nil
	}
	return &p.Entries[p.Cursor]
}

// Focus puts the cursor on name and reports whether it was found.
func (p *Panel) Focus(name string) bool {
	for i, e := range p.Entries {
		if e.Name == name {
			p.Cursor = i
			p.ensureVisible()
			return true
		}
	}
	return false
}

// SetRows sets the number of visible list rows (from the layout).
func (p *Panel) SetRows(n int) {
	n = max(n, 1)
	if n == p.rows {
		return
	}
	p.rows = n
	p.Top = 0
	p.ensureVisible()
}

func (p *Panel) SetMode(m Mode) {
	p.Mode = m
	p.Top = 0
	p.ensureVisible()
}

func (p *Panel) capacity() int {
	if p.Mode == Brief {
		return p.rows * briefCols
	}
	return p.rows
}

func (p *Panel) ensureVisible() {
	capacity := p.capacity()
	if p.Mode == Brief {
		// Scroll a whole column at a time, like NC.
		if p.Cursor < p.Top {
			p.Top = p.Cursor / p.rows * p.rows
		}
		if p.Cursor >= p.Top+capacity {
			p.Top = (p.Cursor/p.rows - (briefCols - 1)) * p.rows
		}
	} else {
		if p.Cursor < p.Top {
			p.Top = p.Cursor
		}
		if p.Cursor >= p.Top+capacity {
			p.Top = p.Cursor - capacity + 1
		}
	}
	p.Top = max(p.Top, 0)
}

// Move shifts the cursor by delta, clamped to the list.
func (p *Panel) Move(delta int) {
	p.Cursor = min(max(p.Cursor+delta, 0), max(len(p.Entries)-1, 0))
	p.ensureVisible()
}

func (p *Panel) Left() {
	if p.Mode == Brief {
		p.Move(-p.rows)
	} else {
		p.PageUp()
	}
}

func (p *Panel) Right() {
	if p.Mode == Brief {
		p.Move(p.rows)
	} else {
		p.PageDown()
	}
}

func (p *Panel) PageUp()   { p.Move(-p.capacity()) }
func (p *Panel) PageDown() { p.Move(p.capacity()) }
func (p *Panel) Home()     { p.Move(-len(p.Entries)) }
func (p *Panel) End()      { p.Move(len(p.Entries)) }

// Enter opens the directory under the cursor. It reports false when the
// cursor is not on a directory.
func (p *Panel) Enter() (bool, error) {
	e := p.Current()
	if e == nil || !e.IsDir {
		return false, nil
	}
	if e.IsUp {
		return true, p.Up()
	}
	return true, p.Load(filepath.Join(p.Path, e.Name))
}

// Up goes to the parent directory and puts the cursor on the one we left.
func (p *Panel) Up() error {
	if p.Path == "/" {
		return nil
	}
	child := filepath.Base(p.Path)
	if err := p.Load(filepath.Dir(p.Path)); err != nil {
		return err
	}
	p.Focus(child)
	return nil
}

// ToggleSelect flips selection of the entry under the cursor (never "..")
// and moves down.
func (p *Panel) ToggleSelect() {
	e := p.Current()
	if e == nil {
		return
	}
	if !e.IsUp {
		if p.Selected[e.Name] {
			delete(p.Selected, e.Name)
		} else {
			p.Selected[e.Name] = true
		}
	}
	p.Move(1)
}

// SelectionStats returns the number of selected entries and the total size
// of the selected files.
func (p *Panel) SelectionStats() (count int, bytes int64) {
	for _, e := range p.Entries {
		if p.Selected[e.Name] {
			count++
			if !e.IsDir {
				bytes += e.Size
			}
		}
	}
	return count, bytes
}

func (p *Panel) SetShowHidden(show bool) error {
	p.ShowHidden = show
	return p.Reload()
}
```

- [ ] **Step 5: Запустить — должен пройти**

Run: `go test ./internal/panel/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/panel
git commit -m "feat: add file panel model with navigation and selection"
```

---

### Task 5: Отрисовка панели

**Files:**
- Create: `internal/panel/render.go`
- Test: `internal/panel/render_test.go`, `internal/panel/testdata/*.golden` (создаются флагом `-update`)

**Interfaces:**
- Consumes: `term.Canvas`, стили и `term.Fit/Width/Tail` (Task 1); `fs.Format*`, `fs.DisplayPath` (Task 3); `Panel` (Task 4)
- Produces:
  - `func (p *Panel) Draw(c term.Canvas, x, y, w, h int, active bool, home string)` — рисует панель в прямоугольнике; вызывающий заранее вызывает `p.SetRows(h - 5)`.

Раскладка внутри прямоугольника (x, y, w, h):
строка `y` — верхняя рамка с путём; `y+1` — заголовки колонок;
`y+2 … y+h-4` — список (`h-5` строк); `y+h-3` — `╟─╢`;
`y+h-2` — мини-статус; `y+h-1` — нижняя рамка.
Колонки: краткий режим — 3 по `(w-4)/3` (последняя берёт остаток),
полный — `Name` шириной `w-2-26`, `Size` 9, `Date` 8, `Time` 6.

- [ ] **Step 1: Написать тесты (golden + форматирование имён)**

`internal/panel/render_test.go`:

```go
package panel

import (
	"strings"
	"testing"
	"time"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

var day = time.Date(1994, 5, 31, 6, 22, 0, 0, time.Local)

func demo() *Panel {
	p := New()
	p.Path = "/Users/nc/GAMES"
	for _, e := range []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true},
		{Name: "CIV", IsDir: true},
		{Name: "LINES", IsDir: true},
		{Name: "autoexec.bat", Size: 13},
		{Name: "command.com", Size: 54645},
		{Name: "config.sys", Size: 13},
		{Name: "read.me", Size: 577},
		{Name: "a-very-long-file-name.txt", Size: 1234567},
		{Name: "Makefile", Size: 99},
	} {
		e.ModTime = day
		p.Entries = append(p.Entries, e)
	}
	p.Selected["config.sys"] = true
	return p
}

func draw(t *testing.T, p *Panel, active bool) string {
	s := termtest.NewScreen(t, 40, 12)
	p.SetRows(12 - 5)
	p.Draw(term.Canvas{Screen: s}, 0, 0, 40, 12, active, "/Users/nc")
	return termtest.Dump(s)
}

func TestDrawBrief(t *testing.T) {
	p := demo()
	p.Focus("command.com")
	termtest.Golden(t, "brief", draw(t, p, true))
}

func TestDrawFull(t *testing.T) {
	p := demo()
	p.SetMode(Full)
	p.Focus("CIV")
	termtest.Golden(t, "full", draw(t, p, true))
}

func TestDrawInactiveHasNoCursor(t *testing.T) {
	out := draw(t, demo(), false)
	styles := strings.SplitN(out, "\n\n", 2)[1]
	if strings.ContainsAny(styles, "cY") {
		t.Fatalf("inactive panel shows cursor or active title:\n%s", out)
	}
}

func TestDrawSelectionStatus(t *testing.T) {
	out := draw(t, demo(), true)
	if !strings.Contains(out, "13 bytes in 1 selected files") {
		t.Fatalf("no selection status:\n%s", out)
	}
}

func TestDrawEmptyPanel(t *testing.T) {
	p := New()
	p.Path = "/empty"
	draw(t, p, true) // must not panic
}

func TestBriefNameLayout(t *testing.T) {
	cases := []struct {
		e    fs.Entry
		want string
	}{
		{fs.Entry{Name: "autoexec.bat"}, "autoexec bat"},
		{fs.Entry{Name: "read.me"}, "read     me "},
		{fs.Entry{Name: "Makefile"}, "Makefile    "},
		{fs.Entry{Name: ".zshrc"}, ".zshrc      "},
		{fs.Entry{Name: "archive.tar.gz"}, "archive} gz "},
		{fs.Entry{Name: "index.html"}, "index.html  "},
		{fs.Entry{Name: "my.dir", IsDir: true}, "my.dir      "},
		{fs.Entry{Name: "a-very-long-file-name.txt"}, "a-very-} txt"},
	}
	for _, c := range cases {
		if got := fitName(c.e, 12); got != c.want {
			t.Errorf("%s: got %q want %q", c.e.Name, got, c.want)
		}
	}
}

func TestBriefNFDName(t *testing.T) {
	e := fs.Entry{Name: "йод.txt"} // «йод.txt» в NFD
	if got := fitName(e, 12); got != "йод      txt" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: Запустить — должен упасть**

Run: `go test ./internal/panel/`
Expected: FAIL — `p.Draw undefined`, `undefined: fitName`

- [ ] **Step 3: Реализовать `render.go`**

`internal/panel/render.go`:

```go
package panel

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
)

const (
	sizeW = 9
	dateW = 8
	timeW = 6
	// Columns taken by "│size│date│time" in full mode.
	fullFixed = 1 + sizeW + 1 + dateW + 1 + timeW
)

type column struct {
	off, width int
	header     string
}

func (p *Panel) columns(inner int) []column {
	if p.Mode == Full {
		nameW := inner - fullFixed
		return []column{
			{0, nameW, "Name"},
			{nameW + 1, sizeW, "Size"},
			{nameW + 2 + sizeW, dateW, "Date"},
			{nameW + 3 + sizeW + dateW, timeW, "Time"},
		}
	}
	cw := (inner - 2) / briefCols
	return []column{
		{0, cw, "Name"},
		{cw + 1, cw, "Name"},
		{2*cw + 2, inner - 2*cw - 2, "Name"},
	}
}

// Draw renders the panel into the rectangle (x, y, w, h). The caller sets
// the list height with SetRows(h - 5) beforehand.
func (p *Panel) Draw(c term.Canvas, x, y, w, h int, active bool, home string) {
	st := term.PanelStyle
	inner := w - 2
	sepY := y + h - 3
	c.Fill(x, y, w, h, ' ', st)
	c.Box(x, y, w, h, st)
	c.Put(x, sepY, '╟', st)
	c.HLine(x+1, sepY, inner, '─', st)
	c.Put(x+w-1, sepY, '╢', st)

	cols := p.columns(inner)
	for i, col := range cols {
		cx := x + 1 + col.off
		if i > 0 {
			c.Put(cx-1, y, '╤', st)
			c.VLine(cx-1, y+1, h-4, '│', st)
			c.Put(cx-1, sepY, '┴', st)
		}
		hw := term.Width(col.header)
		c.Text(cx+max(0, (col.width-hw)/2), y+1, col.header, col.width, term.HeaderStyle)
	}

	p.drawEntries(c, x+1, y+2, cols, active)
	p.drawStatus(c, x+1, y+h-2, inner)
	p.drawTitle(c, x, y, w, active, home)
}

func (p *Panel) drawEntries(c term.Canvas, x0, y0 int, cols []column, active bool) {
	if p.Mode == Full {
		for r := 0; r < p.rows; r++ {
			i := p.Top + r
			if i >= len(p.Entries) {
				return
			}
			e := p.Entries[i]
			st := p.entryStyle(i, active)
			fields := []string{
				fitName(e, cols[0].width),
				fitRight(sizeText(e), sizeW),
				fitRight(fs.FormatDate(e.ModTime), dateW),
				fitRight(fs.FormatTime(e.ModTime), timeW),
			}
			for j, col := range cols {
				c.Text(x0+col.off, y0+r, fields[j], col.width, st)
				if j > 0 && active && i == p.Cursor {
					c.Put(x0+col.off-1, y0+r, '│', st)
				}
			}
		}
		return
	}
	for k := 0; k < p.rows*len(cols); k++ {
		i := p.Top + k
		if i >= len(p.Entries) {
			return
		}
		col := cols[k/p.rows]
		c.Text(x0+col.off, y0+k%p.rows, fitName(p.Entries[i], col.width), col.width, p.entryStyle(i, active))
	}
}

func (p *Panel) entryStyle(i int, active bool) tcell.Style {
	sel := p.Selected[p.Entries[i].Name]
	cur := active && i == p.Cursor
	switch {
	case cur && sel:
		return term.SelectedCursorStyle
	case cur:
		return term.CursorStyle
	case sel:
		return term.SelectedStyle
	}
	return term.PanelStyle
}

func (p *Panel) drawStatus(c term.Canvas, x, y, w int) {
	if n, bytes := p.SelectionStats(); n > 0 {
		s := fmt.Sprintf("%s bytes in %d selected files", fs.FormatThousands(bytes), n)
		c.Text(x+max(0, (w-term.Width(s))/2), y, s, w, term.SelectedStyle)
		return
	}
	e := p.Current()
	if e == nil {
		return
	}
	right := fitRight(sizeText(*e), sizeW) + " " +
		fitRight(fs.FormatDate(e.ModTime), dateW) + " " +
		fitRight(fs.FormatTime(e.ModTime), timeW)
	nameW := w - term.Width(right) - 1
	c.Text(x, y, term.Fit(e.Name, nameW)+" "+right, w, term.PanelStyle)
}

func (p *Panel) drawTitle(c term.Canvas, x, y, w int, active bool, home string) {
	t := fs.DisplayPath(p.Path, home)
	if limit := w - 4; term.Width(t) > limit {
		t = "…" + term.Tail(t, limit-1)
	}
	t = " " + t + " "
	st := term.PanelStyle
	if active {
		st = term.ActiveTitleStyle
	}
	tw := term.Width(t)
	c.Text(x+(w-tw)/2, y, t, tw, st)
}

// fitName formats a name for a column of width w. Files with a 1–3 column
// extension get the DOS layout: name, then the extension in the last three
// columns ("autoexec bat"). Everything else is shown whole, cut with '}'.
func fitName(e fs.Entry, w int) string {
	if !e.IsDir && w >= 5 {
		if base, ext, ok := splitExt(e.Name); ok {
			return term.Fit(base, w-4) + " " + term.Fit(ext, 3)
		}
	}
	return term.Fit(e.Name, w)
}

func splitExt(name string) (base, ext string, ok bool) {
	i := strings.LastIndexByte(name, '.')
	if i <= 0 || i == len(name)-1 {
		return "", "", false
	}
	if term.Width(name[i+1:]) > 3 {
		return "", "", false
	}
	return name[:i], name[i+1:], true
}

func sizeText(e fs.Entry) string {
	switch {
	case e.IsUp:
		return "►UP--DIR◄"
	case e.IsDir:
		return "►SUB-DIR◄"
	}
	return fs.FormatSize(e.Size, sizeW)
}

func fitRight(s string, w int) string {
	if sw := term.Width(s); sw < w {
		return strings.Repeat(" ", w-sw) + s
	}
	return term.Fit(s, w)
}
```

- [ ] **Step 4: Запустить нерисующие тесты — должны пройти**

Run: `go test ./internal/panel/ -run 'Name|Inactive|Selection|Empty'`
Expected: PASS

- [ ] **Step 5: Создать golden-файлы и проверить их глазами**

Run: `go test ./internal/panel/ -run 'TestDrawBrief|TestDrawFull' -update && cat internal/panel/testdata/brief.golden internal/panel/testdata/full.golden`

Проверить вручную по спеке и скриншотам:
- верх: `╔════…╤…╗` с ` ~/GAMES ` по центру; под буквами пути в сетке стилей — `c` (активная панель);
- строка 1: три жёлтых (`y`) `Name` по центру колонок (полный режим — `Name Size Date Time`);
- краткий: колонки по 12 символов, `autoexec bat`, `read     me `, `a-very-} txt`; курсор (`c`) на `command com` ровно на ширину колонки; `config   sys` — жёлтый (`y`);
- полный: `CIV` с `►SUB-DIR◄  5-31-94  6:22a`, разделители `│`, на строке курсора разделители тоже в стиле `c`;
- строка 9: `╟───…┴…╢`; строка 10: `13 bytes in 1 selected files` по центру; строка 11: `╚═…╝`.

Если что-то не совпадает — исправить `render.go` и перегенерировать.

- [ ] **Step 6: Запустить все тесты пакета**

Run: `go test ./internal/panel/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/panel
git commit -m "feat: draw file panels in NC brief and full modes"
```

---

### Task 6: Приложение, `main`, режим `--keytest`

**Files:**
- Create: `internal/app/app.go`, `internal/app/draw.go`, `internal/keytest/keytest.go`, `cmd/tc/main.go`
- Test: `internal/app/app_test.go`, `internal/app/testdata/screen.golden`, `internal/keytest/keytest_test.go`

**Interfaces:**
- Consumes: всё из Tasks 1–5: `term.Canvas`, стили, `termtest.*`, `keys.Normalizer`, `fs.DisplayPath`, `panel.Panel` и его методы
- Produces:
  - `func app.New(s tcell.Screen, leftDir, rightDir string) *app.App`
  - `func (a *App) Run()`, `func (a *App) HandleEvent(ev tcell.Event)`, `func (a *App) Draw()`
  - `func keytest.Run(s tcell.Screen)`, `func keytest.Describe(raw, norm *tcell.EventKey) string`

Клавиши этого этапа: Tab, стрелки, Home/End, PgUp/PgDn, Enter (только
папки), Backspace и Control-PgUp — вверх, Insert и Space — выделение,
Alt-. / Esc `.` — скрытые файлы, Control-R — перечитать, Control-U —
поменять панели местами, Control-1/2 — режим, F10 (и Esc `0`) — выход
(подтверждение появится вместе с диалогами на этапе 3). Ошибки
выводятся белым на красном в строке команд до следующей клавиши
(до появления диалогов).

- [ ] **Step 1: Написать тест `keytest.Describe`**

`internal/keytest/keytest_test.go`:

```go
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
```

- [ ] **Step 2: Запустить — должен упасть**

Run: `go test ./internal/keytest/`
Expected: FAIL — `undefined: Describe`

- [ ] **Step 3: Реализовать `keytest.go`**

`internal/keytest/keytest.go`:

```go
// Package keytest shows how key presses reach the program (tc --keytest).
package keytest

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/keys"
	"github.com/navoznov/terminal-commander/internal/term"
)

// Describe formats a raw key event and the result of normalizing it.
func Describe(raw, norm *tcell.EventKey) string {
	return fmt.Sprintf("%-22s key=%-4d rune=%-6q mod=%-2d -> %s",
		raw.Name(), raw.Key(), raw.Rune(), raw.Modifiers(), norm.Name())
}

// Run prints every key event until Control-C.
func Run(s tcell.Screen) {
	var n keys.Normalizer
	lines := []string{"Press keys to see how they are recognised. Control-C quits."}
	for {
		s.Clear()
		c := term.Canvas{Screen: s}
		w, h := s.Size()
		for i, l := range lines[max(0, len(lines)-h):] {
			c.Text(0, i, l, w, tcell.StyleDefault)
		}
		s.Show()
		switch ev := s.PollEvent().(type) {
		case nil:
			return
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlC {
				return
			}
			lines = append(lines, Describe(ev, n.Feed(ev, ev.When())))
		}
	}
}
```

- [ ] **Step 4: Запустить — должен пройти**

Run: `go test ./internal/keytest/`
Expected: PASS

- [ ] **Step 5: Написать падающие тесты приложения**

`internal/app/app_test.go`:

```go
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func press(a *App, k tcell.Key, r rune, m tcell.ModMask) {
	a.HandleEvent(tcell.NewEventKey(k, r, m))
	a.Draw()
}

func newApp(t *testing.T) (*App, string) {
	dir := t.TempDir()
	must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "sub", "f.txt"), nil, 0o644))
	must(t, os.WriteFile(filepath.Join(dir, ".dot"), nil, 0o644))
	s := termtest.NewScreen(t, 80, 25)
	a := New(s, dir, dir)
	a.Draw()
	return a, dir
}

func TestTabSwitchesPanel(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyTab, 0, 0)
	if a.active != 1 {
		t.Fatalf("active %d", a.active)
	}
}

func TestEnterAndBackspace(t *testing.T) {
	a, dir := newApp(t)
	p := a.panels[0]
	p.Focus("sub")
	press(a, tcell.KeyEnter, 0, 0)
	if p.Path != filepath.Join(dir, "sub") {
		t.Fatalf("path %s", p.Path)
	}
	press(a, tcell.KeyBackspace, 0, 0)
	if p.Path != dir || p.Current().Name != "sub" {
		t.Fatalf("path %s current %+v", p.Path, p.Current())
	}
}

func TestCtrlPgUpGoesUp(t *testing.T) {
	a, dir := newApp(t)
	must(t, a.panels[0].Load(filepath.Join(dir, "sub")))
	press(a, tcell.KeyPgUp, 0, tcell.ModCtrl)
	if a.panels[0].Path != dir {
		t.Fatalf("path %s", a.panels[0].Path)
	}
}

func TestAltDotTogglesHiddenInBothPanels(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyRune, '.', tcell.ModAlt)
	for i, p := range a.panels {
		if !p.Focus(".dot") {
			t.Fatalf("panel %d: .dot not shown", i)
		}
	}
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyRune, '.', 0)
	if a.panels[0].Focus(".dot") {
		t.Fatal("Esc . did not hide .dot")
	}
}

func TestSpaceSelects(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyRune, ' ', 0)
	if !a.panels[0].Selected["sub"] {
		t.Fatal("sub not selected")
	}
}

func TestCtrlUSwapsPanels(t *testing.T) {
	a, dir := newApp(t)
	must(t, a.panels[0].Load(filepath.Join(dir, "sub")))
	press(a, tcell.KeyCtrlU, 0, 0)
	if a.panels[1].Path != filepath.Join(dir, "sub") || a.active != 1 {
		t.Fatalf("right %s active %d", a.panels[1].Path, a.active)
	}
}

func TestEscZeroQuits(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyRune, '0', 0)
	if !a.quit {
		t.Fatal("Esc 0 did not quit")
	}
}

func TestEnterUnreadableDirShowsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	a, dir := newApp(t)
	locked := filepath.Join(dir, "locked")
	must(t, os.Mkdir(locked, 0o000))
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	p := a.panels[0]
	must(t, p.Reload())
	p.Focus("locked")
	press(a, tcell.KeyEnter, 0, 0)
	if p.Path != dir || a.errMsg == "" {
		t.Fatalf("path %s err %q", p.Path, a.errMsg)
	}
	if !strings.Contains(a.errMsg, "permission denied") {
		t.Fatalf("err %q", a.errMsg)
	}
	if !strings.Contains(termtest.Dump(a.screen.(tcell.SimulationScreen)), "open /") {
		t.Fatal("error not shown in the command line")
	}
	press(a, tcell.KeyDown, 0, 0)
	if a.errMsg != "" {
		t.Fatal("error not cleared by next key")
	}
}

func TestSmallWindow(t *testing.T) {
	a, _ := newApp(t)
	s := a.screen.(tcell.SimulationScreen)
	s.SetSize(40, 10)
	a.HandleEvent(tcell.NewEventResize(40, 10))
	a.Draw()
	if !strings.Contains(termtest.Dump(s), "Window too small") {
		t.Fatal("no small-window message")
	}
	press(a, tcell.KeyDown, 0, 0) // must not panic
}

func TestScreenGolden(t *testing.T) {
	a, _ := newApp(t)
	a.home = "/Users/nc"
	day := time.Date(1994, 5, 31, 6, 22, 0, 0, time.Local)
	left, right := a.panels[0], a.panels[1]
	left.Path, right.Path = "/Users/nc", "/Users/nc/GAMES"
	left.Entries = []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true, ModTime: day},
		{Name: "DOS", IsDir: true, ModTime: day},
		{Name: "GAMES", IsDir: true, ModTime: day},
		{Name: "autoexec.bat", Size: 13, ModTime: day},
		{Name: "config.sys", Size: 13, ModTime: day},
	}
	right.Entries = []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true, ModTime: day},
		{Name: "CIV", IsDir: true, ModTime: day},
		{Name: "tetris.exe", Size: 42001, ModTime: day},
	}
	left.Cursor, right.Cursor = 4, 1
	right.SetMode(panel.Full)
	a.Draw()
	termtest.Golden(t, "screen", termtest.Dump(a.screen.(tcell.SimulationScreen)))
}
```

- [ ] **Step 6: Запустить — должен упасть**

Run: `go test ./internal/app/`
Expected: FAIL — `undefined: New`

- [ ] **Step 7: Реализовать `app.go`**

`internal/app/app.go`:

```go
// Package app wires the panels, key handling and screen layout together.
package app

import (
	"os"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/keys"
	"github.com/navoznov/terminal-commander/internal/panel"
)

type App struct {
	screen     tcell.Screen
	panels     [2]*panel.Panel
	active     int
	keys       keys.Normalizer
	showHidden bool
	home       string
	errMsg     string // shown in the command line until the next key
	quit       bool
}

// New opens the left and right panels in the given directories, falling back
// to the home directory and then "/".
func New(s tcell.Screen, leftDir, rightDir string) *App {
	home, _ := os.UserHomeDir()
	a := &App{screen: s, home: home}
	for i, dir := range []string{leftDir, rightDir} {
		p := panel.New()
		if err := p.Load(dir); err != nil {
			a.report(err)
			if p.Load(home) != nil {
				a.report(p.Load("/"))
			}
		}
		a.panels[i] = p
	}
	return a
}

func (a *App) Run() {
	for !a.quit {
		a.Draw()
		a.screen.Show()
		a.HandleEvent(a.screen.PollEvent())
	}
}

func (a *App) HandleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case nil:
		a.quit = true // screen finalized
	case *tcell.EventResize:
		a.screen.Sync()
	case *tcell.EventKey:
		a.errMsg = ""
		a.handleKey(a.keys.Feed(ev, ev.When()))
	}
}

func (a *App) report(err error) {
	if err != nil {
		a.errMsg = err.Error()
	}
}

func (a *App) handleKey(ev *tcell.EventKey) {
	p := a.panels[a.active]
	switch ev.Key() {
	case tcell.KeyF10:
		a.quit = true
	case tcell.KeyTab:
		a.active = 1 - a.active
	case tcell.KeyUp:
		p.Move(-1)
	case tcell.KeyDown:
		p.Move(1)
	case tcell.KeyLeft:
		p.Left()
	case tcell.KeyRight:
		p.Right()
	case tcell.KeyHome:
		p.Home()
	case tcell.KeyEnd:
		p.End()
	case tcell.KeyPgUp:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			a.report(p.Up())
		} else {
			p.PageUp()
		}
	case tcell.KeyPgDn:
		p.PageDown()
	case tcell.KeyEnter:
		_, err := p.Enter()
		a.report(err)
	case tcell.KeyBackspace:
		a.report(p.Up())
	case tcell.KeyInsert:
		p.ToggleSelect()
	case tcell.KeyCtrlR:
		a.report(p.Reload())
	case tcell.KeyCtrlU:
		a.panels[0], a.panels[1] = a.panels[1], a.panels[0]
		a.active = 1 - a.active
	case tcell.KeyRune:
		a.handleRune(ev.Rune(), ev.Modifiers())
	}
}

func (a *App) handleRune(r rune, mod tcell.ModMask) {
	p := a.panels[a.active]
	switch {
	case r == '.' && mod&tcell.ModAlt != 0:
		a.toggleHidden()
	case r == '1' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Brief)
	case r == '2' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Full)
	case r == ' ' && mod == 0:
		p.ToggleSelect()
	}
}

func (a *App) toggleHidden() {
	a.showHidden = !a.showHidden
	for _, p := range a.panels {
		a.report(p.SetShowHidden(a.showHidden))
	}
}
```

- [ ] **Step 8: Реализовать `draw.go`**

`internal/app/draw.go`:

```go
package app

import (
	"strconv"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
)

const (
	minWidth  = 80
	minHeight = 24
)

var keyLabels = [10]string{"Help", "Menu", "View", "Edit", "Copy", "RenMov", "Mkdir", "Delete", "PullDn", "Quit"}

// Draw lays out the whole screen: two panels, the command line and the key bar.
func (a *App) Draw() {
	c := term.Canvas{Screen: a.screen}
	w, h := a.screen.Size()
	a.screen.Clear()
	if w < minWidth || h < minHeight {
		a.screen.HideCursor()
		msg := "Window too small"
		c.Text(max(0, (w-len(msg))/2), h/2, msg, w, term.CmdLineStyle)
		return
	}
	ph := h - 2
	lw := w / 2
	for i, p := range a.panels {
		x, pw := 0, lw
		if i == 1 {
			x, pw = lw, w-lw
		}
		p.SetRows(ph - 5)
		p.Draw(c, x, 0, pw, ph, i == a.active, a.home)
	}
	a.drawCmdLine(c, h-2, w)
	a.drawKeyBar(c, h-1, w)
}

func (a *App) drawCmdLine(c term.Canvas, y, w int) {
	c.HLine(0, y, w, ' ', term.CmdLineStyle)
	if a.errMsg != "" {
		c.Text(0, y, a.errMsg, w, term.ErrorStyle)
		a.screen.HideCursor()
		return
	}
	prompt := fs.DisplayPath(a.panels[a.active].Path, a.home) + ">"
	n := c.Text(0, y, prompt, w, term.CmdLineStyle)
	a.screen.ShowCursor(n, y)
}

func (a *App) drawKeyBar(c term.Canvas, y, w int) {
	c.HLine(0, y, w, ' ', term.KeyNumStyle)
	for i, label := range keyLabels {
		x0, x1 := i*w/10, (i+1)*w/10
		n := c.Text(x0, y, strconv.Itoa(i+1), x1-x0, term.KeyNumStyle)
		c.Text(x0+n, y, term.Fit(label, x1-x0-n), x1-x0-n, term.KeyLabelStyle)
	}
}
```

- [ ] **Step 9: Запустить нерисующие тесты — должны пройти**

Run: `go test ./internal/app/ -run 'Tab|Enter|Ctrl|Alt|Space|Esc|Small'`
Expected: PASS

- [ ] **Step 10: Создать golden экрана и проверить глазами**

Run: `go test ./internal/app/ -run TestScreenGolden -update && cat internal/app/testdata/screen.golden`

Проверить: левая панель 40 колонок (краткий режим), правая 40 (полный);
путь левой ` ~ ` на `c` (активна), правой ` ~/GAMES ` на `p`;
курсор только в левой (на `config   sys`); строка 23 — `~>` на `k`;
строка 24 — `1Help    2Menu    …10Quit` с цифрами `k` и подписями `c`,
каждая ячейка по 8 колонок.

- [ ] **Step 11: Реализовать `cmd/tc/main.go`**

`cmd/tc/main.go`:

```go
// Command tc is Terminal Commander, a Norton Commander look-alike for macOS.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/app"
	"github.com/navoznov/terminal-commander/internal/keytest"
)

func main() {
	keyTest := flag.Bool("keytest", false, "show how key presses are recognised")
	flag.Parse()

	s, err := tcell.NewScreen()
	if err == nil {
		err = s.Init()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tc:", err)
		os.Exit(1)
	}
	defer func() {
		if r := recover(); r != nil {
			s.Fini()
			path := writeCrashLog(r)
			fmt.Fprintf(os.Stderr, "tc crashed: %v\ndetails: %s\n", r, path)
			os.Exit(2)
		}
	}()

	if *keyTest {
		keytest.Run(s)
		s.Fini()
		return
	}
	start, _ := os.Getwd()
	if flag.NArg() > 0 {
		if abs, err := filepath.Abs(flag.Arg(0)); err == nil {
			start = abs
		}
	}
	app.New(s, start, start).Run()
	s.Fini()
}

// writeCrashLog saves the panic and stack to
// ~/.config/terminal-commander/crash.log and returns the file path.
func writeCrashLog(r any) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "(no home directory)"
	}
	dir := filepath.Join(home, ".config", "terminal-commander")
	path := filepath.Join(dir, "crash.log")
	if os.MkdirAll(dir, 0o755) != nil {
		return "(cannot create " + dir + ")"
	}
	msg := fmt.Sprintf("panic: %v\n\n%s", r, debug.Stack())
	if os.WriteFile(path, []byte(msg), 0o644) != nil {
		return "(cannot write " + path + ")"
	}
	return path
}
```

- [ ] **Step 12: Собрать и прогнать все тесты**

Run: `go vet ./... && go test ./... && make build`
Expected: всё PASS, в корне появился бинарник `tc`.

- [ ] **Step 13: Ручная проверка в терминалах (выполняет пользователь)**

Исполнитель просит пользователя пройти список ниже и присылает ему его;
по ответу пользователя заполняет `docs/terminal-compat.md`.
В каждом из терминалов — iTerm2, VS Code, JetBrains, Orca, Terminal.app:

1. `./tc` — две синие панели с двойными рамками, строка клавиш; Tab,
   стрелки, Enter в папку, Backspace вверх (курсор на папке, из которой
   вышли), Space выделяет (жёлтым, внизу «N bytes in M selected files»),
   Option-. и `Esc` `.` показывают/скрывают dotfiles, F10 и `Esc` `0` выходят.
2. Уменьшить окно ниже 80×24 — `Window too small`; вернуть — панели на месте.
3. `./tc --keytest` — нажать F1…F10, Option-F1, Option-F2, Shift-F8,
   Option-., Control-1, Control-2, Control-PgUp (Control-Fn-↑), `Esc` затем `5`.
   Записать, что распознаётся в каждом терминале (для этапов 3–6).

Результаты проверки записать в `docs/terminal-compat.md` (таблица:
клавиша × терминал → что пришло).

- [ ] **Step 14: Commit**

```bash
git add internal/app internal/keytest cmd/tc docs/terminal-compat.md
git commit -m "feat: run two-panel NC screen with key handling and keytest mode"
```

---

## После плана

- Push ветки `feature/v1` и черновой PR в `main` (`gh pr create --draft`),
  чтобы следующие этапы копились в том же PR или шли отдельными PR — по выбору пользователя.
- Следующий план: этап 3 (диалоги, меню F9, подтверждение выхода, выбор диска, справка F1).
