# Terminal Commander — этап 5: командная строка. План реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Командная строка NC: набор команды под панелями, Enter —
выполнение через `$SHELL -c` с показом вывода, встроенный `cd`,
история (Control-E / Control-X, пункт меню «Command history»),
вставка имени файла (Control-Enter), Control-O — показать экран
терминала; Enter на программе запускает её, на документе — `open`.

**Architecture:** Новый пакет `internal/shell` без `tcell`: `Line`
(текст строки и история), разбор `cd`, экранирование имён, проверка
«это программа», `Console` — запуск команды в терминале и ожидание
клавиши в сыром режиме (`golang.org/x/term`). `internal/app` держит
`shell.Line` и `shell.Console`, маршрутизирует печатные клавиши в
строку, а внешние команды выполняет между `screen.Suspend()` и
`screen.Resume()`. В `internal/ui` — окно списка `ui.List` для истории.

**Tech Stack:** Go 1.27, `github.com/gdamore/tcell/v2` v2.13.x,
`golang.org/x/term` v0.37 (уже есть в `go.sum` как косвенная зависимость).

**Spec:** `docs/superpowers/specs/2026-09-28-terminal-commander-design.md`
(этап 5: «`shell`: командная строка, Control-O, история»; разделы
«Командная строка», «Control-O», «Внешние программы», таблица клавиш).

## Global Constraints

- Модуль `github.com/navoznov/terminal-commander`; бинарник `tc`.
- `fs`, `ops`, `shell` не импортируют `tcell` и ничего не знают об экране; `panel` не знает о другой панели, меню и командной строке; `ui` не импортирует `panel`, `fs`, `ops`, `shell`, `app`; только `app` знает обо всём.
- Командная строка: строка `H-2`, приглашение `<путь активной панели>>` (домашний каталог — `~`), lightgray / black. Курсор всегда в конце строки: стрелки управляют панелью.
- Редактирование: печать, Backspace, Esc — очистить строку.
- Если командная строка не пуста, печатные символы (включая Space и `+ - *`) и Backspace идут в строку. Пустая строка: Space / Insert — выделение, `+ - *` — маски, Backspace — на уровень выше.
- Enter: если строка не пуста — выполнить её; иначе: папка — войти, `..` — вверх, исполняемый файл — запустить, остальное — открыть через `open`.
- `cd <путь>` (включая `cd`, `cd ~`, `cd -`, `cd ..`) обрабатывается внутри — меняет каталог активной панели.
- Остальное: `screen.Suspend()`, печать `<prompt> <команда>`, запуск `$SHELL -c` (по умолчанию `/bin/zsh`) в каталоге активной панели с подключёнными stdin/stdout/stderr, затем «Press any key to continue…», `screen.Resume()`, перечитать панели.
- История: до 500 команд, без повторов подряд. Control-E / Control-X — предыдущая / следующая команда.
- Control-Enter — вставить имя под курсором в командную строку.
- Control-O: `Suspend`, показать основной экран терминала с выводом прошлых команд, ждать любую клавишу (в сыром режиме чтения stdin), затем `Resume`.
- Меню Commands: «Panels on/off Control-O», «Command history».
- Недоступный каталог — ошибка в красном диалоге, панель остаётся в прежнем каталоге.
- Коммиты — Conventional Commits на английском, **без** `Co-Authored-By` и пометок «Generated with Claude Code».
- Работа в ветке `feature/stage-5` от `main`; в `main` — только через PR.

## Решения этого этапа (уточняют спеку)

