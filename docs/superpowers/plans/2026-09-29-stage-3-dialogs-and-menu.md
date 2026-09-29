# Terminal Commander — этап 3: диалоги, меню, выбор диска, справка. План реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Модальные окна в стиле NC: диалоги с кнопками и полем ввода,
меню F9/F2, выбор «диска» Alt-F1/Alt-F2, справка F1, подтверждение
выхода F10, выделение по маске `+`/`-`/`*`, режимы сортировки, ошибки
в красном диалоге.

**Architecture:** Новый пакет `internal/ui` — виджеты (`Dialog`, `Input`,
`MenuBar`, `TextView`) и стек модальных окон `Stack`. Виджеты ничего не
знают о панелях: действия передаются замыканиями (`Done`, `Item.Action`).
`internal/app` держит `ui.Stack`; пока стек не пуст, все клавиши идут
верхнему окну. Модели (`panel` — сортировка и маски, `fs` — список дисков)
расширяются без зависимости от UI.

**Tech Stack:** Go 1.27, `github.com/gdamore/tcell/v2` v2.13.x,
`github.com/mattn/go-runewidth`, `golang.org/x/text/unicode/norm`.

**Spec:** `docs/superpowers/specs/2026-09-28-terminal-commander-design.md`
(этап 3: «`ui`: диалоги, меню F9, выбор диска, справка F1»).

## Global Constraints

- Модуль `github.com/navoznov/terminal-commander`; бинарник `tc`.
- `fs` не импортирует `tcell`; `panel` не знает о другой панели, меню и диалогах; `ui` не импортирует `panel`, `fs`, `app`.
- Цветовые роли из спеки: меню и выпадающие списки black / cyan, выбранный пункт white / black; диалог black / lightgray, двойная рамка white, заголовок в разрыве рамки; кнопка black / white, в фокусе black / yellow; диалог ошибки white / red; тень black, смещение +2 колонки вправо, +1 строка вниз.
- Меню (тексты пунктов и горячих клавиш — как в спеке):
  - Left / Right: Brief, Full, ─, Name, Extension, Time, Size, Unsorted, ─, Re-read, Drive…
  - Files: Help F1, View F3, Edit F4, Copy F5, Rename/Move F6, Make directory F7, Delete F8, Delete permanently Shift-F8, ─, Select group +, Unselect group -, Invert selection *, ─, Quit F10
  - Commands: Swap panels Control-U, Panels on/off Control-O, Command history
  - Options: Show hidden files Alt-., Save setup
- Полоса меню: `Left  Files  Commands  Options  Right` в строке 0 поверх рамок, только пока меню открыто.
- Выход F10: диалог «Do you want to quit Terminal Commander?» Yes / No.
- Диалог выбора диска — по центру над соответствующей панелью; кнопки `/`, `~`, тома из `/Volumes` (кроме ссылки на загрузочный том); подпись тома обрезается до 12 символов; ←/→ — выбор, Enter — перейти, Esc — отмена.
- Мышь — этап 6, в этом плане её нет.
- Коммиты — Conventional Commits на английском, **без** `Co-Authored-By` и пометок «Generated with Claude Code».
- Работа в ветке `feature/stage-3` от `main`; в `main` — только через PR.

## Решения этого этапа (уточняют спеку)

- Пункты меню, чьи функции появятся в этапах 4–6 (View, Edit, Copy, Rename/Move, Make directory, Delete, Delete permanently, Panels on/off, Command history, Save setup), присутствуют в меню и показывают диалог «Not implemented yet».
- F9 сразу открывает выпадающий список: `Left`, если активна левая панель, иначе `Right`.
- ←/→ в меню переключают выпадающие списки по кругу; ↑/↓ пропускают разделители и идут по кругу; Enter выполняет; Esc / F9 / F10 закрывают.
- В пунктах Brief / Full, режимах сортировки и «Show hidden files» текущее значение отмечено `√` (как в NC).
- Поле ввода: курсор всегда в конце (как командная строка); начальный текст «свежий» — первая напечатанная буква заменяет его, Backspace редактирует.
- В диалоге без поля ввода первая буква кнопки нажимает её (`y` — Yes, `/` — корень в выборе диска).
- `+` / `-` / `*` работают только с файлами, папки не трогают (как в NC). Маска — один шаблон `filepath.Match` без учёта регистра, по умолчанию `*`.
- Сортировка: `..` первой; кроме Unsorted, папки перед файлами. Extension — по расширению, затем по имени; Time — новые сверху; Size — большие сверху (папки между собой по имени). Unsorted — порядок каталога, для этого `fs.ReadDir` читает каталог без сортировки.
- Ошибки (`a.report`) показываются красным диалогом «Error» с кнопкой OK вместо строки в командной строке; текст переносится по 60 колонок.
- Быстрый поиск Alt-буква в этап не входит — issue #3.

## Review Focus

1. **Пункт меню открывает новое окно** (Drive…, Help, Quit, маски) — меню закрывается, а открытое им окно остаётся на экране и получает клавиши. Тесты: `TestStackKeepsViewPushedByFinishingOne` (Task 1), `TestMenuDriveOpensDialog` (Task 8).
2. **Длинная ошибка** (глубокий путь без пробелов) — диалог не шире экрана, текст перенесён. Тесты: `TestWrap*` (Task 1), `TestLongErrorIsWrapped` (Task 7).
3. **Изменение размера / маленькое окно при открытом диалоге** — нет паники, клавиши по-прежнему идут диалогу. Тест: `TestSmallWindowWithDialog` (Task 7).
4. **Неверная маска** (`[`) — красный диалог ошибки, выделение не меняется. Тесты: `TestSelectMaskBadPattern` (Task 5), `TestBadMaskShowsError` (Task 8).
5. **Alt-F1 / `Esc` F1 не открывают справку**, а открывают выбор диска; Alt-F2 не открывает меню. Тест: `TestAltF1OpensDriveNotHelp` (Task 8).

---

## Карта файлов

```
internal/term/palette.go        + стили диалога, кнопок, поля, меню, тени; коды для golden
internal/term/text.go           + Wrap
internal/ui/stack.go            View, Stack
internal/ui/window.go           window (заливка + тень), drawTitle
internal/ui/input.go            Input
internal/ui/dialog.go           Dialog
internal/ui/menu.go             Item, Menu, MenuBar
internal/ui/textview.go         TextView (справка)
internal/fs/entry.go            ReadDir без сортировки
internal/fs/drives.go           Drive, Drives
internal/panel/sort.go          SortMode, SortEntries(es, mode)
internal/panel/panel.go         поле Sort, SetSort
internal/panel/select.go        SelectMask, InvertSelection
internal/app/app.go             стек окон, маршрутизация клавиш, report, новые клавиши
internal/app/draw.go            panelSpan, отрисовка окон
internal/app/dialogs.go         confirmQuit, showHelp, notImplemented, chooseDrive, askMask
internal/app/help.go            текст справки
internal/app/menu.go            содержимое меню
```

---

### Task 1: Стили, перенос текста, стек окон и диалог

**Files:**
- Modify: `internal/term/palette.go`, `internal/term/text.go`
- Create: `internal/ui/stack.go`, `internal/ui/window.go`, `internal/ui/input.go`, `internal/ui/dialog.go`
- Test: `internal/term/text_test.go`, `internal/term/palette_test.go`, `internal/ui/stack_test.go`, `internal/ui/dialog_test.go`

**Interfaces:**
- Consumes: `term.Canvas` (`Put`, `Text`, `Fill`, `HLine`, `Box`), `term.Width`, `term.Tail`, `term.Fit`, `termtest.NewScreen/Dump/Golden`.
- Produces:
  - стили `term.DialogStyle, DialogFrameStyle, ButtonStyle, ButtonFocusStyle, InputStyle, MenuStyle, MenuSelStyle, ShadowStyle tcell.Style`
  - `term.Wrap(s string, w int) []string`
  - `type ui.View interface { Draw(c term.Canvas, w, h int); HandleKey(ev *tcell.EventKey) bool }` — `HandleKey` возвращает `true`, когда окно закончило работу и его надо убрать
  - `type ui.Stack` с методами `Push(View)`, `Empty() bool`, `Len() int`, `Top() View`, `Draw(c term.Canvas, w, h int)`, `HandleKey(ev *tcell.EventKey)`
  - `ui.NewInput(text string) *Input`; поле `Input.Text string`; `(*Input).HandleKey(ev) bool`; `(*Input).Draw(c, x, y, w int)`
  - `type ui.Dialog struct { Title string; Lines []string; Input *Input; Buttons []string; Focus int; Danger bool; Over func(w int) (x, width int); Done func(button int, text string) }`, реализует `View`; `Done` получает индекс кнопки или `-1` на Esc
  - внутренние `window(c, x, y, w, h int, st tcell.Style)` и `drawTitle(c, x, y, w int, title string, st tcell.Style)` — их используют Task 2 и Task 3

- [ ] **Step 1: Создать ветку**

```bash
git checkout main && git pull && git checkout -b feature/stage-3
```

- [ ] **Step 2: Написать падающие тесты `term`**

`internal/term/text_test.go` — добавить в конец:

```go
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
```

(в импорт `text_test.go` добавить `"strings"`.)

Создать `internal/term/palette_test.go`:

```go
package term

import "testing"

func TestDialogStyleCodes(t *testing.T) {
	cases := []struct {
		name string
		code rune
		want rune
	}{
		{"dialog", StyleCode(DialogStyle), 'g'},
		{"frame", StyleCode(DialogFrameStyle), 'G'},
		{"button", StyleCode(ButtonStyle), 'b'},
		{"focus", StyleCode(ButtonFocusStyle), 'B'},
		{"input", StyleCode(InputStyle), 'c'},
		{"menu", StyleCode(MenuStyle), 'c'},
		{"menusel", StyleCode(MenuSelStyle), 'w'},
		{"shadow", StyleCode(ShadowStyle), 'x'},
	}
	for _, c := range cases {
		if c.code != c.want {
			t.Errorf("%s: got %c want %c", c.name, c.code, c.want)
		}
	}
}
```

- [ ] **Step 3: Запустить — падают**

Run: `go test ./internal/term/`
Expected: FAIL — `undefined: Wrap`, `undefined: DialogStyle` и т. д.

- [ ] **Step 4: Реализовать `Wrap` и стили**

`internal/term/text.go` — добавить `"unicode/utf8"` в импорт и функцию:

```go
// Wrap splits s (NFC) into lines of at most w columns, breaking at the last
// space that fits or, inside a long word, anywhere.
func Wrap(s string, w int) []string {
	s = norm.NFC.String(s)
	w = max(w, 1)
	var lines []string
	for runewidth.StringWidth(s) > w {
		cut, used := 0, 0
		for i, r := range s {
			rw := runewidth.RuneWidth(r)
			if used+rw > w {
				break
			}
			used += rw
			cut = i + utf8.RuneLen(r)
		}
		if cut == 0 {
			_, cut = utf8.DecodeRuneInString(s)
		}
		if s[cut] != ' ' {
			if sp := strings.LastIndexByte(s[:cut], ' '); sp > 0 {
				cut = sp
			}
		}
		lines = append(lines, strings.TrimRight(s[:cut], " "))
		s = strings.TrimLeft(s[cut:], " ")
	}
	return append(lines, s)
}
```

(`s[cut]` безопасен: внутри цикла ширина `s` больше `w`, значит `cut < len(s)`.)

`internal/term/palette.go` — в блок `// Color roles.` добавить:

```go
	DialogStyle      = style(Black, LightGray)
	DialogFrameStyle = style(White, LightGray)
	ButtonStyle      = style(Black, White)
	ButtonFocusStyle = style(Black, Yellow)
	InputStyle       = style(Black, Cyan)
	MenuStyle        = style(Black, Cyan)
	MenuSelStyle     = style(White, Black)
	ShadowStyle      = style(Black, Black)
```

и в список в `init()` перед `{tcell.StyleDefault, '.'}`:

```go
		{DialogStyle, 'g'},
		{DialogFrameStyle, 'G'},
		{ButtonStyle, 'b'},
		{ButtonFocusStyle, 'B'},
		{MenuSelStyle, 'w'},
		{ShadowStyle, 'x'},
```

Поправить комментарий у `CursorStyle`: `// also ActiveTitleStyle, KeyLabelStyle, InputStyle, MenuStyle`.

- [ ] **Step 5: Запустить — проходят**

Run: `go test ./internal/term/`
Expected: PASS

- [ ] **Step 6: Написать падающие тесты `ui`**

`internal/ui/stack_test.go`:

```go
package ui_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

type fakeView struct{ onKey func() bool }

func (f *fakeView) Draw(term.Canvas, int, int)       {}
func (f *fakeView) HandleKey(*tcell.EventKey) bool { return f.onKey() }

func key(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, 0) }

func runeKey(r rune) *tcell.EventKey { return tcell.NewEventKey(tcell.KeyRune, r, 0) }

func TestStackRoutesToTop(t *testing.T) {
	var s ui.Stack
	var got []string
	s.Push(&fakeView{onKey: func() bool { got = append(got, "bottom"); return false }})
	s.Push(&fakeView{onKey: func() bool { got = append(got, "top"); return true }})
	s.HandleKey(key(tcell.KeyEnter))
	s.HandleKey(key(tcell.KeyEnter))
	if len(got) != 2 || got[0] != "top" || got[1] != "bottom" || s.Len() != 1 {
		t.Fatalf("got %v, len %d", got, s.Len())
	}
}

func TestStackKeepsViewPushedByFinishingOne(t *testing.T) {
	var s ui.Stack
	next := &fakeView{onKey: func() bool { return false }}
	s.Push(&fakeView{onKey: func() bool { s.Push(next); return true }})
	s.HandleKey(key(tcell.KeyEnter))
	if s.Len() != 1 || s.Top() != next {
		t.Fatalf("len %d, top %v", s.Len(), s.Top())
	}
}

func TestEmptyStack(t *testing.T) {
	var s ui.Stack
	s.HandleKey(key(tcell.KeyEnter)) // must not panic
	if !s.Empty() || s.Top() != nil {
		t.Fatal("stack not empty")
	}
}
```

`internal/ui/dialog_test.go`:

```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

type result struct {
	button int
	text   string
	calls  int
}

func dialog(r *result, buttons ...string) *ui.Dialog {
	return &ui.Dialog{
		Title:   "Quit",
		Lines:   []string{"Do you want to quit?"},
		Buttons: buttons,
		Done:    func(b int, text string) { r.button, r.text = b, text; r.calls++ },
	}
}

func TestDialogEnterPressesFocused(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	d.HandleKey(key(tcell.KeyRight))
	if !d.HandleKey(key(tcell.KeyEnter)) || r.button != 1 || r.calls != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestDialogFocusWraps(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	d.HandleKey(key(tcell.KeyLeft))
	if d.Focus != 1 {
		t.Fatalf("focus %d", d.Focus)
	}
	d.HandleKey(key(tcell.KeyTab))
	if d.Focus != 0 {
		t.Fatalf("focus %d", d.Focus)
	}
}

func TestDialogEscCancels(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	if !d.HandleKey(key(tcell.KeyEscape)) || r.button != -1 {
		t.Fatalf("%+v", r)
	}
}

func TestDialogLetterPressesButton(t *testing.T) {
	var r result
	d := dialog(&r, "Yes", "No")
	if d.HandleKey(runeKey('x')) || r.calls != 0 {
		t.Fatal("x pressed a button")
	}
	if !d.HandleKey(runeKey('n')) || r.button != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestDialogInput(t *testing.T) {
	var r result
	d := dialog(&r, "OK", "Cancel")
	d.Input = ui.NewInput("*")
	for _, c := range "ab" {
		d.HandleKey(runeKey(c)) // first letter replaces "*"
	}
	d.HandleKey(key(tcell.KeyBackspace))
	d.HandleKey(runeKey('n')) // goes to the field, not to a button
	if !d.HandleKey(key(tcell.KeyEnter)) || r.button != 0 || r.text != "an" {
		t.Fatalf("%+v", r)
	}
}

func TestInputBackspaceEditsFreshText(t *testing.T) {
	in := ui.NewInput("abc")
	in.HandleKey(key(tcell.KeyBackspace))
	in.HandleKey(runeKey('d'))
	if in.Text != "abd" {
		t.Fatalf("got %q", in.Text)
	}
}

func TestInputIgnoresAltRunes(t *testing.T) {
	in := ui.NewInput("")
	if in.HandleKey(tcell.NewEventKey(tcell.KeyRune, '.', tcell.ModAlt)) || in.Text != "" {
		t.Fatalf("got %q", in.Text)
	}
}

func background(t *testing.T, w, h int) tcell.SimulationScreen {
	s := termtest.NewScreen(t, w, h)
	term.Canvas{Screen: s}.Fill(0, 0, w, h, '.', term.PanelStyle)
	return s
}

func TestDialogGolden(t *testing.T) {
	var r result
	s := background(t, 50, 10)
	dialog(&r, "Yes", "No").Draw(term.Canvas{Screen: s}, 50, 10)
	termtest.Golden(t, "dialog", termtest.Dump(s))
}

func TestDialogInputGolden(t *testing.T) {
	s := background(t, 60, 10)
	d := &ui.Dialog{Title: "Select", Input: ui.NewInput("*.go"), Buttons: []string{"OK", "Cancel"}}
	d.Draw(term.Canvas{Screen: s}, 60, 10)
	termtest.Golden(t, "dialog_input", termtest.Dump(s))
}

func TestDangerDialogIsRed(t *testing.T) {
	var r result
	s := background(t, 50, 10)
	d := dialog(&r, "OK")
	d.Danger = true
	d.Draw(term.Canvas{Screen: s}, 50, 10)
	styles := strings.SplitN(termtest.Dump(s), "\n\n", 2)[1]
	if !strings.Contains(styles, "r") || strings.Contains(styles, "g") {
		t.Fatalf("not red:\n%s", styles)
	}
}

func TestDialogOverCentersInSpan(t *testing.T) {
	var r result
	s := background(t, 80, 10)
	d := dialog(&r, "Yes", "No")
	d.Over = func(w int) (int, int) { return w / 2, w - w/2 }
	d.Draw(term.Canvas{Screen: s}, 80, 10)
	row := []rune(strings.Split(termtest.Dump(s), "\n")[3])
	// Window is 28 wide, centered in columns 40..79: x = 46, frame from 48.
	if row[48] != '╔' || row[47] != ' ' {
		t.Fatalf("row %q", string(row))
	}
}
```

- [ ] **Step 7: Запустить — падают**

Run: `go test ./internal/ui/`
Expected: FAIL — пакета `ui` ещё нет (`no non-test Go files` / `undefined: ui.Stack`).

- [ ] **Step 8: Реализовать `Stack`**

`internal/ui/stack.go`:

```go
// Package ui has the modal windows drawn over the panels: dialogs, the
// pull-down menu and the help window.
package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// View is a modal window.
type View interface {
	// Draw draws the view on a screen of w×h cells.
	Draw(c term.Canvas, w, h int)
	// HandleKey processes a key and reports whether the view is finished
	// and should be removed.
	HandleKey(ev *tcell.EventKey) bool
}

// Stack holds the open modal windows; the last one gets the keys.
type Stack struct {
	views []View
}

func (s *Stack) Push(v View) { s.views = append(s.views, v) }
func (s *Stack) Empty() bool { return len(s.views) == 0 }
func (s *Stack) Len() int    { return len(s.views) }

func (s *Stack) Top() View {
	if s.Empty() {
		return nil
	}
	return s.views[len(s.views)-1]
}

// Draw draws the views from the bottom up.
func (s *Stack) Draw(c term.Canvas, w, h int) {
	for _, v := range s.views {
		v.Draw(c, w, h)
	}
}

// HandleKey sends ev to the top view. A finished view is removed even when
// it opened another view while handling the key.
func (s *Stack) HandleKey(ev *tcell.EventKey) {
	v := s.Top()
	if v == nil || !v.HandleKey(ev) {
		return
	}
	for i, x := range s.views {
		if x == v {
			s.views = append(s.views[:i], s.views[i+1:]...)
			return
		}
	}
}
```