- **Esc при непустой строке** очищает её и **не взводит** Esc-префикс (как Esc, закрывший диалог): иначе `Esc` `.` сразу после очистки переключал бы скрытые файлы. Следствие: `Esc 5` при набранной команде сначала очищает строку, а `5` печатается.
- **Вставка имени:** Control-Enter (доходит только в терминалах с kitty / CSI-u, которые tcell включает сам), плюс два способа, работающие везде: **Control-J** и **Alt-Enter** (`Esc` `Enter` при пустой строке). Имя вставляется с пробелом после; имена со спецсимволами — в одинарных кавычках (`'my file.txt' `). На `..` ничего не вставляется.
- **Встроенный `cd`** — только простая форма: `cd` и ровно одно слово, кавычки `'…'`, `"…"` и `\ ` раскрываются. Если в строке есть `$`, `` ` ``, `*`, `?`, `[`, `{`, `;`, `&`, `|`, `<`, `>`, `(`, `)` или несколько слов (`cd a && make`, `cd -P x`, `cd $HOME`) — это обычная команда для шелла (и каталог панели не меняется). `cd` без аргумента — домашний каталог; `cd -` — предыдущий каталог этой панели (`panel.Prev`, меняется при любом переходе, не только через `cd`); если его нет — ничего. Относительные пути и `~` — как в F5/F7 (`resolve`).
- **Строка после Enter** очищается; в историю попадает текст без пробелов по краям, в том числе `cd …`. Запуск программы по Enter на файле в историю не попадает.
- **Выход команды с ненулевым кодом** не показывается отдельно — вывод команды уже на экране. Если шелл не запустился (нет `$SHELL`, каталог исчез), печатается `tc: <ошибка>`.
- **Control-C** во время команды останавливает команду, а не `tc`: пока команда работает, `tc` перехватывает SIGINT и SIGQUIT (`signal.Notify`, а не `signal.Ignore` — игнорирование унаследовала бы команда).
- **«Это программа»** (Enter на файле): обычный файл с битом `x`, который начинается с `#!` или с сигнатуры Mach-O. Одного бита `x` мало: на exFAT/FAT-флешках он стоит у всех файлов, и Enter на фото «запускал» бы его. Программа запускается как команда `./<имя>` (с кавычками при нужде) тем же путём, что и командная строка. Остальное — `open <путь>` без Suspend; ошибка `open` (его stderr) — красный диалог. Команда `open` — переменная `openCmd` (тесты подменяют её, чтобы не открывать приложения).
- **Экран после команды:** «Press any key to continue...»; клавиша читается из stdin в сыром режиме (до 64 байт за раз, чтобы стрелка не «протекла» в `tc` хвостом Esc-последовательности). Если stdin не терминал — ожидания нет.
- **Command history** — окно-список (`ui.List`) в цветах диалога, курсор black / cyan, последняя команда внизу и под курсором. Enter — поставить команду в строку (не выполнять), Esc — закрыть. При пустой истории окно открывается пустым.
- **Длинная строка** — показывается хвост `приглашение+текст`, влезающий в `W-1` колонок; курсор после него.
- История живёт только в памяти; сохранение в конфиг — этап 6.

## Review Focus

1. **Control-C в запущенной команде** (`sleep 100`, `ping`) — останавливает команду, `tc` остаётся жив и показывает «Press any key». Тест: `TestRunSurvivesInterrupt` (Task 3).
2. **Каталоги с пробелами и кавычками в `cd`** (`cd "My Dir"`, `cd My\ Dir`) меняют каталог; `cd a && make` уходит в шелл и каталог панели не трогает. Тесты: `TestParseCd` (Task 2), `TestCd`, `TestCdChainGoesToShell` (Task 5).
3. **Enter на документе с битом `x`** (фото на exFAT) не пытается его исполнить, а открывает через `open`. Тесты: `TestRunnable` (Task 2), `TestEnterOpensDocument` (Task 6).
4. **Space / `+ - *` посреди набора команды** идут в строку, а не выделяют файлы; Esc очищает строку и не превращает следующую клавишу в Alt. Тесты: `TestSpacePlusMinusStarGoToNonEmptyLine`, `TestEscClearsLineWithoutAltPrefix` (Task 4).
5. **Имена с пробелами, кавычками, кириллицей**, вставленные Control-Enter, безопасны для шелла. Тесты: `TestQuote` (Task 2), `TestInsertName` (Task 6).

---

## Карта файлов

```
internal/shell/line.go          Line: Text, Insert, Backspace, Clear, Prev, Next, Commit, History; HistoryLimit
internal/shell/words.go         ParseCd, Quote, Runnable
internal/shell/console.go       Console: Run, Pause, WaitKey
internal/panel/panel.go         + поле Prev (заполняет Load)
internal/ui/list.go             List — окно-список для истории
internal/app/app.go             поля cmd/console; маршрутизация клавиш
internal/app/cmdline.go         openCmd, prompt, execute, cd, runLine, outside, enter, insertName, showHistory, panelsOff
internal/app/draw.go            drawCmdLine: приглашение + текст
internal/app/menu.go            Commands: Panels on/off, Command history
internal/app/help.go            строки справки
README.md, docs/terminal-compat.md
```

---

### Task 1: `shell.Line` — текст строки и история

**Files:**
- Create: `internal/shell/line.go`
- Test: `internal/shell/line_test.go`

**Interfaces:**
- Consumes: —
- Produces: `const shell.HistoryLimit = 500`; `type shell.Line struct { Text string; … }` с методами `Insert(s string)`, `Backspace()`, `Clear()`, `Prev()`, `Next()`, `Commit() string`, `History() []string`. Нулевое значение готово к работе.

- [ ] **Step 1: Write the failing test**

`internal/shell/line_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shell/`
Expected: FAIL — `undefined: Line`, `undefined: HistoryLimit`.

- [ ] **Step 3: Write minimal implementation**

`internal/shell/line.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shell/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/line.go internal/shell/line_test.go
git commit -m "feat(shell): add command line text and history"
```

---

### Task 2: `cd`, кавычки и «это программа»

**Files:**
- Create: `internal/shell/words.go`
- Test: `internal/shell/words_test.go`

**Interfaces:**
- Consumes: —
- Produces: `shell.ParseCd(line string) (dir string, ok bool)` — `"~"` для голого `cd`, иначе аргумент без кавычек; `shell.Quote(s string) string`; `shell.Runnable(path string) bool`.

- [ ] **Step 1: Write the failing test**

`internal/shell/words_test.go`:

```go
package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCd(t *testing.T) {
	tests := []struct {
		line, dir string
		ok        bool
	}{
		{"cd", "~", true},
		{"  cd  ", "~", true},
		{"cd ..", "..", true},
		{"cd -", "-", true},
		{"cd ~/Projects", "~/Projects", true},
		{"cd   /tmp ", "/tmp", true},
		{`cd "My Dir"`, "My Dir", true},
		{`cd 'it''s'`, "its", true},
		{`cd My\ Dir`, "My Dir", true},
		{`cd "a\"b"`, `a"b`, true},
		{"cd Папка", "Папка", true},
		{"cd a b", "", false},
		{"cd a && make", "", false},
		{"cd a;ls", "", false},
		{"cd $HOME", "", false},
		{`cd "$HOME"`, "", false},
		{"cd *.d", "", false},
		{"cd -P /tmp", "", false},
		{`cd "unclosed`, "", false},
		{`cd trailing\`, "", false},
		{"cdx", "", false},
		{"ls", "", false},
	}
	for _, tt := range tests {
		dir, ok := ParseCd(tt.line)
		if dir != tt.dir || ok != tt.ok {
			t.Errorf("ParseCd(%q) = %q, %v; want %q, %v", tt.line, dir, ok, tt.dir, tt.ok)
		}
	}
}

func TestQuote(t *testing.T) {
	tests := []struct{ in, want string }{
		{"file.txt", "file.txt"},
		{"a-b_c+d@e%f:g,h=i/j", "a-b_c+d@e%f:g,h=i/j"},
		{"Отчёт.pdf", "Отчёт.pdf"},
		{"my file.txt", "'my file.txt'"},
		{"it's", `'it'\''s'`},
		{"a&b", "'a&b'"},
		{"~x", "'~x'"},
		{"", "''"},
	}
	for _, tt := range tests {
		if got := Quote(tt.in); got != tt.want {
			t.Errorf("Quote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRunnable(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string, mode os.FileMode) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(data), mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	script := write("script", "#!/bin/sh\necho hi\n", 0o755)
	if err := os.Symlink(script, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable() // the test binary is Mach-O
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path string
		want bool
	}{
		{script, true},
		{filepath.Join(dir, "link"), true},
		{write("short", "#!", 0o755), true},
		{self, true},
		{write("noexec.sh", "#!/bin/sh\n", 0o644), false},
		{write("photo.jpg", "\xff\xd8\xff\xe0JFIF", 0o755), false},
		{write("empty", "", 0o755), false},
		{dir, false},
		{filepath.Join(dir, "missing"), false},
	}
	for _, tt := range tests {
		if got := Runnable(tt.path); got != tt.want {
			t.Errorf("Runnable(%s) = %v, want %v", filepath.Base(tt.path), got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shell/ -run 'ParseCd|Quote|Runnable'`
Expected: FAIL — `undefined: ParseCd`, `undefined: Quote`, `undefined: Runnable`.

- [ ] **Step 3: Write minimal implementation**

`internal/shell/words.go`:

```go
package shell

import (
	"encoding/binary"
	"io"
	"os"
	"strings"
	"unicode"
)

// ParseCd recognizes a plain "cd [dir]" command and returns dir without
// quotes; "cd" alone gives "~". A line the shell would have to expand or
// chain (variables, globs, ";", "&&", several arguments) is not a plain cd.
func ParseCd(line string) (dir string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "cd" {
		return "~", true
	}
	rest, found := strings.CutPrefix(line, "cd ")
	if !found {
		return "", false
	}
	return word(strings.TrimSpace(rest))
}

// special are the characters that make the shell do more than take a word
// as it is.
const special = " \t;&|<>()$`*?[{"

// word unquotes s if it is one shell word with nothing to expand.
func word(s string) (string, bool) {
	var b strings.Builder
	var quote rune
	escaped := false
	for _, r := range s {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				b.WriteRune(r)
			}
		case quote == '"':
			switch r {
			case '"':
				quote = 0
			case '\\':
				escaped = true
			case '$', '`':
				return "", false
			default:
				b.WriteRune(r)
			}
		case r == '\\':
			escaped = true
		case r == '\'' || r == '"':
			quote = r
		case strings.ContainsRune(special, r):
			return "", false
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 || escaped {
		return "", false
	}
	return b.String(), true
}

// Quote makes s safe to paste into a shell command: plain names stay as
// they are, others go in single quotes.
func Quote(s string) string {
	plain := s != ""
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("._-+/@%:,=", r) {
			plain = false
			break
		}
	}
	if plain {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Runnable reports whether path is a program: an executable regular file
// that starts with "#!" or is a Mach-O binary. The x bit alone is not
// enough: on exFAT and FAT drives every file has it.
func Runnable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var b [4]byte
	n, _ := io.ReadFull(f, b[:])
	if n >= 2 && b[0] == '#' && b[1] == '!' {
		return true
	}
	if n < 4 {
		return false
	}
	switch binary.BigEndian.Uint32(b[:]) {
	case 0xfeedface, 0xfeedfacf, 0xcefaedfe, 0xcffaedfe, 0xcafebabe: // Mach-O 32/64 both byte orders, universal
		return true
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shell/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/words.go internal/shell/words_test.go
git commit -m "feat(shell): parse plain cd, quote names, detect programs"
```

---

### Task 3: `shell.Console` — запуск команды и ожидание клавиши

**Files:**
- Create: `internal/shell/console.go`
- Test: `internal/shell/console_test.go`
- Modify: `go.mod` (`golang.org/x/term` становится прямой зависимостью)

**Interfaces:**
- Consumes: —
- Produces: `type shell.Console struct { In *os.File; Out io.Writer }` — `In == nil`: команда без stdin и без ожидания клавиши (так в тестах); методы `Run(dir, line string)`, `Pause()`, `WaitKey()`.

- [ ] **Step 1: Write the failing test**

`internal/shell/console_test.go`:

```go
package shell

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInDirWithShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	dir := t.TempDir()
	var out bytes.Buffer
	Console{Out: &out}.Run(dir, "echo hi; pwd -P; echo oops >&2; exit 3")
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := "hi\n" + real + "\noops\n"; out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
}

func TestRunDefaultsToZsh(t *testing.T) {
	t.Setenv("SHELL", "")
	var out bytes.Buffer
	Console{Out: &out}.Run(t.TempDir(), "echo z$ZSH_VERSION")
	if strings.TrimSpace(out.String()) == "z" {
		t.Fatalf("not run by zsh: %q", out.String())
	}
}

func TestRunReportsStartError(t *testing.T) {
	t.Setenv("SHELL", "/nonexistent/sh")
	var out bytes.Buffer
	Console{Out: &out}.Run(t.TempDir(), "true")
	if !strings.HasPrefix(out.String(), "tc: ") {
		t.Fatalf("output %q", out.String())
	}
}

// If tc did not catch SIGINT while a command runs, Control-C in the
// terminal would kill it together with the command — here, the test binary.
func TestRunSurvivesInterrupt(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	var out bytes.Buffer
	Console{Out: &out}.Run(t.TempDir(), "kill -INT $PPID; sleep 0.2; echo alive")
	if out.String() != "alive\n" {
		t.Fatalf("output %q", out.String())
	}
}

func TestPauseWithoutTerminal(t *testing.T) {
	var out bytes.Buffer
	Console{Out: &out}.Pause() // must not wait
	if out.String() != "Press any key to continue...\n" {
		t.Fatalf("output %q", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shell/ -run 'Run|Pause'`
Expected: FAIL — `undefined: Console`.

- [ ] **Step 3: Write minimal implementation**

`internal/shell/console.go`:

```go
package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// Console is the terminal while the panels are hidden. In is nil when
// there is no terminal to read from: commands get no input and nothing
// waits for a key.
type Console struct {
	In  *os.File
	Out io.Writer
}

// Run runs line with $SHELL -c (/bin/zsh if unset) in dir. Control-C stops
// the command but not tc: tc catches SIGINT and SIGQUIT while it runs.
// A caught signal, unlike an ignored one, is reset for the command.
func (c Console) Run(dir, line string) {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/zsh"
	}
	cmd := exec.Command(sh, "-c", line)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = c.Out, c.Out
	if c.In != nil {
		cmd.Stdin = c.In
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGQUIT)
	defer signal.Stop(sig)
	var exit *exec.ExitError
	if err := cmd.Run(); err != nil && !errors.As(err, &exit) {
		fmt.Fprintln(c.Out, "tc:", err)
	}
}

// Pause asks for a key and waits for it.
func (c Console) Pause() {
	fmt.Fprint(c.Out, "Press any key to continue...")
	c.WaitKey()
	fmt.Fprintln(c.Out)
}

// WaitKey waits for a key in raw mode. It reads up to 64 bytes at once so
// that the rest of an escape sequence (an arrow key) is not left for tc.
func (c Console) WaitKey() {
	if c.In == nil {
		return
	}
	fd := int(c.In.Fd())
	if !term.IsTerminal(fd) {
		return
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, old)
	var buf [64]byte
	c.In.Read(buf[:])
}
```

- [ ] **Step 4: Make `golang.org/x/term` a direct dependency**

Run: `go mod tidy && git diff go.mod`
Expected: `golang.org/x/term v0.37.0` переехал из блока `// indirect` в основной `require` (версия не меняется).

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/shell/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/shell/console.go internal/shell/console_test.go go.mod go.sum
git commit -m "feat(shell): run commands in the terminal and wait for a key"
```

---

### Task 4: набор команды под панелями

**Files:**
- Modify: `internal/app/app.go` (поле `cmd`, клавиши Backspace / Esc, `handleRune`)
- Modify: `internal/app/draw.go` (`prompt`, `drawCmdLine`)
- Test: `internal/app/cmdline_test.go` (новый)

**Interfaces:**
- Consumes: `shell.Line` (Task 1).
- Produces: поле `App.cmd shell.Line`; `func (a *App) prompt() string` — `<путь активной панели>>`; тестовый помощник `typeText(a *App, s string)`.

- [ ] **Step 1: Write the failing test**

`internal/app/cmdline_test.go`:

```go
package app

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

func typeText(a *App, s string) {
	for _, r := range s {
		press(a, tcell.KeyRune, r, 0)
	}
}

func TestTypingGoesToCommandLine(t *testing.T) {
	a, dir := newApp(t)
	typeText(a, "ls -la")
	press(a, tcell.KeyBackspace, 0, 0)
	if a.cmd.Text != "ls -l" || a.panels[0].Path != dir {
		t.Fatalf("text %q path %s", a.cmd.Text, a.panels[0].Path)
	}
}

func TestSpacePlusMinusStarGoToNonEmptyLine(t *testing.T) {
	a, _ := newApp(t)
	typeText(a, "x +-* ")
	if a.cmd.Text != "x +-* " || len(a.panels[0].Selected) != 0 || !a.modals.Empty() {
		t.Fatalf("text %q selected %v modals %d", a.cmd.Text, a.panels[0].Selected, a.modals.Len())
	}
}

func TestEscClearsLineWithoutAltPrefix(t *testing.T) {
	a, _ := newApp(t)
	typeText(a, "ls")
	press(a, tcell.KeyEscape, 0, 0)
	if a.cmd.Text != "" {
		t.Fatalf("text %q after Esc", a.cmd.Text)
	}
	press(a, tcell.KeyRune, '.', 0)
	if a.cmd.Text != "." || a.showHidden {
		t.Fatalf("text %q hidden %v", a.cmd.Text, a.showHidden)
	}
}

func TestArrowsMovePanelWhileTyping(t *testing.T) {
	a, _ := newApp(t)
	typeText(a, "ls")
	before := a.panels[0].Cursor
	press(a, tcell.KeyDown, 0, 0)
	if a.panels[0].Cursor == before || a.cmd.Text != "ls" {
		t.Fatalf("cursor %d text %q", a.panels[0].Cursor, a.cmd.Text)
	}
}

// cmdRow returns the command line row (y = 23 on the 80×25 test screen).
func cmdRow(a *App) string {
	return strings.Split(termtest.Dump(a.screen.(tcell.SimulationScreen)), "\n")[23]
}

func TestCommandLineShowsText(t *testing.T) {
	a, dir := newApp(t)
	a.home = dir
	typeText(a, "ls")
	if row := cmdRow(a); !strings.HasPrefix(row, "~>ls ") {
		t.Fatalf("row %q", row)
	}
	if x, y, _ := a.screen.(tcell.SimulationScreen).GetCursor(); x != 4 || y != 23 {
		t.Fatalf("cursor at %d,%d", x, y)
	}
}

func TestLongCommandLineShowsTail(t *testing.T) {
	a, dir := newApp(t)
	a.home = dir
	long := strings.Repeat("abcdefghij", 10)
	typeText(a, long)
	if row := cmdRow(a); row != long[len(long)-79:]+" " {
		t.Fatalf("row %q", row)
	}
	if x, _, _ := a.screen.(tcell.SimulationScreen).GetCursor(); x != 79 {
		t.Fatalf("cursor x %d", x)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run 'Typing|NonEmptyLine|EscClears|WhileTyping|CommandLine'`
Expected: FAIL — `a.cmd undefined`.

- [ ] **Step 3: Implement**

В `internal/app/app.go`:

1. Импорт `"github.com/navoznov/terminal-commander/internal/shell"`.
2. Поле в `App` после `op`:

```go
	cmd        shell.Line  // the command line
```

3. В `handleKey` заменить `case tcell.KeyBackspace:` и добавить `case tcell.KeyEscape:` сразу после него:

```go
	case tcell.KeyBackspace:
		if a.cmd.Text != "" {
			a.cmd.Backspace()
		} else {
			a.report(p.Up())
		}
	case tcell.KeyEscape:
		if a.cmd.Text != "" {
			a.cmd.Clear()
			a.keys.Disarm() // the Esc was used; don't make the next key Alt-key
		}
```

4. `handleRune` целиком:

```go
func (a *App) handleRune(r rune, mod tcell.ModMask) {
	p := a.panels[a.active]
	empty := a.cmd.Text == ""
	switch {
	case (r == '.' || r == 'ю') && mod&tcell.ModAlt != 0: // 'ю' is the '.' key on the Russian layout
		a.toggleHidden()
	case r == '1' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Brief)
	case r == '2' && mod&tcell.ModCtrl != 0:
		p.SetMode(panel.Full)
	case mod&(tcell.ModAlt|tcell.ModCtrl) != 0:
		// other Alt and Control keys do nothing
	case r == ' ' && empty:
		p.ToggleSelect()
	case r == '+' && empty:
		a.askMask(true)
	case r == '-' && empty:
		a.askMask(false)
	case r == '*' && empty:
		p.InvertSelection()
	default:
		a.cmd.Insert(string(r))
	}
}
```

В `internal/app/draw.go` заменить `drawCmdLine` и добавить `prompt`:

```go
// prompt is the command line prompt: the active panel's path and ">".
func (a *App) prompt() string {
	return fs.DisplayPath(a.panels[a.active].Path, a.home) + ">"
}

// drawCmdLine shows the prompt and the command; when they are too long,
// their end is shown, with the cursor after it.
func (a *App) drawCmdLine(c term.Canvas, y, w int) {
	c.HLine(0, y, w, ' ', term.CmdLineStyle)
	n := c.Text(0, y, term.Tail(a.prompt()+a.cmd.Text, w-1), w, term.CmdLineStyle)
	a.screen.ShowCursor(n, y)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/app/`
Expected: PASS, включая старые `TestSpaceSelects`, `TestEnterAndBackspace` и golden-экраны (пустая строка рисуется как раньше).

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/draw.go internal/app/cmdline_test.go
git commit -m "feat(app): type commands into the command line"
```

---

### Task 5: Enter выполняет команду; `cd`; история Control-E / Control-X

**Files:**
- Modify: `internal/panel/panel.go` (поле `Prev`, `Load`)
- Test: `internal/panel/panel_test.go`
- Create: `internal/app/cmdline.go`
- Modify: `internal/app/app.go` (поле `console`, `New`, клавиши Enter / Control-E / Control-X)
- Test: `internal/app/cmdline_test.go`

**Interfaces:**
- Consumes: `shell.Line.Commit/Prev/Next/History` (Task 1), `shell.ParseCd` (Task 2), `shell.Console{In, Out}.Run/Pause` (Task 3), `App.prompt()` (Task 4), `resolve(dir, home, s string) string` (`internal/app/files.go`), `a.reloadPanels()`.
- Produces: `panel.Panel.Prev string` — каталог до последнего перехода; поле `App.console shell.Console`; `func (a *App) execute()`, `func (a *App) cd(p *panel.Panel, dir string)`, `func (a *App) runLine(p *panel.Panel, line string)`, `func (a *App) outside(f func())`; тестовые помощники `quietConsole(t, a) *bytes.Buffer`, `run(a, line)`.

- [ ] **Step 1: Write the failing panel test**

Добавить в `internal/panel/panel_test.go` (в файле уже есть нужные импорты `os`, `path/filepath`, `testing`; если какого-то нет — добавить):

```go
func TestLoadRemembersPrev(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := New()
	for _, path := range []string{dir, sub, sub} {
		if err := p.Load(path); err != nil {
			t.Fatal(err)
		}
	}
	if p.Prev != dir {
		t.Fatalf("prev %q, want %q", p.Prev, dir)
	}
	if p.Load(filepath.Join(dir, "missing")) == nil || p.Prev != dir {
		t.Fatalf("failed Load changed prev to %q", p.Prev)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/panel/ -run TestLoadRemembersPrev`
Expected: FAIL — `p.Prev undefined`.

- [ ] **Step 3: Implement `Prev`**

В `internal/panel/panel.go` в структуру `Panel` рядом с `Path` добавить поле:

```go
	Prev string // the directory before the last change, for "cd -"
```

В `Load` после успешного `p.read(path)` и перед присваиванием `p.Path`:

```go
	if path != p.Path {
		p.Prev = p.Path
	}
```

Run: `go test ./internal/panel/`
Expected: PASS.

- [ ] **Step 4: Write the failing app tests**

Добавить в `internal/app/cmdline_test.go` (импорты: `bytes`, `os`, `path/filepath`, `slices`, `strings`, `testing`, `tcell`, `internal/shell`, `internal/term/termtest`, `internal/ui`):

```go
// quietConsole makes commands run with /bin/sh and write into the returned
// buffer instead of the terminal, without waiting for a key.
func quietConsole(t *testing.T, a *App) *bytes.Buffer {
	t.Setenv("SHELL", "/bin/sh")
	out := &bytes.Buffer{}
	a.console = shell.Console{Out: out}
	return out
}

// run types line and presses Enter.
func run(a *App, line string) {
	typeText(a, line)
	press(a, tcell.KeyEnter, 0, 0)
}

func TestEnterRunsCommand(t *testing.T) {
	a, dir := newApp(t)
	a.home = dir
	out := quietConsole(t, a)
	run(a, "touch made.txt")
	if _, err := os.Stat(filepath.Join(dir, "made.txt")); err != nil {
		t.Fatal(err)
	}
	if !a.panels[0].Focus("made.txt") || !a.panels[1].Focus("made.txt") {
		t.Fatal("panels not re-read")
	}
	if want := "~>touch made.txt\nPress any key to continue...\n"; out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
	if a.cmd.Text != "" || !slices.Equal(a.cmd.History(), []string{"touch made.txt"}) {
		t.Fatalf("text %q history %q", a.cmd.Text, a.cmd.History())
	}
}

func TestCommandRunsInActivePanelDir(t *testing.T) {
	a, dir := newApp(t)
	out := quietConsole(t, a)
	must(t, a.panels[1].Load(filepath.Join(dir, "sub")))
	press(a, tcell.KeyTab, 0, 0)
	run(a, "pwd -P")
	real, err := filepath.EvalSymlinks(filepath.Join(dir, "sub"))
	must(t, err)
	if !strings.Contains(out.String(), "\n"+real+"\n") {
		t.Fatalf("output %q", out.String())
	}
}

func TestBlankLineEntersDirectory(t *testing.T) {
	a, dir := newApp(t)
	typeText(a, "x")
	press(a, tcell.KeyBackspace, 0, 0)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyEnter, 0, 0)
	if a.panels[0].Path != filepath.Join(dir, "sub") {
		t.Fatalf("path %s", a.panels[0].Path)
	}
}

func TestCd(t *testing.T) {
	a, dir := newApp(t)
	out := quietConsole(t, a)
	a.home = filepath.Join(dir, "sub")
	must(t, os.Mkdir(filepath.Join(dir, "My Dir"), 0o755))
	p := a.panels[0]
	steps := []struct{ line, want string }{
		{"cd sub", "sub"},
		{"cd ..", ""},
		{`cd "My Dir"`, "My Dir"},
		{"cd -", ""},
		{"cd -", "My Dir"},
		{"cd", "sub"},
		{"cd ~/", "sub"},
		{"cd " + dir, ""},
	}
	for _, st := range steps {
		run(a, st.line)
		if want := filepath.Join(dir, st.want); p.Path != want {
			t.Fatalf("%q: path %s, want %s", st.line, p.Path, want)
		}
	}
	if out.Len() != 0 {
		t.Fatalf("cd reached the shell: %q", out.String())
	}
	if h := a.cmd.History(); h[0] != "cd sub" {
		t.Fatalf("history %q", h)
	}
}

func TestCdErrorKeepsPanel(t *testing.T) {
	a, dir := newApp(t)
	quietConsole(t, a)
	run(a, "cd nope")
	d, ok := a.modals.Top().(*ui.Dialog)
	if a.panels[0].Path != dir || !ok || !d.Danger {
		t.Fatalf("path %s top %#v", a.panels[0].Path, a.modals.Top())
	}
}

func TestCdChainGoesToShell(t *testing.T) {
	a, dir := newApp(t)
	out := quietConsole(t, a)
	run(a, "cd sub && pwd -P")
	if a.panels[0].Path != dir || !strings.Contains(out.String(), "/sub\n") {
		t.Fatalf("path %s output %q", a.panels[0].Path, out.String())
	}
}

func TestHistoryKeys(t *testing.T) {
	a, _ := newApp(t)
	quietConsole(t, a)
	run(a, "cd sub")
	run(a, "cd ..")
	steps := []struct {
		k    tcell.Key
		want string
	}{
		{tcell.KeyCtrlE, "cd .."},
		{tcell.KeyCtrlE, "cd sub"},
		{tcell.KeyCtrlX, "cd .."},
		{tcell.KeyCtrlX, ""},
	}
	for i, st := range steps {
		press(a, st.k, 0, tcell.ModCtrl)
		if a.cmd.Text != st.want {
			t.Fatalf("step %d: text %q, want %q", i, a.cmd.Text, st.want)
		}
	}
}
```

- [ ] **Step 5: Run them to verify they fail**

Run: `go test ./internal/app/ -run 'EnterRuns|ActivePanelDir|BlankLine|Cd|HistoryKeys'`
Expected: FAIL — `a.console undefined`.

- [ ] **Step 6: Implement**

`internal/app/cmdline.go`:

```go
package app

import (
	"fmt"

	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/shell"
)

// execute runs the command line: a plain cd changes the active panel's
// directory, anything else goes to the shell.
func (a *App) execute() {
	line := a.cmd.Commit()
	p := a.panels[a.active]
	if dir, ok := shell.ParseCd(line); ok {
		a.cd(p, dir)
		return
	}
	a.runLine(p, line)
}

// cd changes p's directory the way the shell's cd would; "-" goes back to
// the previous one.
func (a *App) cd(p *panel.Panel, dir string) {
	if dir == "-" {
		if p.Prev == "" {
			return
		}
		dir = p.Prev
	}
	a.report(p.Load(resolve(p.Path, a.home, dir)))
}

// runLine shows the prompt and line on the terminal's own screen, runs line
// in p's directory, waits for a key and re-reads the panels.
func (a *App) runLine(p *panel.Panel, line string) {
	prompt := a.prompt()
	a.outside(func() {
		fmt.Fprintln(a.console.Out, prompt+line)
		a.console.Run(p.Path, line)
		a.console.Pause()
	})
	a.reloadPanels()
}

// outside hides the panels and gives the terminal to f.
func (a *App) outside(f func()) {
	if err := a.screen.Suspend(); err != nil {
		a.report(err)
		return
	}
	f()
	a.report(a.screen.Resume())
	a.screen.Sync()
}
```

В `internal/app/app.go`:

1. Импорт `"strings"`.
2. Поле в `App` после `cmd`:

```go
	console    shell.Console // where commands run
```

3. В `New` — инициализация в литерале:

```go
	a := &App{screen: s, home: home, calls: make(chan func(), 16), console: shell.Console{In: os.Stdin, Out: os.Stdout}}
```

4. В `handleKey` заменить `case tcell.KeyEnter:` и добавить Control-E / Control-X рядом с `KeyCtrlR`:

```go
	case tcell.KeyEnter:
		if strings.TrimSpace(a.cmd.Text) != "" {
			a.execute()
		} else {
			a.cmd.Clear()
			_, err := p.Enter()
			a.report(err)
		}
```

```go
	case tcell.KeyCtrlE:
		a.cmd.Prev()
	case tcell.KeyCtrlX:
		a.cmd.Next()
```

- [ ] **Step 7: Run tests**

Run: `go test ./internal/app/ ./internal/panel/`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/panel/panel.go internal/panel/panel_test.go internal/app/app.go internal/app/cmdline.go internal/app/cmdline_test.go
git commit -m "feat(app): run the command line; built-in cd and history keys"
```

---

### Task 6: Enter на файле; вставка имени в строку

**Files:**
- Modify: `internal/app/cmdline.go` (`openCmd`, `enter`, `insertName`)
- Modify: `internal/app/app.go` (Enter, Control-J)
- Create: `internal/app/main_test.go`
- Test: `internal/app/cmdline_test.go`

**Interfaces:**
- Consumes: `shell.Runnable`, `shell.Quote` (Task 2), `a.runLine` (Task 5), `panel.Panel.Enter() (bool, error)`, `panel.Panel.Current() *fs.Entry`.
- Produces: `var openCmd = "open"`; `func (a *App) enter()`, `func (a *App) insertName()`.

- [ ] **Step 1: Stub `open` for the whole package's tests**

`internal/app/main_test.go`:

```go
package app

import (
	"os"
	"testing"
)

// TestMain keeps tests from opening documents in real apps.
func TestMain(m *testing.M) {
	openCmd = "/usr/bin/true"
	os.Exit(m.Run())
}
```

- [ ] **Step 2: Write the failing tests**

Добавить в `internal/app/cmdline_test.go`:

```go
// fakeOpen replaces the open command with a script running body.
func fakeOpen(t *testing.T, body string) {
	path := filepath.Join(t.TempDir(), "open")
	must(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	openCmd = path
	t.Cleanup(func() { openCmd = "/usr/bin/true" })
}

// addFile creates a file in the left panel's directory and puts the
// cursor on it.
func addFile(t *testing.T, a *App, name, data string, mode os.FileMode) string {
	p := a.panels[0]
	path := filepath.Join(p.Path, name)
	must(t, os.WriteFile(path, []byte(data), mode))
	must(t, p.Reload())
	p.Focus(name)
	return path
}

func TestEnterRunsProgram(t *testing.T) {
	a, dir := newApp(t)
	a.home = dir
	out := quietConsole(t, a)
	addFile(t, a, "go.sh", "#!/bin/sh\ntouch ran\n", 0o755)
	press(a, tcell.KeyEnter, 0, 0)
	if _, err := os.Stat(filepath.Join(dir, "ran")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "~>./go.sh\n") || len(a.cmd.History()) != 0 {
		t.Fatalf("output %q history %q", out.String(), a.cmd.History())
	}
}

func TestEnterOpensDocument(t *testing.T) {
	a, dir := newApp(t)
	log := filepath.Join(dir, "opened")
	fakeOpen(t, `echo "$1" > '`+log+`'`)
	photo := addFile(t, a, "photo.jpg", "\xff\xd8\xff\xe0JFIF", 0o755) // x bit, as on exFAT
	press(a, tcell.KeyEnter, 0, 0)
	got, err := os.ReadFile(log)
	must(t, err)
	if string(got) != photo+"\n" || !a.modals.Empty() {
		t.Fatalf("opened %q modals %d", got, a.modals.Len())
	}
}

func TestOpenErrorShown(t *testing.T) {
	a, _ := newApp(t)
	fakeOpen(t, `echo "No application knows how to open $1" >&2; exit 1`)
	addFile(t, a, "doc.xyz", "", 0o644)
	press(a, tcell.KeyEnter, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || !d.Danger || !strings.Contains(strings.Join(d.Lines, " "), "No application knows") {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestInsertName(t *testing.T) {
	a, _ := newApp(t)
	p := a.panels[0]
	addFile(t, a, "my file.txt", "", 0o644)
	typeText(a, "cat ")
	press(a, tcell.KeyEnter, 0, tcell.ModCtrl)
	p.Focus("sub")
	press(a, tcell.KeyCtrlJ, 0, tcell.ModCtrl)
	p.Focus("..")
	press(a, tcell.KeyEnter, 0, tcell.ModAlt)
	if want := "cat 'my file.txt' sub "; a.cmd.Text != want {
		t.Fatalf("text %q, want %q", a.cmd.Text, want)
	}
}

func TestEscEnterInsertsName(t *testing.T) {
	a, dir := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyEscape, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	if a.cmd.Text != "sub " || a.panels[0].Path != dir {
		t.Fatalf("text %q path %s", a.cmd.Text, a.panels[0].Path)
	}
}
```

- [ ] **Step 3: Run them to verify they fail**

Run: `go test ./internal/app/ -run 'EnterRunsProgram|OpensDocument|OpenError|InsertName|EscEnter'`
Expected: FAIL — `undefined: openCmd` (компиляция), после заглушки — несовпадения.

- [ ] **Step 4: Implement**

Добавить в `internal/app/cmdline.go` (импорты: `errors`, `os/exec`, `path/filepath`, `strings`):

```go
// openCmd opens a document with its app.
var openCmd = "open"

// enter acts on the entry under the cursor: a directory is entered, a
// program is run, anything else is opened with its app.
func (a *App) enter() {
	p := a.panels[a.active]
	if ok, err := p.Enter(); ok || err != nil {
		a.report(err)
		return
	}
	e := p.Current()
	if e == nil {
		return
	}
	path := filepath.Join(p.Path, e.Name)
	if shell.Runnable(path) {
		a.runLine(p, "./"+shell.Quote(e.Name))
		return
	}
	if out, err := exec.Command(openCmd, path).CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			err = errors.New(msg)
		}
		a.report(err)
	}
}

// insertName appends the name under the cursor to the command line.
func (a *App) insertName() {
	e := a.panels[a.active].Current()
	if e == nil || e.IsUp {
		return
	}
	a.cmd.Insert(shell.Quote(e.Name) + " ")
}
```

В `internal/app/app.go` заменить `case tcell.KeyEnter:` (версию из Task 5) и добавить Control-J:

```go
	case tcell.KeyEnter:
		switch {
		case ev.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0:
			a.insertName()
		case strings.TrimSpace(a.cmd.Text) != "":
			a.execute()
		default:
			a.cmd.Clear()
			a.enter()
		}
	case tcell.KeyCtrlJ: // Control-Enter in terminals that send it as LF
		a.insertName()
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/app/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go internal/app/cmdline.go internal/app/main_test.go internal/app/cmdline_test.go
git commit -m "feat(app): Enter runs programs and opens documents; insert names"
```

---

### Task 7: Command history, Control-O и пункты меню

**Files:**
- Create: `internal/ui/list.go`
- Test: `internal/ui/list_test.go`
- Modify: `internal/app/cmdline.go` (`showHistory`, `panelsOff`)
- Modify: `internal/app/app.go` (Control-O)
- Modify: `internal/app/menu.go` (Commands)
- Test: `internal/app/cmdline_test.go`

**Interfaces:**
- Consumes: `ui.View`, `window`, `drawTitle` (`internal/ui/window.go`), `term.DialogStyle`, `term.DialogFrameStyle`, `term.CursorStyle`, `term.Fit`; `shell.Console.WaitKey` (Task 3), `a.outside` (Task 5).
- Produces: `type ui.List struct { Title string; Items []string; Cur, Top int; Done func(i int) }` — `Done(-1)` на Esc и на Enter в пустом списке; `func (a *App) showHistory()`, `func (a *App) panelsOff()`.

- [ ] **Step 1: Write the failing ui tests**

`internal/ui/list_test.go`:

```go
package ui_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestListPick(t *testing.T) {
	got := -2
	l := &ui.List{Title: "History", Items: []string{"a", "b", "c"}, Cur: 2, Done: func(i int) { got = i }}
	l.Draw(term.Canvas{Screen: background(t, 60, 12)}, 60, 12)
	for _, k := range []tcell.Key{tcell.KeyUp, tcell.KeyUp, tcell.KeyUp, tcell.KeyDown} {
		if l.HandleKey(key(k)) {
			t.Fatal("list closed on a move")
		}
	}
	if !l.HandleKey(key(tcell.KeyEnter)) || got != 1 {
		t.Fatalf("got %d", got)
	}
}

func TestListEscAndEmpty(t *testing.T) {
	got := -2
	l := &ui.List{Items: []string{"a"}, Done: func(i int) { got = i }}
	if !l.HandleKey(key(tcell.KeyEscape)) || got != -1 {
		t.Fatalf("Esc gave %d", got)
	}
	got = -2
	empty := &ui.List{Title: "History", Cur: -1, Done: func(i int) { got = i }}
	empty.Draw(term.Canvas{Screen: background(t, 60, 12)}, 60, 12) // must not panic
	empty.HandleKey(key(tcell.KeyDown))
	if !empty.HandleKey(key(tcell.KeyEnter)) || got != -1 {
		t.Fatalf("Enter on empty list gave %d", got)
	}
}

func TestListScrollsToCursor(t *testing.T) {
	var items []string
	for i := 0; i < 30; i++ {
		items = append(items, fmt.Sprintf("item %d", i))
	}
	s := background(t, 60, 12) // 12-6 = 6 visible rows
	l := &ui.List{Title: "History", Items: items, Cur: 29}
	l.Draw(term.Canvas{Screen: s}, 60, 12)
	if l.Top != 24 {
		t.Fatalf("top %d", l.Top)
	}
	lines := strings.Split(termtest.Dump(s), "\n")
	for y, line := range lines[:12] {
		if i := strings.Index(line, "item 29"); i >= 0 {
			x := utf8.RuneCountInString(line[:i])
			if _, _, st, _ := s.GetContent(x, y); st != term.CursorStyle {
				t.Fatal("cursor row not highlighted")
			}
		}
	}
	l.HandleKey(key(tcell.KeyHome))
	l.Draw(term.Canvas{Screen: s}, 60, 12)
	if l.Cur != 0 || l.Top != 0 {
		t.Fatalf("cur %d top %d", l.Cur, l.Top)
	}
}
```

Помощники `key` (`stack_test.go`) и `background` (`dialog_test.go`) уже есть в пакете `ui_test`.

- [ ] **Step 2: Run them to verify they fail**

Run: `go test ./internal/ui/ -run List`
Expected: FAIL — `undefined: ui.List`.

- [ ] **Step 3: Implement `ui.List`**

`internal/ui/list.go`:

```go
package ui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

// List is a window with lines to pick one from, used for the command
// history. It looks like a dialog; the cursor line is black on cyan.
type List struct {
	Title string
	Items []string
	Cur   int // line under the cursor
	Top   int // first visible line
	rows  int // visible lines, set by Draw
	// Done is called with the picked line, or -1 on Esc or in an empty list.
	Done func(i int)
}

func (l *List) move(d int) {
	l.Cur = max(min(l.Cur+d, len(l.Items)-1), 0)
}

func (l *List) HandleKey(ev *tcell.EventKey) bool {
	page := max(l.rows, 1)
	switch ev.Key() {
	case tcell.KeyEscape:
		l.finish(-1)
		return true
	case tcell.KeyEnter:
		if len(l.Items) == 0 {
			l.finish(-1)
		} else {
			l.finish(l.Cur)
		}
		return true
	case tcell.KeyUp:
		l.move(-1)
	case tcell.KeyDown:
		l.move(1)
	case tcell.KeyPgUp:
		l.move(-page)
	case tcell.KeyPgDn:
		l.move(page)
	case tcell.KeyHome:
		l.move(-len(l.Items))
	case tcell.KeyEnd:
		l.move(len(l.Items))
	}
	return false
}

func (l *List) finish(i int) {
	if l.Done != nil {
		l.Done(i)
	}
}

// Draw uses the dialog layout, like TextView, and scrolls to the cursor.
func (l *List) Draw(c term.Canvas, w, h int) {
	cw := max(term.Width(l.Title)+2, 20)
	for _, s := range l.Items {
		cw = max(cw, term.Width(s))
	}
	cw = min(cw, w-10)
	l.rows = max(min(len(l.Items), h-6), 1)
	l.move(0)
	if l.Cur < l.Top {
		l.Top = l.Cur
	}
	if l.Cur >= l.Top+l.rows {
		l.Top = l.Cur - l.rows + 1
	}
	ww, wh := cw+8, l.rows+4
	x, y := max((w-ww)/2, 0), max((h-wh)/2, 0)
	window(c, x, y, ww, wh, term.DialogStyle)
	c.Box(x+2, y+1, ww-4, wh-2, term.DialogFrameStyle)
	drawTitle(c, x, y+1, ww, l.Title, term.DialogFrameStyle)
	for i := 0; i < l.rows && l.Top+i < len(l.Items); i++ {
		st := term.DialogStyle
		if l.Top+i == l.Cur {
			st = term.CursorStyle
		}
		c.Text(x+4, y+2+i, term.Fit(l.Items[l.Top+i], cw), cw, st)
	}
}
```

Run: `go test ./internal/ui/`
Expected: PASS.

- [ ] **Step 4: Write the failing app tests**

Добавить в `internal/app/cmdline_test.go`:

```go
func TestCommandHistoryWindow(t *testing.T) {
	a, _ := newApp(t)
	quietConsole(t, a)
	run(a, "cd sub")
	run(a, "cd ..")
	a.showHistory()
	l, ok := a.modals.Top().(*ui.List)
	if !ok || !slices.Equal(l.Items, []string{"cd sub", "cd .."}) || l.Cur != 1 {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyUp, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	if a.cmd.Text != "cd sub" || !a.modals.Empty() {
		t.Fatalf("text %q modals %d", a.cmd.Text, a.modals.Len())
	}
}

func TestCtrlOReturnsToPanels(t *testing.T) {
	a, _ := newApp(t)
	quietConsole(t, a) // no terminal: WaitKey returns at once
	press(a, tcell.KeyCtrlO, 0, tcell.ModCtrl)
	if !a.modals.Empty() || !strings.Contains(termtest.Dump(a.screen.(tcell.SimulationScreen)), "Name") {
		t.Fatal("panels not back after Control-O")
	}
}
```

Run: `go test ./internal/app/ -run 'CommandHistoryWindow|CtrlO'`
Expected: FAIL — `a.showHistory undefined`.

- [ ] **Step 5: Implement**

Добавить в `internal/app/cmdline.go` (импорт `internal/ui`):

```go
// showHistory lists the commands run; the picked one goes to the command
// line.
func (a *App) showHistory() {
	h := a.cmd.History()
	a.modals.Push(&ui.List{
		Title: "History",
		Items: h,
		Cur:   len(h) - 1,
		Done: func(i int) {
			if i >= 0 {
				a.cmd.Text = h[i]
			}
		},
	})
}

// panelsOff shows the terminal's own screen, with the output of earlier
// commands, until a key is pressed.
func (a *App) panelsOff() {
	a.outside(a.console.WaitKey)
}
```

В `internal/app/app.go` в `handleKey` рядом с `KeyCtrlR`:

```go
	case tcell.KeyCtrlO:
		a.panelsOff()
```

В `internal/app/menu.go` меню Commands:

```go
		{Title: "Commands", Items: []ui.Item{
			{Label: "Swap panels", Key: "Control-U", Action: a.swapPanels},
			{Label: "Panels on/off", Key: "Control-O", Action: a.panelsOff},
			{Label: "Command history", Action: a.showHistory},
		}},
```

- [ ] **Step 6: Run tests**

Run: `go test ./...`
Expected: PASS (golden меню не меняется — подписи пунктов те же).

- [ ] **Step 7: Commit**

```bash
git add internal/ui/list.go internal/ui/list_test.go internal/app/app.go internal/app/cmdline.go internal/app/menu.go internal/app/cmdline_test.go
git commit -m "feat(app): command history window and Control-O"
```

---

### Task 8: справка, README, совместимость терминалов, ручная проверка

**Files:**
- Modify: `internal/app/help.go`
- Modify: `README.md`
- Modify: `docs/terminal-compat.md`

**Interfaces:**
- Consumes: всё из Task 1–7.
- Produces: —

- [ ] **Step 1: Help lines**

В `internal/app/help.go` заменить строку `Control-Enter …` и добавить строки о командной строке после `Control-E / X …`:

```go
	"Control-Enter       Put name into command line",
	"  (also Control-J, Esc Enter)",
	"Control-E / X       Previous / next command",
	"Esc                 Clear command line",
```

И строку про Enter:

```go
	"Enter               Run command line, or enter directory,",
	"                    run program, open file",
```

- [ ] **Step 2: README**

1. Блок NOTE: «Stages 1–5 of 6 are done: …, file operations (…) and the command line. The viewer and the editor aren't built yet.»
2. В «Features» после пункта про Esc prefix:

```markdown
- A **command line** under the panels: type a command and press **Enter** to run it with your `$SHELL` in the active panel's directory. `cd` changes the panel's directory. **Control-E / Control-X** walk the history, **Control-Enter** (or **Control-J**, or **Esc Enter**) puts the file name under the cursor into the line, and **Control-O** shows the terminal with the output of earlier commands.
- **Enter** on a program runs it; on any other file it opens the file with its app, as `open` does.
```

3. В таблице «Keys»: строку `Enter` заменить на «Run the command line if it isn't empty; otherwise go into the directory, run the program or open the file», а перед `F1` добавить:

```markdown
| Control-Enter, Control-J, Esc Enter | Put the name under the cursor into the command line |
| Control-E / Control-X | Previous / next command from the history |
| Control-O | Hide the panels and show the terminal until a key is pressed |
| Esc | Clear the command line |
```

4. «Getting started»: после шага 5 добавить шаг «Type a command, for example `ls -l`, and press **Enter**. Press any key to come back to the panels; **Control-O** shows that output again.» (нумерацию сдвинуть).
5. «Project layout»: добавить строку `internal/shell/    command line: history, built-in cd, running commands`.
6. «Roadmap»: этап 5 — ✅.

- [ ] **Step 3: `docs/terminal-compat.md`**

Добавить в конец раздела «Выводы»:

```markdown
- Control-Enter доходит отдельно от Enter только в терминалах с протоколом
  kitty / CSI-u (tcell включает его сам, если терминал умеет); вставить имя
  в командную строку везде можно через Control-J или `Esc` `Enter`.
```

- [ ] **Step 4: Full test run and build**

Run: `go vet ./... && go test -race ./... && make build`
Expected: всё зелёное, собран `./tc`.

- [ ] **Step 5: Manual check in tmux**

```bash
tmux kill-session -t tc5 2>/dev/null
tmux new-session -d -s tc5 -x 100 -y 30 "./tc $PWD"
sleep 0.5
tmux send-keys -t tc5 'echo hello' Enter; sleep 0.5; tmux capture-pane -p -t tc5 | tail -5   # "…>echo hello", "hello", "Press any key to continue..."
tmux send-keys -t tc5 x; sleep 0.3; tmux capture-pane -p -t tc5 | tail -2                 # панели вернулись, строка пустая
tmux send-keys -t tc5 C-o; sleep 0.3; tmux capture-pane -p -t tc5 | tail -4               # снова виден вывод "hello"
tmux send-keys -t tc5 x; sleep 0.3
tmux send-keys -t tc5 'sleep 30' Enter; sleep 0.5; tmux send-keys -t tc5 C-c; sleep 0.5
tmux capture-pane -p -t tc5 | tail -2                                                      # "Press any key…": tc жив
tmux send-keys -t tc5 x; sleep 0.3
tmux send-keys -t tc5 'cd /tmp' Enter; sleep 0.3; tmux capture-pane -p -t tc5 | tail -2    # приглашение "/tmp>"
tmux send-keys -t tc5 C-e; sleep 0.3; tmux capture-pane -p -t tc5 | tail -2                # в строке "cd /tmp"
tmux send-keys -t tc5 Escape; sleep 1.2
tmux send-keys -t tc5 'vim -u NONE' Enter; sleep 0.5; tmux send-keys -t tc5 ':q' Enter; sleep 0.5; tmux send-keys -t tc5 x; sleep 0.3
tmux capture-pane -p -t tc5 | head -3                                                      # панели на месте, экран не испорчен
tmux send-keys -t tc5 F10 Enter
tmux kill-session -t tc5 2>/dev/null
```

Ожидаемое — в комментариях. Любое расхождение — остановиться и разобраться (superpowers:systematic-debugging), а не править наугад.

- [ ] **Step 6: Commit**

```bash
git add internal/app/help.go README.md docs/terminal-compat.md
git commit -m "docs: command line keys in help, README and terminal notes"
```