- [ ] **Step 9: Реализовать окно, поле ввода и диалог**

`internal/ui/window.go`:

```go
package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// window fills a w×h rectangle and draws NC's shadow: two columns to the
// right and one row down.
func window(c term.Canvas, x, y, w, h int, st tcell.Style) {
	c.Fill(x+2, y+1, w, h, ' ', term.ShadowStyle)
	c.Fill(x, y, w, h, ' ', st)
}

// drawTitle centers " title " in row y of a window at x of width w.
func drawTitle(c term.Canvas, x, y, w int, title string, st tcell.Style) {
	if title == "" {
		return
	}
	t := " " + title + " "
	tw := term.Width(t)
	c.Text(x+(w-tw)/2, y, t, tw, st)
}
```

`internal/ui/input.go`:

```go
package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// Input is a one-line text field. The cursor stays at the end, like the NC
// command line. Until the first edit the initial text is fresh: typing
// replaces it, Backspace edits it.
type Input struct {
	Text  string
	fresh bool
}

func NewInput(text string) *Input {
	return &Input{Text: text, fresh: true}
}

// HandleKey edits the text and reports whether the key was used.
func (in *Input) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyBackspace:
		if rs := []rune(in.Text); len(rs) > 0 {
			in.Text = string(rs[:len(rs)-1])
		}
	case tcell.KeyRune:
		if ev.Modifiers()&(tcell.ModAlt|tcell.ModCtrl) != 0 {
			return false
		}
		if in.fresh {
			in.Text = ""
		}
		in.Text += string(ev.Rune())
	default:
		return false
	}
	in.fresh = false
	return true
}

// Draw draws the field w columns wide at (x, y) and puts the cursor after
// the text. A long text shows its end.
func (in *Input) Draw(c term.Canvas, x, y, w int) {
	c.HLine(x, y, w, ' ', term.InputStyle)
	n := c.Text(x, y, term.Tail(in.Text, w-1), w, term.InputStyle)
	c.Screen.ShowCursor(x+n, y)
}
```

`internal/ui/dialog.go`:

```go
package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

const inputMinWidth = 40

// Dialog is an NC dialog: a gray window with a double frame, centered text
// lines, an optional text field and a row of buttons.
type Dialog struct {
	Title   string
	Lines   []string
	Input   *Input // nil for no text field
	Buttons []string
	Focus   int  // focused button
	Danger  bool // white on red, for errors
	// Over gives the columns (x, width) to center the dialog in for a screen
	// w columns wide; nil centers it on the screen.
	Over func(w int) (x, width int)
	// Done is called with the pressed button, or -1 on Esc, and the text
	// of the field.
	Done func(button int, text string)
}

func (d *Dialog) HandleKey(ev *tcell.EventKey) bool {
	n := len(d.Buttons)
	switch ev.Key() {
	case tcell.KeyEscape:
		return d.finish(-1)
	case tcell.KeyEnter:
		return d.finish(d.Focus)
	case tcell.KeyLeft, tcell.KeyBacktab:
		d.Focus = (d.Focus + n - 1) % n
	case tcell.KeyRight, tcell.KeyTab:
		d.Focus = (d.Focus + 1) % n
	default:
		if d.Input != nil {
			d.Input.HandleKey(ev)
			return false
		}
		if ev.Key() == tcell.KeyRune {
			r := strings.ToLower(string(ev.Rune()))
			for i, b := range d.Buttons {
				if strings.HasPrefix(strings.ToLower(b), r) {
					return d.finish(i)
				}
			}
		}
	}
	return false
}

func (d *Dialog) finish(button int) bool {
	if d.Done != nil {
		text := ""
		if d.Input != nil {
			text = d.Input.Text
		}
		d.Done(button, text)
	}
	return true
}

func (d *Dialog) buttonsWidth() int {
	w := 0
	for i, b := range d.Buttons {
		if i > 0 {
			w += 2
		}
		w += term.Width(b) + 2
	}
	return w
}

// Draw lays the dialog out as: a 2-column, 1-row gray margin, the frame,
// one space of padding, and the content (lines, field, buttons).
func (d *Dialog) Draw(c term.Canvas, w, h int) {
	body, frame := term.DialogStyle, term.DialogFrameStyle
	if d.Danger {
		body, frame = term.ErrorStyle, term.ErrorStyle
	}
	cw := max(d.buttonsWidth(), term.Width(d.Title)+2)
	for _, l := range d.Lines {
		cw = max(cw, term.Width(l))
	}
	if d.Input != nil {
		cw = max(cw, inputMinWidth)
	}
	cw = min(cw, w-10)
	ww, wh := cw+8, len(d.Lines)+5
	if d.Input != nil {
		wh++
	}
	ox, ow := 0, w
	if d.Over != nil {
		ox, ow = d.Over(w)
	}
	x := max(ox+(ow-ww)/2, 0)
	y := max((h-wh)/2, 0)
	window(c, x, y, ww, wh, body)
	c.Box(x+2, y+1, ww-4, wh-2, frame)
	drawTitle(c, x, y+1, ww, d.Title, frame)

	cx, row := x+4, y+2
	for _, l := range d.Lines {
		lw := min(term.Width(l), cw)
		c.Text(cx+(cw-lw)/2, row, l, cw, body)
		row++
	}
	if d.Input != nil {
		d.Input.Draw(c, cx, row, cw)
		row++
	}
	bx := cx + max(cw-d.buttonsWidth(), 0)/2
	for i, b := range d.Buttons {
		st := term.ButtonStyle
		if i == d.Focus {
			st = term.ButtonFocusStyle
		}
		bw := term.Width(b) + 2
		c.Text(bx, row, " "+b+" ", bw, st)
		bx += bw + 2
	}
}
```

- [ ] **Step 10: Создать эталоны и проверить их глазами**

Run: `go test ./internal/ui/ -run Golden -update && go test ./internal/ui/`
Expected: PASS.

Открыть `internal/ui/testdata/dialog.golden` и проверить: окно 28×6 с левым верхним углом в (11, 2); рамка `╔` в колонке 13 строки 3, `╗` в колонке 36; ` Quit ` по центру рамки; строка `Do you want to quit?` в строке 4, колонки 15–34; кнопки ` Yes ` (стиль `B`) и ` No ` (стиль `b`) в строке 5; тень `x` — колонки 39–40 в строках 3–8 и строка 8 с колонки 13; вокруг — `.`/`p`. `dialog_input.golden`: поле шириной 40 в стиле `c` с текстом `*.go`, под ним кнопки.

- [ ] **Step 11: Проверить всё и закоммитить**

Run: `go vet ./... && go test ./...`
Expected: PASS

```bash
git add internal/term internal/ui
git commit -m "feat: add modal stack, dialog and input widgets"
```

---

### Task 2: Меню F9 (`MenuBar`)

**Files:**
- Create: `internal/ui/menu.go`
- Test: `internal/ui/menu_test.go`

**Interfaces:**
- Consumes: `window` (Task 1); `term.MenuStyle`, `term.MenuSelStyle`, `term.Fit`, `term.Width`, `Canvas.Box/Put/HLine/Text`.
- Produces:
  - `type ui.Item struct { Label, Key string; Checked bool; Action func() }` — пустой `Label` = разделитель
  - `type ui.Menu struct { Title string; Items []Item }`
  - `type ui.MenuBar struct { Menus []Menu; Cur, Sel int }`, реализует `View`
  - `ui.NewMenuBar(menus []Menu, cur int) *MenuBar` — открывает список `cur`, выбран первый пункт

- [ ] **Step 1: Написать падающие тесты**

`internal/ui/menu_test.go`:

```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func testMenus(log *[]string) []ui.Menu {
	act := func(s string) func() { return func() { *log = append(*log, s) } }
	return []ui.Menu{
		{Title: "Left", Items: []ui.Item{
			{Label: "Brief", Checked: true, Action: act("brief")},
			{Label: "Full", Action: act("full")},
			{},
			{Label: "Re-read", Key: "Control-R", Action: act("reread")},
		}},
		{Title: "Files", Items: []ui.Item{{Label: "Help", Key: "F1", Action: act("help")}}},
		{Title: "Right", Items: []ui.Item{
			{Label: "Brief", Action: act("rbrief")},
			{},
			{Label: "Full", Action: act("rfull")},
		}},
	}
}

func TestMenuUpDownSkipSeparatorsAndWrap(t *testing.T) {
	var log []string
	m := ui.NewMenuBar(testMenus(&log), 0)
	for _, want := range []int{1, 3, 0} {
		m.HandleKey(key(tcell.KeyDown))
		if m.Sel != want {
			t.Fatalf("down: sel %d want %d", m.Sel, want)
		}
	}
	m.HandleKey(key(tcell.KeyUp))
	if m.Sel != 3 {
		t.Fatalf("up: sel %d", m.Sel)
	}
	m.HandleKey(key(tcell.KeyHome))
	m.HandleKey(key(tcell.KeyEnd))
	if m.Sel != 3 {
		t.Fatalf("end: sel %d", m.Sel)
	}
}

func TestMenuLeftRightSwitchMenus(t *testing.T) {
	var log []string
	m := ui.NewMenuBar(testMenus(&log), 0)
	m.HandleKey(key(tcell.KeyDown))
	for _, want := range []int{1, 2, 0} {
		m.HandleKey(key(tcell.KeyRight))
		if m.Cur != want || m.Sel != 0 {
			t.Fatalf("right: cur %d sel %d want %d", m.Cur, m.Sel, want)
		}
	}
	m.HandleKey(key(tcell.KeyLeft))
	if m.Cur != 2 {
		t.Fatalf("left: cur %d", m.Cur)
	}
}

func TestMenuEnterRunsAction(t *testing.T) {
	var log []string
	m := ui.NewMenuBar(testMenus(&log), 2)
	m.HandleKey(key(tcell.KeyDown))
	if !m.HandleKey(key(tcell.KeyEnter)) || len(log) != 1 || log[0] != "rfull" {
		t.Fatalf("log %v", log)
	}
}

func TestMenuEscClosesWithoutAction(t *testing.T) {
	var log []string
	for _, k := range []tcell.Key{tcell.KeyEscape, tcell.KeyF9, tcell.KeyF10} {
		m := ui.NewMenuBar(testMenus(&log), 0)
		if !m.HandleKey(key(k)) {
			t.Fatalf("%v did not close", k)
		}
	}
	if len(log) != 0 {
		t.Fatalf("log %v", log)
	}
}

func TestMenuGolden(t *testing.T) {
	var log []string
	s := background(t, 50, 10)
	m := ui.NewMenuBar(testMenus(&log), 0)
	m.HandleKey(key(tcell.KeyDown))
	m.Draw(term.Canvas{Screen: s}, 50, 10)
	termtest.Golden(t, "menu", termtest.Dump(s))
}

func TestMenuDropdownStaysOnScreen(t *testing.T) {
	var log []string
	s := background(t, 24, 10)
	ui.NewMenuBar(testMenus(&log), 2).Draw(term.Canvas{Screen: s}, 24, 10)
	row := []rune(strings.Split(termtest.Dump(s), "\n")[1])
	if !strings.ContainsRune(string(row), '╗') {
		t.Fatalf("dropdown cut off: %q", string(row))
	}
}
```

- [ ] **Step 2: Запустить — падают**

Run: `go test ./internal/ui/ -run Menu`
Expected: FAIL — `undefined: ui.NewMenuBar`.

- [ ] **Step 3: Реализовать**

`internal/ui/menu.go`:

```go
package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// Item is one line of a pull-down menu; an empty Label makes a separator.
type Item struct {
	Label   string
	Key     string // hot key shown on the right
	Checked bool
	Action  func()
}

type Menu struct {
	Title string
	Items []Item
}

// MenuBar is the F9 menu: the bar in the top row with one pull-down open.
// Every menu must have at least one item that is not a separator.
type MenuBar struct {
	Menus []Menu
	Cur   int // open menu
	Sel   int // selected item
}

const barX = 2

func NewMenuBar(menus []Menu, cur int) *MenuBar {
	m := &MenuBar{Menus: menus}
	m.open(cur)
	return m
}

func (m *MenuBar) open(i int) {
	n := len(m.Menus)
	m.Cur = (i%n + n) % n
	m.Sel = -1
	m.move(1)
}

// move selects the next item that is not a separator in direction d,
// wrapping around.
func (m *MenuBar) move(d int) {
	items := m.Menus[m.Cur].Items
	n := len(items)
	for i := 1; i <= n; i++ {
		j := ((m.Sel+d*i)%n + n) % n
		if items[j].Label != "" {
			m.Sel = j
			return
		}
	}
}

func (m *MenuBar) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyF9, tcell.KeyF10:
		return true
	case tcell.KeyLeft:
		m.open(m.Cur - 1)
	case tcell.KeyRight:
		m.open(m.Cur + 1)
	case tcell.KeyUp:
		m.move(-1)
	case tcell.KeyDown:
		m.move(1)
	case tcell.KeyHome:
		m.Sel = -1
		m.move(1)
	case tcell.KeyEnd:
		m.Sel = len(m.Menus[m.Cur].Items)
		m.move(-1)
	case tcell.KeyEnter:
		if a := m.Menus[m.Cur].Items[m.Sel].Action; a != nil {
			a()
		}
		return true
	}
	return false
}

// titleX is the column where the " Title " of menu i starts.
func (m *MenuBar) titleX(i int) int {
	x := barX
	for _, mm := range m.Menus[:i] {
		x += term.Width(mm.Title) + 4
	}
	return x
}

func (m *MenuBar) Draw(c term.Canvas, w, h int) {
	c.HLine(0, 0, w, ' ', term.MenuStyle)
	for i, mm := range m.Menus {
		st := term.MenuStyle
		if i == m.Cur {
			st = term.MenuSelStyle
		}
		t := " " + mm.Title + " "
		c.Text(m.titleX(i), 0, t, term.Width(t), st)
	}
	m.drawDropdown(c, w)
}

func (m *MenuBar) drawDropdown(c term.Canvas, w int) {
	items := m.Menus[m.Cur].Items
	lw, kw := 0, 0
	for _, it := range items {
		lw = max(lw, term.Width(it.Label))
		kw = max(kw, term.Width(it.Key))
	}
	gap := ""
	if kw > 0 {
		gap = "  "
	}
	iw := 1 + lw + len(gap) + kw + 1
	bw, bh := iw+2, len(items)+2
	x := max(min(m.titleX(m.Cur), w-bw-2), 0)
	y := 1
	window(c, x, y, bw, bh, term.MenuStyle)
	c.Box(x, y, bw, bh, term.MenuStyle)
	for k, it := range items {
		row := y + 1 + k
		if it.Label == "" {
			c.Put(x, row, '╟', term.MenuStyle)
			c.HLine(x+1, row, iw, '─', term.MenuStyle)
			c.Put(x+bw-1, row, '╢', term.MenuStyle)
			continue
		}
		check := " "
		if it.Checked {
			check = "√"
		}
		key := strings.Repeat(" ", kw-term.Width(it.Key)) + it.Key
		st := term.MenuStyle
		if k == m.Sel {
			st = term.MenuSelStyle
		}
		c.Text(x+1, row, check+term.Fit(it.Label, lw)+gap+key+" ", iw, st)
	}
}
```

- [ ] **Step 4: Эталон и проверка**

Run: `go test ./internal/ui/ -run MenuGolden -update && go test ./internal/ui/`
Expected: PASS.

Открыть `internal/ui/testdata/menu.golden`: строка 0 — голубая полоса (`c`) с ` Left ` в колонках 2–7 в стиле `w`, ` Files ` с колонки 10, ` Right ` с колонки 19; с колонки 2 строки 1 — рамка выпадающего списка шириной 22 (`╔…╗`, колонки 2–23), пункт `√Brief` в строке 2, выделенный ` Full ` в строке 3 (стиль `w` на всю ширину пункта), `╟───╢` в строке 4, `Re-read  Control-R` в строке 5, тень `x` справа и снизу.

- [ ] **Step 5: Закоммитить**

Run: `go vet ./... && go test ./...`
Expected: PASS

```bash
git add internal/ui
git commit -m "feat: add pull-down menu bar widget"
```

---

### Task 3: Окно справки (`TextView`)

**Files:**
- Create: `internal/ui/textview.go`
- Test: `internal/ui/textview_test.go`

**Interfaces:**
- Consumes: `window`, `drawTitle` (Task 1), `term.DialogStyle`, `term.DialogFrameStyle`.
- Produces: `type ui.TextView struct { Title string; Lines []string; Top int }`, реализует `View`. Число видимых строк считается в `Draw`, поэтому прокрутка корректна после первой отрисовки.

- [ ] **Step 1: Написать падающие тесты**

`internal/ui/textview_test.go`:

```go
package ui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func helpView(t *testing.T) (*ui.TextView, tcell.SimulationScreen) {
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	s := background(t, 60, 12)
	v := &ui.TextView{Title: "Help", Lines: lines}
	v.Draw(term.Canvas{Screen: s}, 60, 12) // 12-6 = 6 visible rows
	return v, s
}

func TestTextViewScrollClamps(t *testing.T) {
	v, _ := helpView(t)
	steps := []struct {
		k    tcell.Key
		want int
	}{
		{tcell.KeyUp, 0},
		{tcell.KeyDown, 1},
		{tcell.KeyPgDn, 7},
		{tcell.KeyEnd, 24},
		{tcell.KeyDown, 24},
		{tcell.KeyPgUp, 18},
		{tcell.KeyHome, 0},
	}
	for _, st := range steps {
		if v.HandleKey(key(st.k)) {
			t.Fatalf("%v closed the view", st.k)
		}
		if v.Top != st.want {
			t.Fatalf("after %v: top %d want %d", st.k, v.Top, st.want)
		}
	}
}

func TestTextViewCloses(t *testing.T) {
	for _, k := range []tcell.Key{tcell.KeyEscape, tcell.KeyEnter, tcell.KeyF1, tcell.KeyF10} {
		v, _ := helpView(t)
		if !v.HandleKey(key(k)) {
			t.Fatalf("%v did not close", k)
		}
	}
}

func TestTextViewDraw(t *testing.T) {
	v, s := helpView(t)
	v.HandleKey(key(tcell.KeyDown))
	v.Draw(term.Canvas{Screen: s}, 60, 12)
	out := termtest.Dump(s)
	if !strings.Contains(out, " Help ") || !strings.Contains(out, "line 6") || strings.Contains(out, "line 0 ") || strings.Contains(out, "line 7") {
		t.Fatalf("unexpected view:\n%s", out)
	}
}
```

- [ ] **Step 2: Запустить — падают**

Run: `go test ./internal/ui/ -run TextView`
Expected: FAIL — `undefined: ui.TextView`.

- [ ] **Step 3: Реализовать**

`internal/ui/textview.go`:

```go
package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// TextView is a scrollable read-only window, used for help.
type TextView struct {
	Title string
	Lines []string
	Top   int  // first visible line
	rows  int  // visible lines, set by Draw
}

func (v *TextView) scroll(d int) {
	v.Top = max(min(v.Top+d, len(v.Lines)-v.rows), 0)
}

func (v *TextView) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter, tcell.KeyF1, tcell.KeyF10:
		return true
	case tcell.KeyUp:
		v.scroll(-1)
	case tcell.KeyDown:
		v.scroll(1)
	case tcell.KeyPgUp:
		v.scroll(-v.rows)
	case tcell.KeyPgDn:
		v.scroll(v.rows)
	case tcell.KeyHome:
		v.scroll(-len(v.Lines))
	case tcell.KeyEnd:
		v.scroll(len(v.Lines))
	}
	return false
}

// Draw uses the dialog layout: gray margin, double frame, one space padding.
func (v *TextView) Draw(c term.Canvas, w, h int) {
	cw := term.Width(v.Title) + 2
	for _, l := range v.Lines {
		cw = max(cw, term.Width(l))
	}
	cw = min(cw, w-10)
	v.rows = max(min(len(v.Lines), h-6), 1)
	v.scroll(0)
	ww, wh := cw+8, v.rows+4
	x, y := max((w-ww)/2, 0), max((h-wh)/2, 0)
	window(c, x, y, ww, wh, term.DialogStyle)
	c.Box(x+2, y+1, ww-4, wh-2, term.DialogFrameStyle)
	drawTitle(c, x, y+1, ww, v.Title, term.DialogFrameStyle)
	for i := 0; i < v.rows && v.Top+i < len(v.Lines); i++ {
		c.Text(x+4, y+2+i, v.Lines[v.Top+i], cw, term.DialogStyle)
	}
}
```

- [ ] **Step 4: Запустить — проходят**

Run: `go vet ./... && go test ./internal/ui/`
Expected: PASS

- [ ] **Step 5: Закоммитить**

```bash
git add internal/ui
git commit -m "feat: add scrollable text view for help"
```

---

### Task 4: Режимы сортировки панели

**Files:**
- Modify: `internal/fs/entry.go` (чтение каталога без сортировки)
- Modify: `internal/panel/sort.go`, `internal/panel/panel.go`
- Modify: `internal/panel/panel_test.go:47` (`SortEntries(es)` → `SortEntries(es, SortName)`)
- Test: `internal/panel/sort_test.go` (создать), `internal/panel/panel_test.go`

**Interfaces:**
- Consumes: `fs.Entry`, `fs.ReadDir`.
- Produces:
  - `type panel.SortMode int`; константы `panel.SortName, SortExt, SortTime, SortSize, Unsorted`
  - `panel.SortEntries(es []fs.Entry, mode SortMode)`
  - поле `Panel.Sort SortMode` (по умолчанию `SortName`); `(*Panel).SetSort(m SortMode) error` — перечитывает каталог, сохраняя курсор и выделение по имени

- [ ] **Step 1: Написать падающие тесты**

`internal/panel/sort_test.go`:

```go
package panel

import (
	"strings"
	"testing"
	"time"

	"github.com/navoznov/terminal-commander/internal/fs"
)

func sortDemo() []fs.Entry {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	return []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true},
		{Name: "b.txt", Size: 10, ModTime: t0.Add(3 * time.Hour)},
		{Name: "zdir", IsDir: true, Size: 900, ModTime: t0.Add(5 * time.Hour)},
		{Name: "a.md", Size: 30, ModTime: t0.Add(1 * time.Hour)},
		{Name: "Adir", IsDir: true, Size: 100, ModTime: t0},
		{Name: "c.go", Size: 20, ModTime: t0.Add(2 * time.Hour)},
		{Name: "Makefile", Size: 20, ModTime: t0.Add(2 * time.Hour)},
	}
}

func entryNames(es []fs.Entry) string {
	var ns []string
	for _, e := range es {
		ns = append(ns, e.Name)
	}
	return strings.Join(ns, " ")
}

func TestSortModes(t *testing.T) {
	cases := []struct {
		mode SortMode
		want string
	}{
		{SortName, ".. Adir zdir a.md b.txt c.go Makefile"},
		{SortExt, ".. Adir zdir Makefile c.go a.md b.txt"},
		{SortTime, ".. zdir Adir b.txt c.go Makefile a.md"},
		{SortSize, ".. Adir zdir a.md c.go Makefile b.txt"},
		{Unsorted, ".. b.txt zdir a.md Adir c.go Makefile"},
	}
	for _, c := range cases {
		es := sortDemo()
		SortEntries(es, c.mode)
		if got := entryNames(es); got != c.want {
			t.Errorf("mode %d: got %q want %q", c.mode, got, c.want)
		}
	}
}
```

(Пояснения к ожиданиям: у `Makefile` нет расширения — пустое расширение идёт первым; при Time/Size равные значения `c.go` и `Makefile` упорядочены по имени; при Size папки между собой — по имени.)

`internal/panel/panel_test.go` — добавить:

```go
func TestSetSortKeepsCursorAndSelection(t *testing.T) {
	dir := t.TempDir()
	for name, size := range map[string]int{"a.txt": 1, "b.txt": 50, "c.txt": 10} {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := New()
	if err := p.Load(dir); err != nil {
		t.Fatal(err)
	}
	p.Focus("c.txt")
	p.Selected["a.txt"] = true
	if err := p.SetSort(SortSize); err != nil {
		t.Fatal(err)
	}
	if got := entryNames(p.Entries); got != ".. b.txt c.txt a.txt" {
		t.Fatalf("order %q", got)
	}
	if p.Current().Name != "c.txt" || !p.Selected["a.txt"] || p.Sort != SortSize {
		t.Fatalf("cursor %s selected %v sort %d", p.Current().Name, p.Selected, p.Sort)
	}
}
```

(`os`, `path/filepath` и `must` в `panel_test.go` уже есть; `entryNames` — из `sort_test.go`.)

- [ ] **Step 2: Запустить — падают**

Run: `go test ./internal/panel/`
Expected: FAIL — `undefined: SortMode`, `too many arguments in call to SortEntries`.

- [ ] **Step 3: Реализовать**

`internal/fs/entry.go` — заменить `des, err := os.ReadDir(path)` / проверку ошибки на чтение без сортировки (`os.ReadDir` сортирует по имени, а режиму Unsorted нужен порядок каталога):

```go
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	des, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
```

Комментарий к `ReadDir` уже говорит «Entries after ".." are not sorted» — теперь это правда.

`internal/panel/sort.go` — заменить целиком:

```go
package panel

import (
	"sort"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/navoznov/terminal-commander/internal/fs"
)

type SortMode int

const (
	SortName SortMode = iota
	SortExt
	SortTime
	SortSize
	Unsorted
)

func sortKey(s string) string { return strings.ToLower(norm.NFC.String(s)) }

// ext returns the extension of name without the dot; ".zshrc" has none.
func ext(name string) string {
	if i := strings.LastIndexByte(name, '.'); i > 0 {
		return name[i+1:]
	}
	return ""
}

// SortEntries puts ".." first. Unsorted keeps the rest as is; other modes put
// directories before files and order each group by the mode, then by
// case-insensitive name. Time puts newest first, Size largest first
// (directories are ordered by name).
func SortEntries(es []fs.Entry, mode SortMode) {
	if mode == Unsorted {
		return
	}
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.IsUp != b.IsUp {
			return a.IsUp
		}
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		switch mode {
		case SortExt:
			if ea, eb := sortKey(ext(a.Name)), sortKey(ext(b.Name)); ea != eb {
				return ea < eb
			}
		case SortTime:
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.After(b.ModTime)
			}
		case SortSize:
			if !a.IsDir && a.Size != b.Size {
				return a.Size > b.Size
			}
		}
		if ka, kb := sortKey(a.Name), sortKey(b.Name); ka != kb {
			return ka < kb
		}
		return a.Name < b.Name
	})
}
```

`internal/panel/panel.go`:
- в структуру `Panel` после `ShowHidden bool` добавить поле `Sort SortMode`;
- в `read` заменить `SortEntries(es)` на `SortEntries(es, p.Sort)`;
- в конец файла добавить:

```go
// SetSort changes the sort mode and re-reads the directory, keeping the
// cursor and selection.
func (p *Panel) SetSort(m SortMode) error {
	p.Sort = m
	return p.Reload()
}
```

В `internal/panel/panel_test.go:47` заменить `SortEntries(es)` на `SortEntries(es, SortName)`.

- [ ] **Step 4: Запустить — проходят**

Run: `go vet ./... && go test ./...`
Expected: PASS. Тесты `fs` ищут записи через `find` и от порядка не зависят; `TestSortEntries` в `panel_test.go:47` вызывает `SortEntries(es)` — заменить на `SortEntries(es, SortName)`.

- [ ] **Step 5: Закоммитить**

```bash
git add internal/fs internal/panel
git commit -m "feat: add sort modes by extension, time, size and unsorted"
```

---

### Task 5: Выделение по маске и инверсия

**Files:**
- Create: `internal/panel/select.go`
- Test: `internal/panel/select_test.go`

**Interfaces:**
- Consumes: `Panel.Entries`, `Panel.Selected`.
- Produces:
  - `(*Panel).SelectMask(mask string, on bool) error` — выделяет (`on`) или снимает выделение с файлов, чьё имя подходит под шаблон `filepath.Match` без учёта регистра; папки не трогает; при неверном шаблоне возвращает `filepath.ErrBadPattern` и ничего не меняет
  - `(*Panel).InvertSelection()` — инвертирует выделение всех файлов (не папок)

- [ ] **Step 1: Написать падающие тесты**

`internal/panel/select_test.go`:

```go
package panel

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/navoznov/terminal-commander/internal/fs"
)

func selectDemo() *Panel {
	p := New()
	p.Entries = []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true},
		{Name: "docs.txt", IsDir: true},
		{Name: "a.txt"},
		{Name: "B.TXT"},
		{Name: "c.go"},
	}
	return p
}

// selected lists the selected names in panel order.
func selected(p *Panel) string {
	var ns []string
	for _, e := range p.Entries {
		if p.Selected[e.Name] {
			ns = append(ns, e.Name)
		}
	}
	return strings.Join(ns, " ")
}

func TestSelectMaskFilesOnlyIgnoringCase(t *testing.T) {
	p := selectDemo()
	if err := p.SelectMask("*.txt", true); err != nil {
		t.Fatal(err)
	}
	if got := selected(p); got != "a.txt B.TXT" {
		t.Fatalf("got %q", got)
	}
	if err := p.SelectMask("B*", false); err != nil {
		t.Fatal(err)
	}
	if got := selected(p); got != "a.txt" {
		t.Fatalf("got %q", got)
	}
}

func TestSelectMaskKeepsManualDirSelection(t *testing.T) {
	p := selectDemo()
	p.Selected["docs.txt"] = true
	if err := p.SelectMask("*", false); err != nil {
		t.Fatal(err)
	}
	if got := selected(p); got != "docs.txt" {
		t.Fatalf("got %q", got)
	}
}

func TestSelectMaskBadPattern(t *testing.T) {
	p := selectDemo()
	p.Selected["c.go"] = true
	err := p.SelectMask("[", true)
	if !errors.Is(err, filepath.ErrBadPattern) {
		t.Fatalf("err %v", err)
	}
	if got := selected(p); got != "c.go" {
		t.Fatalf("selection changed: %q", got)
	}
}

func TestInvertSelection(t *testing.T) {
	p := selectDemo()
	p.Selected["a.txt"] = true
	p.Selected["docs.txt"] = true
	p.InvertSelection()
	if got := selected(p); got != "docs.txt B.TXT c.go" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: Запустить — падают**

Run: `go test ./internal/panel/ -run 'Mask|Invert'`
Expected: FAIL — `p.SelectMask undefined`.

- [ ] **Step 3: Реализовать**

`internal/panel/select.go`:

```go
package panel

import "path/filepath"

// SelectMask selects (on) or unselects the files whose names match the shell
// pattern mask, ignoring case. Directories are left alone, as in NC.
func (p *Panel) SelectMask(mask string, on bool) error {
	mask = sortKey(mask)
	if _, err := filepath.Match(mask, ""); err != nil {
		return err
	}
	for _, e := range p.Entries {
		if e.IsDir {
			continue
		}
		if ok, _ := filepath.Match(mask, sortKey(e.Name)); !ok {
			continue
		}
		if on {
			p.Selected[e.Name] = true
		} else {
			delete(p.Selected, e.Name)
		}
	}
	return nil
}

// InvertSelection flips the selection of every file (not directories).
func (p *Panel) InvertSelection() {
	for _, e := range p.Entries {
		if e.IsDir {
			continue
		}
		if p.Selected[e.Name] {
			delete(p.Selected, e.Name)
		} else {
			p.Selected[e.Name] = true
		}
	}
}
```

(`filepath.Match(mask, "")` проверяет весь шаблон: с Go 1.16 `Match` возвращает `ErrBadPattern` для неверного шаблона даже при несовпадении.)

- [ ] **Step 4: Запустить — проходят**

Run: `go vet ./... && go test ./internal/panel/`
Expected: PASS

- [ ] **Step 5: Закоммитить**

```bash
git add internal/panel
git commit -m "feat: select, unselect and invert files by mask"
```

---

### Task 6: Список «дисков»

**Files:**
- Create: `internal/fs/drives.go`
- Test: `internal/fs/drives_test.go`

**Interfaces:**
- Consumes: —
- Produces:
  - `type fs.Drive struct { Name, Path string }`
  - `fs.Drives(home string) []Drive` — `/`, `~` (путь `home`, если он не пуст) и каждый элемент `/Volumes`, кроме скрытых (`.`-имена) и ссылки на загрузочный том (разрешается в `/`); тома — по имени

- [ ] **Step 1: Написать падающий тест**

`internal/fs/drives_test.go`:

```go
package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDrives(t *testing.T) {
	vol := t.TempDir()
	for _, name := range []string{"USB", "Backup", ".timemachine"} {
		if err := os.Mkdir(filepath.Join(vol, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("/", filepath.Join(vol, "Macintosh HD")); err != nil {
		t.Fatal(err)
	}
	got := drives(vol, "/Users/nc")
	want := []Drive{
		{"/", "/"},
		{"~", "/Users/nc"},
		{"Backup", filepath.Join(vol, "Backup")},
		{"USB", filepath.Join(vol, "USB")},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestDrivesWithoutVolumes(t *testing.T) {
	got := drives(filepath.Join(t.TempDir(), "missing"), "")
	if len(got) != 1 || got[0].Path != "/" {
		t.Fatalf("got %v", got)
	}
}
```

- [ ] **Step 2: Запустить — падают**

Run: `go test ./internal/fs/ -run Drives`
Expected: FAIL — `undefined: drives`.

- [ ] **Step 3: Реализовать**

`internal/fs/drives.go`:

```go
package fs

import (
	"os"
	"path/filepath"
	"strings"
)

// Drive is a place the drive dialog offers to jump to.
type Drive struct {
	Name, Path string
}

// Drives lists "/", the home directory as "~" and the mounted volumes.
func Drives(home string) []Drive {
	return drives("/Volumes", home)
}

// drives lists the entries of volumes by name, skipping hidden ones and the
// link to the boot volume.
func drives(volumes, home string) []Drive {
	ds := []Drive{{"/", "/"}}
	if home != "" {
		ds = append(ds, Drive{"~", home})
	}
	des, _ := os.ReadDir(volumes)
	for _, de := range des {
		if strings.HasPrefix(de.Name(), ".") {
			continue
		}
		p := filepath.Join(volumes, de.Name())
		if t, err := filepath.EvalSymlinks(p); err == nil && t == "/" {
			continue
		}
		ds = append(ds, Drive{de.Name(), p})
	}
	return ds
}
```

- [ ] **Step 4: Запустить — проходят**

Run: `go vet ./... && go test ./internal/fs/`
Expected: PASS

- [ ] **Step 5: Закоммитить**

```bash
git add internal/fs
git commit -m "feat: list drives for the drive dialog"
```

---

### Task 7: Окна в приложении: ошибки, выход F10, справка F1

**Files:**
- Modify: `internal/app/app.go`, `internal/app/draw.go`
- Create: `internal/app/dialogs.go`, `internal/app/help.go`
- Test: `internal/app/app_test.go`

**Interfaces:**
- Consumes: `ui.Stack`, `ui.Dialog`, `ui.TextView` (Task 1, 3), `term.Wrap`.
- Produces (для Task 8):
  - поле `App.modals ui.Stack`; поле `App.errMsg` удаляется
  - `(*App).report(err error)` — при `err != nil` открывает красный диалог «Error» с кнопкой OK
  - `(*App).confirmQuit()`, `(*App).showHelp()`
  - `panelSpan(i, w int) (x, width int)` в `draw.go` — колонки панели `i` на экране шириной `w`

- [ ] **Step 1: Обновить и написать тесты**

В `internal/app/app_test.go`:

1. Добавить в импорт `"errors"` и `"github.com/navoznov/terminal-commander/internal/ui"`.

2. Заменить `TestEscZeroQuits` на:

```go
func TestEscZeroAsksBeforeQuit(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyRune, '0', 0)
	if a.quit {
		t.Fatal("quit without asking")
	}
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != "Do you want to quit Terminal Commander?" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyEnter, 0, 0)
	if !a.quit {
		t.Fatal("Yes did not quit")
	}
}

func TestQuitNo(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF10, 0, 0)
	press(a, tcell.KeyRight, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	if a.quit || !a.modals.Empty() {
		t.Fatalf("quit %v, modals %d", a.quit, a.modals.Len())
	}
}
```

3. Заменить `TestEnterUnreadableDirShowsError` на:

```go
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
	d, ok := a.modals.Top().(*ui.Dialog)
	if p.Path != dir || !ok || !d.Danger {
		t.Fatalf("path %s top %#v", p.Path, a.modals.Top())
	}
	if !strings.Contains(strings.Join(d.Lines, " "), "permission denied") {
		t.Fatalf("lines %q", d.Lines)
	}
	if !strings.Contains(termtest.Dump(a.screen.(tcell.SimulationScreen)), " Error ") {
		t.Fatal("error dialog not drawn")
	}
	press(a, tcell.KeyDown, 0, 0) // goes to the dialog, not the panel
	if p.Current().Name != "locked" {
		t.Fatalf("panel moved to %s", p.Current().Name)
	}
	press(a, tcell.KeyEnter, 0, 0)
	if !a.modals.Empty() {
		t.Fatal("OK did not close the error")
	}
}
```

4. Добавить:

```go
func TestLongErrorIsWrapped(t *testing.T) {
	a, _ := newApp(t)
	a.report(errors.New(strings.Repeat("x", 300)))
	a.Draw()
	d := a.modals.Top().(*ui.Dialog)
	if len(d.Lines) != 5 {
		t.Fatalf("lines %q", d.Lines)
	}
	for _, l := range d.Lines {
		if len(l) > 60 {
			t.Fatalf("line too long: %d", len(l))
		}
	}
}

func TestSmallWindowWithDialog(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF10, 0, 0)
	s := a.screen.(tcell.SimulationScreen)
	s.SetSize(40, 10)
	a.HandleEvent(tcell.NewEventResize(40, 10))
	a.Draw()
	press(a, tcell.KeyRight, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	if a.quit || !a.modals.Empty() {
		t.Fatalf("quit %v, modals %d", a.quit, a.modals.Len())
	}
}

func TestF1ShowsHelp(t *testing.T) {
	a, _ := newApp(t)
	cur := a.panels[0].Cursor
	press(a, tcell.KeyF1, 0, 0)
	if _, ok := a.modals.Top().(*ui.TextView); !ok {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyDown, 0, 0)
	if a.panels[0].Cursor != cur {
		t.Fatal("key reached the panel")
	}
	if !strings.Contains(termtest.Dump(a.screen.(tcell.SimulationScreen)), " Help ") {
		t.Fatal("help not drawn")
	}
	press(a, tcell.KeyEscape, 0, 0)
	if !a.modals.Empty() {
		t.Fatal("Esc did not close help")
	}
}
```

5. В `TestScreenGolden` вынести подготовку сцены в функцию, чтобы Task 8 могла снять эталон с меню:

```go
func goldenApp(t *testing.T) *App {
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
	return a
}

func TestScreenGolden(t *testing.T) {
	a := goldenApp(t)
	termtest.Golden(t, "screen", termtest.Dump(a.screen.(tcell.SimulationScreen)))
}

func TestQuitDialogGolden(t *testing.T) {
	a := goldenApp(t)
	press(a, tcell.KeyF10, 0, 0)
	termtest.Golden(t, "quit", termtest.Dump(a.screen.(tcell.SimulationScreen)))
}
```

- [ ] **Step 2: Запустить — падают**

Run: `go test ./internal/app/`
Expected: FAIL — `a.modals undefined`, `a.report` / `errMsg` и т. п.

- [ ] **Step 3: Реализовать**

`internal/app/app.go`:
- в импорт добавить `"github.com/navoznov/terminal-commander/internal/term"` и `"github.com/navoznov/terminal-commander/internal/ui"`;
- в `App` заменить поле `errMsg string // shown in the command line until the next key` на `modals ui.Stack`;
- `HandleEvent`, ветку `*tcell.EventKey` заменить на:

```go
	case *tcell.EventKey:
		ev = a.keys.Feed(ev, ev.When())
		if a.modals.Empty() {
			a.handleKey(ev)
		} else {
			a.modals.HandleKey(ev)
		}
```

- `report` заменить на:

```go
// report shows err, if any, in a red dialog.
func (a *App) report(err error) {
	if err == nil {
		return
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Error",
		Lines:   term.Wrap(err.Error(), 60),
		Buttons: []string{"OK"},
		Danger:  true,
	})
}
```

- в `handleKey` заменить `case tcell.KeyF10: a.quit = true` на `case tcell.KeyF10: a.confirmQuit()` и добавить:

```go
	case tcell.KeyF1:
		if ev.Modifiers()&tcell.ModAlt == 0 {
			a.showHelp()
		}
```

`internal/app/dialogs.go`:

```go
package app

import "github.com/navoznov/terminal-commander/internal/ui"

func (a *App) confirmQuit() {
	a.modals.Push(&ui.Dialog{
		Title:   "Terminal Commander",
		Lines:   []string{"Do you want to quit Terminal Commander?"},
		Buttons: []string{"Yes", "No"},
		Done: func(b int, _ string) {
			if b == 0 {
				a.quit = true
			}
		},
	})
}

func (a *App) showHelp() {
	a.modals.Push(&ui.TextView{Title: "Help", Lines: helpLines})
}
```

`internal/app/help.go`:

```go
package app

var helpLines = []string{
	"Arrows              Move the cursor",
	"Home / End          First / last file",
	"PgUp / PgDn         Page up / down",
	"Enter               Enter directory, run or open file",
	"Backspace           Parent directory",
	"Control-PgUp        Parent directory",
	"Tab                 Other panel",
	"Insert, Space       Select file and move down",
	"+ / -               Select / unselect files by mask",
	"*                   Invert selection",
	"",
	"F1                  Help",
	"F2, F9              Menu",
	"F3                  View",
	"F4                  Edit",
	"F5                  Copy",
	"F6                  Rename or move",
	"F7                  Make directory",
	"F8                  Move to Trash",
	"Shift-F8            Delete permanently",
	"F10                 Quit",
	"",
	"Alt-F1 / Alt-F2     Left / right drive",
	"Alt-.               Show hidden files",
	"Control-O           Panels on/off",
	"Control-R           Re-read panel",
	"Control-T           Brief / full mode",
	"Control-U           Swap panels",
	"Control-Enter       Put name into command line",
	"Control-E / X       Previous / next command",
	"",
	"Esc then a key works as Alt (Option) + key:",
	"Esc 1 ... Esc 0 = F1 ... F10, Esc . = Alt-.",
}
```

`internal/app/draw.go`:
- добавить функцию и использовать её в цикле панелей `Draw` вместо ручного расчёта `x, pw`:

```go
// panelSpan returns the columns of panel i on a screen w columns wide; the
// right panel gets the odd column.
func panelSpan(i, w int) (x, width int) {
	lw := w / 2
	if i == 0 {
		return 0, lw
	}
	return lw, w - lw
}
```

```go
	ph := h - 2
	for i, p := range a.panels {
		x, pw := panelSpan(i, w)
		p.SetRows(ph - 5)
		p.Draw(c, x, 0, pw, ph, i == a.active, a.home)
	}
	a.drawCmdLine(c, h-2, w)
	a.drawKeyBar(c, h-1, w)
	if !a.modals.Empty() {
		a.screen.HideCursor()
		a.modals.Draw(c, w, h)
	}
```

- из `drawCmdLine` убрать блок `if a.errMsg != "" { ... }`.

- [ ] **Step 4: Эталоны и проверка**

Run: `go test ./internal/app/ -run QuitDialogGolden -update && go vet ./... && go test ./...`
Expected: PASS; `screen.golden` не изменился (`git diff --stat internal/app/testdata/screen.golden` пуст).

Открыть `internal/app/testdata/quit.golden`: серый диалог по центру экрана 80×25 поверх панелей, заголовок ` Terminal Commander `, кнопка ` Yes ` в стиле `B`, ` No ` в стиле `b`, тень `x`.

- [ ] **Step 5: Ручная проверка**

Run: `make build && ./tc`
Проверить: F1 — справка, ↑/↓/PgDn прокручивают, Esc закрывает; F10 — вопрос, `n` / Esc — остаться, Enter — выйти; `Esc` `0` — тот же вопрос; Enter на папке без прав — красный диалог Error.

- [ ] **Step 6: Закоммитить**

```bash
git add internal/app
git commit -m "feat: show errors, quit confirmation and help in dialogs"
```

---

### Task 8: Меню F9/F2, выбор диска, маски, заглушки

**Files:**
- Create: `internal/app/menu.go`
- Modify: `internal/app/app.go`, `internal/app/dialogs.go`
- Modify: `docs/terminal-compat.md` (строка про переключение режима через меню)
- Test: `internal/app/app_test.go`

**Interfaces:**
- Consumes: `ui.NewMenuBar`, `ui.Menu`, `ui.Item`, `ui.Dialog`, `ui.NewInput` (Task 1–2); `panel.SortMode` и константы, `(*Panel).SetSort` (Task 4); `(*Panel).SelectMask`, `(*Panel).InvertSelection` (Task 5); `fs.Drives`, `fs.Drive` (Task 6); `report`, `confirmQuit`, `showHelp`, `panelSpan` (Task 7).
- Produces: `(*App).openMenu()`, `(*App).chooseDrive(i int)`, `(*App).askMask(on bool)`, `(*App).notImplemented()`, `(*App).swapPanels()`, `driveLabel(name string) string`.

- [ ] **Step 1: Написать падающие тесты**

В `internal/app/app_test.go` добавить:

```go
func TestF9OpensMenuOfActivePanel(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF9, 0, 0)
	if m, ok := a.modals.Top().(*ui.MenuBar); !ok || m.Menus[m.Cur].Title != "Left" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyTab, 0, 0)
	press(a, tcell.KeyF2, 0, 0)
	if m, ok := a.modals.Top().(*ui.MenuBar); !ok || m.Menus[m.Cur].Title != "Right" {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestMenuFullMode(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyDown, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	if a.panels[0].Mode != panel.Full || !a.modals.Empty() {
		t.Fatalf("mode %v modals %d", a.panels[0].Mode, a.modals.Len())
	}
}

func TestMenuSortBySize(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF9, 0, 0)
	for i := 0; i < 5; i++ { // Brief → Full → Name → Extension → Time → Size
		press(a, tcell.KeyDown, 0, 0)
	}
	press(a, tcell.KeyEnter, 0, 0)
	if a.panels[0].Sort != panel.SortSize {
		t.Fatalf("sort %v", a.panels[0].Sort)
	}
	press(a, tcell.KeyF9, 0, 0)
	m := a.modals.Top().(*ui.MenuBar)
	if !m.Menus[0].Items[6].Checked || m.Menus[0].Items[3].Checked {
		t.Fatal("Size is not the checked sort mode")
	}
}

func TestMenuDriveOpensDialog(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyEnd, 0, 0) // Drive…
	press(a, tcell.KeyEnter, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || a.modals.Len() != 1 || d.Buttons[0] != "/" {
		t.Fatalf("top %#v len %d", a.modals.Top(), a.modals.Len())
	}
	press(a, tcell.KeyEnter, 0, 0)
	if a.panels[0].Path != "/" || !a.modals.Empty() {
		t.Fatalf("path %s", a.panels[0].Path)
	}
}

func TestAltF1OpensDriveNotHelp(t *testing.T) {
	a, dir := newApp(t)
	press(a, tcell.KeyF1, 0, tcell.ModAlt)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != "Choose left drive:" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyEscape, 0, 0) // Esc, then F2 = Alt-F2
	press(a, tcell.KeyF2, 0, 0)
	d, ok = a.modals.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != "Choose right drive:" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, '~', 0)
	if a.panels[1].Path != a.home || a.panels[0].Path != dir {
		t.Fatalf("left %s right %s", a.panels[0].Path, a.panels[1].Path)
	}
}

func TestDriveLabel(t *testing.T) {
	if got := driveLabel("USB"); got != "USB" {
		t.Fatalf("got %q", got)
	}
	if got := driveLabel("A very long volume"); got != "A very long}" {
		t.Fatalf("got %q", got)
	}
}

func TestPlusSelectsByMask(t *testing.T) {
	a, dir := newApp(t)
	must(t, os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "b.md"), nil, 0o644))
	must(t, a.panels[0].Reload())
	press(a, tcell.KeyRune, '+', 0)
	for _, r := range "*.txt" {
		press(a, tcell.KeyRune, r, 0)
	}
	press(a, tcell.KeyEnter, 0, 0)
	p := a.panels[0]
	if !p.Selected["a.txt"] || p.Selected["b.md"] || p.Selected["sub"] {
		t.Fatalf("selected %v", p.Selected)
	}
	press(a, tcell.KeyRune, '-', 0)
	press(a, tcell.KeyEnter, 0, 0) // default mask "*"
	press(a, tcell.KeyRune, '*', 0)
	if !p.Selected["a.txt"] || !p.Selected["b.md"] || p.Selected["sub"] {
		t.Fatalf("after - and *: %v", p.Selected)
	}
}

func TestBadMaskShowsError(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyRune, '+', 0)
	press(a, tcell.KeyRune, '[', 0)
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || !d.Danger {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestNotImplementedItem(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyRight, 0, 0) // Files
	press(a, tcell.KeyDown, 0, 0)  // View
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || d.Lines[0] != "Not implemented yet" {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestMenuGolden(t *testing.T) {
	a := goldenApp(t)
	press(a, tcell.KeyF9, 0, 0)
	termtest.Golden(t, "menu", termtest.Dump(a.screen.(tcell.SimulationScreen)))
}
```

(`TestMenuDriveOpensDialog` и `TestAltF1OpensDriveNotHelp` полагаются на то, что первые две кнопки — всегда `/` и `~`: `fs.Drives` добавляет их до томов.)

- [ ] **Step 2: Запустить — падают**

Run: `go test ./internal/app/`
Expected: FAIL — `undefined: driveLabel`, F9 не открывает меню.

- [ ] **Step 3: Реализовать**

`internal/app/menu.go`:

```go
package app

import (
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// openMenu opens the F9 menu at the Left or Right menu of the active panel.
func (a *App) openMenu() {
	menus := a.menus()
	cur := 0
	if a.active == 1 {
		cur = len(menus) - 1
	}
	a.modals.Push(ui.NewMenuBar(menus, cur))
}

func (a *App) menus() []ui.Menu {
	return []ui.Menu{
		{Title: "Left", Items: a.panelItems(0)},
		{Title: "Files", Items: []ui.Item{
			{Label: "Help", Key: "F1", Action: a.showHelp},
			{Label: "View", Key: "F3", Action: a.notImplemented},
			{Label: "Edit", Key: "F4", Action: a.notImplemented},
			{Label: "Copy", Key: "F5", Action: a.notImplemented},
			{Label: "Rename/Move", Key: "F6", Action: a.notImplemented},
			{Label: "Make directory", Key: "F7", Action: a.notImplemented},
			{Label: "Delete", Key: "F8", Action: a.notImplemented},
			{Label: "Delete permanently", Key: "Shift-F8", Action: a.notImplemented},
			{},
			{Label: "Select group", Key: "+", Action: func() { a.askMask(true) }},
			{Label: "Unselect group", Key: "-", Action: func() { a.askMask(false) }},
			{Label: "Invert selection", Key: "*", Action: func() { a.panels[a.active].InvertSelection() }},
			{},
			{Label: "Quit", Key: "F10", Action: a.confirmQuit},
		}},
		{Title: "Commands", Items: []ui.Item{
			{Label: "Swap panels", Key: "Control-U", Action: a.swapPanels},
			{Label: "Panels on/off", Key: "Control-O", Action: a.notImplemented},
			{Label: "Command history", Action: a.notImplemented},
		}},
		{Title: "Options", Items: []ui.Item{
			{Label: "Show hidden files", Key: "Alt-.", Checked: a.showHidden, Action: a.toggleHidden},
			{Label: "Save setup", Action: a.notImplemented},
		}},
		{Title: "Right", Items: a.panelItems(1)},
	}
}

func (a *App) panelItems(i int) []ui.Item {
	p := a.panels[i]
	mode := func(label string, m panel.Mode) ui.Item {
		return ui.Item{Label: label, Checked: p.Mode == m, Action: func() { p.SetMode(m) }}
	}
	sort := func(label string, m panel.SortMode) ui.Item {
		return ui.Item{Label: label, Checked: p.Sort == m, Action: func() { a.report(p.SetSort(m)) }}
	}
	return []ui.Item{
		mode("Brief", panel.Brief),
		mode("Full", panel.Full),
		{},
		sort("Name", panel.SortName),
		sort("Extension", panel.SortExt),
		sort("Time", panel.SortTime),
		sort("Size", panel.SortSize),
		sort("Unsorted", panel.Unsorted),
		{},
		{Label: "Re-read", Action: func() { a.report(p.Reload()) }},
		{Label: "Drive…", Action: func() { a.chooseDrive(i) }},
	}
}
```

`internal/app/dialogs.go` — дописать импорты (`"github.com/navoznov/terminal-commander/internal/fs"`, `"github.com/navoznov/terminal-commander/internal/term"`) и функции:

```go
func (a *App) notImplemented() {
	a.modals.Push(&ui.Dialog{Lines: []string{"Not implemented yet"}, Buttons: []string{"OK"}})
}

const driveLabelWidth = 12

// driveLabel cuts a volume name to 12 columns, NC style.
func driveLabel(name string) string {
	if term.Width(name) > driveLabelWidth {
		return term.Fit(name, driveLabelWidth)
	}
	return name
}

// chooseDrive opens the drive dialog over panel i.
func (a *App) chooseDrive(i int) {
	drives := fs.Drives(a.home)
	labels := make([]string, len(drives))
	for k, d := range drives {
		labels[k] = driveLabel(d.Name)
	}
	side := "left"
	if i == 1 {
		side = "right"
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Drive",
		Lines:   []string{"Choose " + side + " drive:"},
		Buttons: labels,
		Over:    func(w int) (int, int) { return panelSpan(i, w) },
		Done: func(b int, _ string) {
			if b >= 0 {
				a.report(a.panels[i].Load(drives[b].Path))
			}
		},
	})
}

// askMask asks for a mask and selects (on) or unselects matching files in
// the active panel.
func (a *App) askMask(on bool) {
	title := "Select"
	if !on {
		title = "Unselect"
	}
	a.modals.Push(&ui.Dialog{
		Title:   title,
		Input:   ui.NewInput("*"),
		Buttons: []string{"OK", "Cancel"},
		Done: func(b int, mask string) {
			if b == 0 {
				a.report(a.panels[a.active].SelectMask(mask, on))
			}
		},
	})
}
```

`internal/app/app.go`:
- вынести обмен панелей в метод и использовать его в `case tcell.KeyCtrlU:`:

```go
func (a *App) swapPanels() {
	a.panels[0], a.panels[1] = a.panels[1], a.panels[0]
	a.active = 1 - a.active
}
```

- в `handleKey` заменить ветку `case tcell.KeyF1:` из Task 7 и добавить F2, F9:

```go
	case tcell.KeyF1:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			a.chooseDrive(0)
		} else {
			a.showHelp()
		}
	case tcell.KeyF2:
		if ev.Modifiers()&tcell.ModAlt != 0 {
			a.chooseDrive(1)
		} else {
			a.openMenu()
		}
	case tcell.KeyF9:
		a.openMenu()
```

- в `handleRune` в `switch` добавить:

```go
	case r == '+' && mod == 0:
		a.askMask(true)
	case r == '-' && mod == 0:
		a.askMask(false)
	case r == '*' && mod == 0:
		p.InvertSelection()
```

`docs/terminal-compat.md` — строку `- режим панели переключается Control-T (и через меню F9 на этапе 3);` заменить на `- режим панели переключается Control-T или через меню F9 → Left/Right → Brief/Full;`.

- [ ] **Step 4: Эталоны и проверка**

Run: `go test ./internal/app/ -run MenuGolden -update && go vet ./... && go test ./...`
Expected: PASS.

Открыть `internal/app/testdata/menu.golden`: в строке 0 вместо верхних рамок — полоса `  Left    Files    Commands    Options    Right` (`c`, ` Left ` в стиле `w`); под ` Left ` — выпадающий список с `√Brief` (выделен, `w`), `Full`, разделителями, `√Name`, …, `Re-read`, `Drive…`; тень `x` справа и снизу; остальной экран как в `screen.golden`.

- [ ] **Step 5: Ручная проверка**

Run: `make build && ./tc`
Проверить в iTerm2 и Orca:
- F9 и F2 открывают меню; ←/→ переключают списки; ↑/↓ пропускают разделители; Esc закрывает.
- Left → Full, Left → Size — панель меняется, `√` стоит на выбранном.
- Left → Drive… и `Esc` `F1` / `Esc` `F2` — диалог над своей панелью; `/`, `~`, подключённые тома; Enter переходит; Esc отменяет.
- Files → Copy — «Not implemented yet».
- `+`, ввести `*.go`, Enter — выделены только файлы `.go`; `-`, Enter — выделение снято; `*` — инверсия.
- Options → Show hidden files — скрытые файлы показаны, `√` в меню.

- [ ] **Step 6: Закоммитить**

```bash
git add internal/app docs/terminal-compat.md
git commit -m "feat: add F9 menu, drive dialog and group selection"
```

---

## После всех задач

- `go vet ./... && go test ./...` — зелёные.
- Пуш ветки `feature/stage-3` и PR в `main` (по согласованию с пользователем).
