# Terminal Commander — этап 4: операции с файлами. План реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** F5 копирование, F6 переименование/перемещение, F7 создание
папки, F8 удаление в Корзину, Shift-F8 удаление навсегда — с окном
прогресса, отменой, вопросами о конфликтах и ошибках (Retry / Skip / Cancel).

**Architecture:** Новый пакет `internal/ops` без `tcell`: функции
`Copy`, `Move`, `Remove`, `Trash` выполняются на горутине вызывающего и
общаются с пользователем через колбэки `ops.Job` (прогресс, конфликт,
непустая папка, ошибка); отмена — атомарный флаг, проверяется между
файлами и между блоками по 1 МиБ. `internal/app` запускает операцию
в горутине; всё, что должно произойти в UI, горутина отправляет
замыканием в канал `a.calls`, который главный цикл читает наравне
с событиями tcell. В `internal/ui` — окно прогресса `ui.Progress`
и `Stack.Remove`.

**Tech Stack:** Go 1.27, `github.com/gdamore/tcell/v2` v2.13.x,
`/usr/bin/trash` (macOS 14+).

**Spec:** `docs/superpowers/specs/2026-09-28-terminal-commander-design.md`
(этап 4: «`ops`: F5 / F6 / F7 / F8 / Shift-F8 с прогрессом и ошибками»,
раздел «Операции с файлами»).

## Global Constraints

- Модуль `github.com/navoznov/terminal-commander`; бинарник `tc`.
- `fs`, `ops` не импортируют `tcell` и ничего не знают об экране; `panel` не знает о другой панели, меню и диалогах; `ui` не импортирует `panel`, `fs`, `ops`, `app`; только `app` знает обо всём.
- Источник операции: выделенные файлы активной панели, если их нет — файл под курсором (`..` не участвует). Назначение по умолчанию — путь другой панели.
- F5 / F6: если назначение — существующая папка, файлы кладутся внутрь; иначе (один источник) — это новое имя. Копирование рекурсивное; сохраняются права и время изменения; симлинки копируются как ссылки.
- Конфликт имён: Overwrite / Skip / Overwrite all / Skip all / Cancel.
- Копировать/перемещать папку внутрь неё самой нельзя (ошибка).
- F6: на одном томе — `os.Rename`; при `EXDEV` — копирование и удаление источника после успешного копирования.
- F7: поле имени; `os.MkdirAll`; после создания курсор на новой папке.
- F8: подтверждение, затем `/usr/bin/trash`.
- Shift-F8: красный диалог; для непустых папок «Directory X is not empty. Delete it?» Delete / All / Skip / Cancel; `os.RemoveAll`.
- Прогресс: окно появляется, если операция длится дольше 300 мс; имя текущего файла, полоса текущего файла и общая (по байтам), кнопка Cancel. При отмене недокопированный файл удаляется.
- Ошибки: красный диалог с текстом и кнопками Retry / Skip / Cancel. После операции обе панели перечитываются, выделение снимается с успешно обработанных файлов.
- Цвета: диалог black / lightgray, рамка white; кнопка в фокусе black / yellow; ошибки и Shift-F8 white / red; тень +2 колонки, +1 строка.
- Коммиты — Conventional Commits на английском, **без** `Co-Authored-By` и пометок «Generated with Claude Code».
- Работа в ветке `feature/stage-4` от `main`; в `main` — только через PR.

## Решения этого этапа (уточняют спеку)

- Текст диалогов: при одном источнике — имя в кавычках (`Copy "a.txt" to:`), при нескольких — число (`Copy 3 files to:`). Так же в F6 (`Rename or move … to:`), F8 (`Move … to Trash?`), Shift-F8 (`Delete … permanently?`).
- Кнопки: F5 — Copy / Cancel, F6 — Move / Cancel, F7 — OK / Cancel, F8 и Shift-F8 — Delete / Cancel.
- Путь в поле F5/F6/F7: относительный считается от каталога активной панели; `~` и `~/…` — домашний каталог. Путь, оканчивающийся на `/`, и назначение при нескольких источниках — это папка; если её нет, она создаётся.
- Копирование папки в существующую папку с тем же именем сливает содержимое; о конфликтах спрашивается по файлам. Перемещение папки на существующую папку — ошибка «X already exists» (без слияния). Файл нельзя записать поверх папки и наоборот — ошибка «cannot overwrite X».
- Копирование файла на самого себя (обе панели в одном каталоге) — ошибка «cannot copy X onto itself». Перемещение на тот же файл под другим регистром имени (`foo` → `Foo` на APFS без учёта регистра) — переименование.
- Проверка «папка внутрь себя» сравнивает файлы через `os.SameFile` по всем предкам назначения, поэтому ловит и путь в другом регистре, и путь через симлинк.
- Шкалы прогресса: F5 — текущий файл и общая по байтам; F6 — текущий файл и общая по числу источников (подсчёт байт потребовал бы обхода дерева даже для мгновенного `rename`); F8 / Shift-F8 — одна общая по числу источников.
- До появления окна прогресса (первые 300 мс) клавиши игнорируются; потом Esc, Enter или `c` — отмена. Отмена ждёт конца текущего блока / файла; окно закрывается, когда операция действительно остановилась.
- Вопросы из операции (конфликт, непустая папка, ошибка) показываются поверх окна прогресса; Esc в них = Cancel.
- Shift-F8 распознаётся и как F8 с Shift, и как `F20` (так его шлют некоторые терминалы).
- Корзина вызывается по одному пути на запуск `/usr/bin/trash` — так известно, какой файл не удалось удалить. Путь к команде — переменная `ops.TrashCmd` (тесты `app` подменяют её на `/bin/rm`, чтобы не засорять Корзину).
- Каталог назначения после копирования не открывается и курсор в другой панели не двигается — обе панели только перечитываются.

## Review Focus

1. **Папка внутрь самой себя** — через другой регистр имени или через симлинк: ошибка, а не бесконечная рекурсия. Тесты: `TestCopyDirIntoItself`, `TestInsideFollowsSymlinks` (Task 3), `TestMoveDirIntoItself` (Task 4).
2. **Отмена посреди файла / посреди межтомового перемещения** — недокопированный файл удалён, источник перемещения не удалён. Тесты: `TestCopyCancelRemovesPartialFile` (Task 3), `TestMoveAcrossVolumesCanceledKeepsSource` (Task 4).
3. **Смена только регистра имени** (`foo` → `Foo` на APFS без учёта регистра) — переименование, а не «into itself» и не «onto itself». Тест: `TestMoveChangesCaseOnly` (Task 4).
4. **Клавиши во время операции** не доходят до панелей; до появления окна игнорируются; Esc потом отменяет, окно уходит только после остановки. Тест: `TestProgressAppearsAndCancels` (Task 7).
5. **Частичный успех** — пропущенные (Skip) файлы остаются выделенными, скопированные — нет. Тест: `TestCopyConflictDialogSkip` (Task 8).

---

## Карта файлов

```
internal/ui/stack.go             + Stack.Remove
internal/ui/progress.go          Progress — окно прогресса
internal/panel/select.go         + Sources
internal/ops/job.go              Answer, Progress, Job, do/fail/overwrite
internal/ops/copy.go             Copy, Move, targets, copyItem, copyDir, copyFile, moveItem, inside, sameFile
internal/ops/remove.go           Remove, Trash, TrashCmd
internal/app/app.go              поле calls/op, Run с select, новые клавиши
internal/app/files.go            resolve, subject, sources, mkdir, trash, remove, transfer
internal/app/ops.go              op, run, post, ask*, showProgress, finish
internal/app/menu.go             пункты Files → Copy … Delete permanently
README.md                        статус этапов 3–4, клавиши
```

---

### Task 1: Окно прогресса и `Stack.Remove`

**Files:**
- Modify: `internal/ui/stack.go`
- Create: `internal/ui/progress.go`
- Test: `internal/ui/stack_test.go`, `internal/ui/progress_test.go`, `internal/ui/testdata/progress.golden`

**Interfaces:**
- Consumes: `window`, `drawTitle` (`internal/ui/window.go`), `term.Fit`, стили `term.DialogStyle`, `term.DialogFrameStyle`, `term.ButtonFocusStyle`.
- Produces:
  - `func (s *Stack) Remove(v View)` — убирает `v` из стека, где бы он ни был.
  - `type Progress struct { Title, File string; Bars []float64; Hidden bool; Cancel func() }` — `View`; `HandleKey` всегда возвращает `false` (окно убирает владелец); при `Hidden` не рисуется и игнорирует клавиши; Esc / Enter / `c` / `C` вызывают `Cancel`.

- [ ] **Step 1: Write the failing tests**

В конец `internal/ui/stack_test.go`:

```go
func TestStackRemove(t *testing.T) {
	var s ui.Stack
	a := &fakeView{onKey: func() bool { return false }}
	b := &fakeView{onKey: func() bool { return false }}
	s.Push(a)
	s.Push(b)
	s.Remove(a)
	if s.Len() != 1 || s.Top() != b {
		t.Fatalf("len %d top %v", s.Len(), s.Top())
	}
	s.Remove(a) // not in the stack: no-op
	if s.Len() != 1 {
		t.Fatalf("len %d", s.Len())
	}
}
```

Создать `internal/ui/progress_test.go`:

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

func TestProgressKeysCancel(t *testing.T) {
	for _, ev := range []*tcell.EventKey{key(tcell.KeyEscape), key(tcell.KeyEnter), runeKey('c'), runeKey('C')} {
		n := 0
		p := &ui.Progress{Cancel: func() { n++ }}
		if p.HandleKey(ev) || n != 1 {
			t.Fatalf("%v: finished or cancels %d", ev.Name(), n)
		}
	}
	n := 0
	p := &ui.Progress{Cancel: func() { n++ }}
	p.HandleKey(runeKey('x'))
	if n != 0 {
		t.Fatal("x cancelled")
	}
}

func TestProgressHiddenIgnoresKeysAndDrawsNothing(t *testing.T) {
	n := 0
	p := &ui.Progress{Title: "Copy", Bars: []float64{0.5}, Hidden: true, Cancel: func() { n++ }}
	p.HandleKey(key(tcell.KeyEscape))
	s := termtest.NewScreen(t, 80, 25)
	p.Draw(term.Canvas{Screen: s}, 80, 25)
	if n != 0 || strings.Contains(termtest.Dump(s), "Copy") {
		t.Fatalf("cancels %d, drawn %v", n, strings.Contains(termtest.Dump(s), "Copy"))
	}
}

func TestProgressGolden(t *testing.T) {
	s := termtest.NewScreen(t, 80, 25)
	p := &ui.Progress{Title: "Copy", File: "autoexec.bat", Bars: []float64{0.5, 0.25}}
	p.Draw(term.Canvas{Screen: s}, 80, 25)
	termtest.Golden(t, "progress", termtest.Dump(s))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/`
Expected: FAIL — `s.Remove undefined`, `undefined: ui.Progress`.

- [ ] **Step 3: Implement**

В `internal/ui/stack.go` заменить `HandleKey` и добавить `Remove`:

```go
// HandleKey sends ev to the top view. A finished view is removed even when
// it opened another view while handling the key.
func (s *Stack) HandleKey(ev *tcell.EventKey) {
	v := s.Top()
	if v == nil || !v.HandleKey(ev) {
		return
	}
	s.Remove(v)
}

// Remove takes v off the stack wherever it is.
func (s *Stack) Remove(v View) {
	for i, x := range s.views {
		if x == v {
			s.views = append(s.views[:i], s.views[i+1:]...)
			return
		}
	}
}
```

Создать `internal/ui/progress.go`:

```go
package ui

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/term"
)

const progressWidth = 50

// Progress is the window of a running file operation: the current file,
// progress bars and a Cancel button. Keys never close it; its owner removes
// it when the operation stops.
type Progress struct {
	Title  string
	File   string
	Bars   []float64 // each 0…1, one row each
	Hidden bool      // not shown yet: drawn as nothing, keys ignored
	Cancel func()
}

func (p *Progress) HandleKey(ev *tcell.EventKey) bool {
	if p.Hidden {
		return false
	}
	switch {
	case ev.Key() == tcell.KeyEscape, ev.Key() == tcell.KeyEnter,
		ev.Key() == tcell.KeyRune && unicode.ToLower(ev.Rune()) == 'c':
		p.Cancel()
	}
	return false
}

// Draw uses the dialog layout: gray margin, double frame, one space of
// padding, then the file name, the bars and the Cancel button.
func (p *Progress) Draw(c term.Canvas, w, h int) {
	if p.Hidden {
		return
	}
	body := term.DialogStyle
	cw := min(progressWidth, w-10)
	ww, wh := cw+8, len(p.Bars)+6
	x, y := max((w-ww)/2, 0), max((h-wh)/2, 0)
	window(c, x, y, ww, wh, body)
	c.Box(x+2, y+1, ww-4, wh-2, term.DialogFrameStyle)
	drawTitle(c, x, y+1, ww, p.Title, term.DialogFrameStyle)
	c.Text(x+4, y+2, term.Fit(p.File, cw), cw, body)
	for i, f := range p.Bars {
		n := int(max(min(f, 1), 0)*float64(cw) + 0.5)
		c.Text(x+4, y+3+i, strings.Repeat("█", n)+strings.Repeat("░", cw-n), cw, body)
	}
	b := " Cancel "
	c.Text(x+4+(cw-len(b))/2, y+3+len(p.Bars), b, len(b), term.ButtonFocusStyle)
}
```

- [ ] **Step 4: Create the golden file and run the tests**

Run: `go test ./internal/ui/ -run TestProgressGolden -update && go test ./internal/ui/`
Expected: PASS.

Открыть `internal/ui/testdata/progress.golden` и проверить глазами: серое окно 58×8 по центру (x=11, y=8), двойная белая рамка (`G`), ` Copy ` в разрыве верхней рамки, строка `autoexec.bat`, полоса из 25 `█` и 25 `░`, полоса из 13 `█` и 37 `░`, ` Cancel ` кодом `B` по центру, тень `x` справа (2 колонки) и снизу (1 строка).

- [ ] **Step 5: Commit**

```bash
go vet ./... && go test ./...
git add internal/ui
git commit -m "feat(ui): add progress window and Stack.Remove"
```

---

### Task 2: Источники операции в панели

**Files:**
- Modify: `internal/panel/select.go`
- Test: `internal/panel/select_test.go`

**Interfaces:**
- Consumes: `Panel.Entries`, `Panel.Selected`, `Panel.Current()`.
- Produces: `func (p *Panel) Sources() []string` — имена выделенных записей в порядке панели, иначе имя записи под курсором; `..` никогда; пустая панель или курсор на `..` без выделения — `nil`.

- [ ] **Step 1: Write the failing test**

В конец `internal/panel/select_test.go`:

```go
func TestSources(t *testing.T) {
	p := selectDemo()
	if got := p.Sources(); got != nil {
		t.Fatalf("cursor on ..: %q", got)
	}
	p.Cursor = 2
	if got := strings.Join(p.Sources(), " "); got != "a.txt" {
		t.Fatalf("cursor: %q", got)
	}
	p.Selected["c.go"] = true
	p.Selected["docs.txt"] = true
	if got := strings.Join(p.Sources(), " "); got != "docs.txt c.go" {
		t.Fatalf("selection: %q", got)
	}
	if got := New().Sources(); got != nil {
		t.Fatalf("empty panel: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/panel/ -run TestSources`
Expected: FAIL — `p.Sources undefined`.

- [ ] **Step 3: Implement**

В конец `internal/panel/select.go`:

```go
// Sources returns the names a file operation works on: the selected entries
// in panel order, or else the entry under the cursor. ".." never counts.
func (p *Panel) Sources() []string {
	var names []string
	for _, e := range p.Entries {
		if p.Selected[e.Name] {
			names = append(names, e.Name)
		}
	}
	if names == nil {
		if e := p.Current(); e != nil && !e.IsUp {
			names = []string{e.Name}
		}
	}
	return names
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/panel/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/panel
git commit -m "feat(panel): add Sources for file operations"
```

---

### Task 3: `ops` — задание и копирование

**Files:**
- Create: `internal/ops/job.go`, `internal/ops/copy.go`
- Test: `internal/ops/copy_test.go`, `internal/ops/helpers_test.go`

**Interfaces:**
- Consumes: только стандартная библиотека.
- Produces:
  - `type Answer int` с константами `Cancel` (0), `Yes` (перезаписать / удалить / повторить), `Skip`, `All` (перезаписать все / удалить все), `SkipAll`.
  - `type Progress struct { File string; FileDone, FileSize, Done, Total int64 }`.
  - `type Job struct { Progress func(Progress); Conflict func(dst string) Answer; NotEmpty func(dir string) Answer; Fail func(err error) Answer; … }` — все четыре колбэка обязательны; вызываются на горутине операции; вопросы блокируют её до ответа.
  - `func (j *Job) Cancel()`, `func (j *Job) Canceled() bool` — безопасны из любой горутины.
  - `func Copy(j *Job, srcs []string, dst string) []string` — возвращает имена (`filepath.Base`) источников, скопированных полностью.
  - Внутренние, для Task 4: `(j *Job) do(func() error) bool`, `(j *Job) fail(error) bool`, `(j *Job) overwrite(dst string) bool`, `(j *Job) targets(srcs []string, dst string, move bool) ([]string, bool)`, `(j *Job) copyItem(src, dst string) bool`, `inside(path, dir string) bool`, `sameFile(a, b string) bool`, `size(path string) int64`, поле `j.countBytes`.

- [ ] **Step 1: Write the test helpers**

Создать `internal/ops/helpers_test.go`:

```go
package ops

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// script is a scripted user: each question takes the next answer and is
// recorded in asked. Without answers left it cancels.
type script struct {
	answers []Answer
	asked   []string
}

func (s *script) next(q string) Answer {
	s.asked = append(s.asked, q)
	if len(s.answers) == 0 {
		return Cancel
	}
	a := s.answers[0]
	s.answers = s.answers[1:]
	return a
}

func newJob(s *script) *Job {
	return &Job{
		Progress: func(Progress) {},
		Conflict: func(dst string) Answer { return s.next("conflict " + filepath.Base(dst)) },
		NotEmpty: func(dir string) Answer { return s.next("not empty " + filepath.Base(dir)) },
		Fail:     func(err error) Answer { return s.next("fail " + err.Error()) },
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

var stamp = time.Date(1994, 5, 31, 6, 22, 0, 0, time.UTC)

// write creates path with content, mode and the fixed stamp time, making
// parent directories.
func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0o755))
	must(t, os.WriteFile(path, []byte(content), mode))
	must(t, os.Chmod(path, mode))
	must(t, os.Chtimes(path, stamp, stamp))
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	must(t, err)
	return string(b)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
```

- [ ] **Step 2: Write the failing tests**

Создать `internal/ops/copy_test.go`:

```go
package ops

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFileIntoDirKeepsModeAndTime(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	write(t, src, "hello", 0o640)
	dst := filepath.Join(dir, "out")
	must(t, os.Mkdir(dst, 0o755))
	var s script
	j := newJob(&s)
	var last Progress
	j.Progress = func(p Progress) { last = p }
	done := Copy(j, []string{src}, dst)
	if strings.Join(done, " ") != "a.txt" || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	out := filepath.Join(dst, "a.txt")
	fi, err := os.Stat(out)
	must(t, err)
	if read(t, out) != "hello" || fi.Mode().Perm() != 0o640 || !fi.ModTime().Equal(stamp) {
		t.Fatalf("mode %v time %v", fi.Mode(), fi.ModTime())
	}
	if last.Total != 5 || last.Done != 5 || last.FileSize != 5 || last.FileDone != 5 {
		t.Fatalf("progress %+v", last)
	}
	if !exists(src) {
		t.Fatal("source removed")
	}
}

func TestCopyToNewName(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	write(t, src, "x", 0o644)
	var s script
	Copy(newJob(&s), []string{src}, filepath.Join(dir, "b.txt"))
	if read(t, filepath.Join(dir, "b.txt")) != "x" {
		t.Fatal("not copied")
	}
}

func TestCopyTreeWithSymlink(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	write(t, filepath.Join(d, "x.txt"), "x", 0o644)
	write(t, filepath.Join(d, "sub", "y.txt"), "y", 0o600)
	must(t, os.Symlink("x.txt", filepath.Join(d, "link")))
	must(t, os.Chmod(d, 0o750))
	must(t, os.Chtimes(d, stamp, stamp))
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	if done := Copy(newJob(&s), []string{d}, out); strings.Join(done, " ") != "d" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	c := filepath.Join(out, "d")
	if read(t, filepath.Join(c, "x.txt")) != "x" || read(t, filepath.Join(c, "sub", "y.txt")) != "y" {
		t.Fatal("files not copied")
	}
	if target, err := os.Readlink(filepath.Join(c, "link")); err != nil || target != "x.txt" {
		t.Fatalf("link %q %v", target, err)
	}
	fi, err := os.Stat(c)
	must(t, err)
	if fi.Mode().Perm() != 0o750 || !fi.ModTime().Equal(stamp) {
		t.Fatalf("dir mode %v time %v", fi.Mode(), fi.ModTime())
	}
}

func TestCopyManyCreatesDestination(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, a, "a", 0o644)
	write(t, b, "b", 0o644)
	dst := filepath.Join(dir, "new", "place")
	var s script
	if done := Copy(newJob(&s), []string{a, b}, dst); len(done) != 2 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "a" || read(t, filepath.Join(dst, "b")) != "b" {
		t.Fatal("not copied into the new directory")
	}
}

func TestCopyTrailingSlashMeansDirectory(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	write(t, a, "a", 0o644)
	var s script
	Copy(newJob(&s), []string{a}, filepath.Join(dir, "new")+"/")
	if read(t, filepath.Join(dir, "new", "a")) != "a" {
		t.Fatal("not copied into new/")
	}
}

// conflictDir makes src/a, src/b ("new") and dst/a, dst/b ("old").
func conflictDir(t *testing.T) (srcs []string, dst string) {
	dir := t.TempDir()
	dst = filepath.Join(dir, "dst")
	for _, n := range []string{"a", "b"} {
		write(t, filepath.Join(dir, "src", n), "new", 0o644)
		write(t, filepath.Join(dst, n), "old", 0o644)
		srcs = append(srcs, filepath.Join(dir, "src", n))
	}
	return srcs, dst
}

func TestCopyConflictSkipThenOverwrite(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{Skip, Yes}}
	done := Copy(newJob(&s), srcs, dst)
	if strings.Join(done, " ") != "b" || strings.Join(s.asked, ",") != "conflict a,conflict b" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "old" || read(t, filepath.Join(dst, "b")) != "new" {
		t.Fatal("wrong files overwritten")
	}
}

func TestCopyOverwriteAllAsksOnce(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{All}}
	if done := Copy(newJob(&s), srcs, dst); len(done) != 2 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "new" || read(t, filepath.Join(dst, "b")) != "new" {
		t.Fatal("not overwritten")
	}
}

func TestCopySkipAllAsksOnce(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{SkipAll}}
	if done := Copy(newJob(&s), srcs, dst); len(done) != 0 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "b")) != "old" {
		t.Fatal("overwritten")
	}
}

func TestCopyCancelOnConflictStops(t *testing.T) {
	srcs, dst := conflictDir(t)
	var s script // no answers: Cancel
	j := newJob(&s)
	if done := Copy(j, srcs, dst); len(done) != 0 || len(s.asked) != 1 || !j.Canceled() {
		t.Fatalf("done %q asked %q canceled %v", done, s.asked, j.Canceled())
	}
}

func TestCopyMergesDirectories(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "d", "x"), "x", 0o644)
	write(t, filepath.Join(dir, "out", "d", "old"), "old", 0o644)
	var s script
	if done := Copy(newJob(&s), []string{filepath.Join(dir, "d")}, filepath.Join(dir, "out")); len(done) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(filepath.Join(dir, "out", "d", "x")) || !exists(filepath.Join(dir, "out", "d", "old")) {
		t.Fatal("not merged")
	}
}

func TestCopyFileOverDirFails(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "src", "a"), "a", 0o644)
	must(t, os.MkdirAll(filepath.Join(dir, "dst", "a"), 0o755))
	s := script{answers: []Answer{Skip}}
	Copy(newJob(&s), []string{filepath.Join(dir, "src", "a")}, filepath.Join(dir, "dst"))
	if len(s.asked) != 1 || !strings.Contains(s.asked[0], "cannot overwrite") {
		t.Fatalf("asked %q", s.asked)
	}
}

func TestCopyDirIntoItself(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	must(t, os.MkdirAll(filepath.Join(d, "sub"), 0o755))
	s := script{answers: []Answer{Skip}}
	done := Copy(newJob(&s), []string{d}, filepath.Join(d, "sub"))
	if len(done) != 0 || len(s.asked) != 1 || !strings.Contains(s.asked[0], "into itself") {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(filepath.Join(d, "sub", "d")) {
		t.Fatal("copied into itself")
	}
}

func TestCopyOntoItself(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	write(t, a, "a", 0o644)
	s := script{answers: []Answer{Skip}}
	Copy(newJob(&s), []string{a}, dir)
	if len(s.asked) != 1 || !strings.Contains(s.asked[0], "onto itself") || read(t, a) != "a" {
		t.Fatalf("asked %q", s.asked)
	}
}

func TestInsideFollowsSymlinks(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	must(t, os.MkdirAll(filepath.Join(d, "sub"), 0o755))
	must(t, os.Symlink(d, filepath.Join(dir, "link")))
	if !inside(filepath.Join(dir, "link", "sub", "new"), d) {
		t.Fatal("path through a symlink not detected")
	}
	if !inside(d, d) {
		t.Fatal("the directory itself not detected")
	}
	if inside(filepath.Join(dir, "other"), d) {
		t.Fatal("sibling detected")
	}
	if inside(filepath.Join(d, "sub"), filepath.Join(dir, "link")) {
		t.Fatal("a symlink is copied as a link, never into itself")
	}
}

func TestCopyCancelRemovesPartialFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "big")
	must(t, os.WriteFile(src, bytes.Repeat([]byte{'x'}, 3*blockSize), 0o644))
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	j := newJob(&s)
	j.Progress = func(p Progress) {
		if p.FileDone >= blockSize {
			j.Cancel()
		}
	}
	if done := Copy(j, []string{src}, out); len(done) != 0 || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(filepath.Join(out, "big")) {
		t.Fatal("partial file left")
	}
}

func TestCopyErrorRetry(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "locked")
	write(t, src, "x", 0o000)
	t.Cleanup(func() { os.Chmod(src, 0o644) })
	var s script
	j := newJob(&s)
	fails := 0
	j.Fail = func(err error) Answer {
		fails++
		must(t, os.Chmod(src, 0o644)) // the user fixed it and pressed Retry
		return Yes
	}
	dst := filepath.Join(dir, "copy")
	if done := Copy(j, []string{src}, dst); len(done) != 1 || fails != 1 || read(t, dst) != "x" {
		t.Fatalf("done %q fails %d", done, fails)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/ops/`
Expected: FAIL — пакет не компилируется: `undefined: Answer`, `undefined: Copy` и т. д.

- [ ] **Step 4: Implement `job.go`**

Создать `internal/ops/job.go`:

```go
// Package ops copies, moves and deletes files. An operation runs on the
// caller's goroutine, reports progress and asks the user through its Job,
// and checks for cancellation between files and between copy blocks.
package ops

import "sync/atomic"

// Answer is the user's decision on a question from an operation.
type Answer int

const (
	Cancel  Answer = iota
	Yes            // overwrite, delete, retry
	Skip           // leave this file alone
	All            // overwrite all, delete all
	SkipAll        // skip all conflicts
)

// Progress is a snapshot of a running operation. Done and Total count bytes
// for Copy and sources for the other operations.
type Progress struct {
	File               string // path being processed
	FileDone, FileSize int64  // bytes of the current file
	Done, Total        int64
}

// Job carries an operation's callbacks and its cancel flag. The callbacks
// run on the operation's goroutine; the questions block it until answered.
type Job struct {
	Progress func(Progress)
	// Conflict asks about an existing destination file: Yes (overwrite),
	// Skip, All (overwrite all), SkipAll or Cancel.
	Conflict func(dst string) Answer
	// NotEmpty asks whether to delete a non-empty directory: Yes, All, Skip
	// or Cancel.
	NotEmpty func(dir string) Answer
	// Fail reports an error: Yes (retry), Skip or Cancel.
	Fail func(err error) Answer

	canceled     atomic.Bool
	progress     Progress
	countBytes   bool // Done counts bytes, not sources
	overwriteAll bool
	skipAll      bool
	deleteAll    bool
}

func (j *Job) Cancel()        { j.canceled.Store(true) }
func (j *Job) Canceled() bool { return j.canceled.Load() }

// do runs f until it succeeds, or the user skips or cancels, and reports
// whether f succeeded. Errors after a cancel are not reported.
func (j *Job) do(f func() error) bool {
	for !j.Canceled() {
		err := f()
		if err == nil {
			return true
		}
		if j.Canceled() {
			return false
		}
		switch j.Fail(err) {
		case Yes:
			continue
		case Skip:
			return false
		default:
			j.Cancel()
		}
	}
	return false
}

// fail reports err like do and returns false unless a retry succeeds, which
// for a fixed error it never does.
func (j *Job) fail(err error) bool {
	return j.do(func() error { return err })
}

// overwrite asks whether to replace the existing dst, remembering the "all"
// answers.
func (j *Job) overwrite(dst string) bool {
	switch {
	case j.overwriteAll:
		return true
	case j.skipAll:
		return false
	}
	switch j.Conflict(dst) {
	case Yes:
		return true
	case All:
		j.overwriteAll = true
		return true
	case Skip:
		return false
	case SkipAll:
		j.skipAll = true
		return false
	}
	j.Cancel()
	return false
}
```

- [ ] **Step 5: Implement `copy.go`**

Создать `internal/ops/copy.go`:

```go
package ops

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
)

const blockSize = 1 << 20

// Copy copies srcs into the directory dst, or to the new name dst when there
// is one source and dst is not a directory. Directories are copied
// recursively and merged into existing ones; permissions, modification times
// and symlinks are kept. It returns the names of the sources copied
// completely.
func Copy(j *Job, srcs []string, dst string) []string {
	j.countBytes = true
	for _, s := range srcs {
		j.progress.Total += size(s)
	}
	targets, ok := j.targets(srcs, dst, false)
	if !ok {
		return nil
	}
	var done []string
	for i, src := range srcs {
		if j.Canceled() {
			break
		}
		if j.copyTop(src, targets[i]) {
			done = append(done, filepath.Base(src))
		}
	}
	return done
}

// size is the total size of the regular files under path.
func size(path string) int64 {
	var n int64
	filepath.WalkDir(path, func(_ string, d iofs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if fi, err := d.Info(); err == nil {
				n += fi.Size()
			}
		}
		return nil
	})
	return n
}

// targets maps each source to its destination path. With several sources,
// a trailing "/" or an existing directory, dst is a directory, created if
// missing. A single source that is dst itself (maybe in another case of the
// name) is renamed to dst when moving.
func (j *Job) targets(srcs []string, dst string, move bool) ([]string, bool) {
	intoDir := len(srcs) > 1 || strings.HasSuffix(dst, "/")
	dst = filepath.Clean(dst)
	if fi, err := os.Stat(dst); err == nil && fi.IsDir() {
		intoDir = !(move && len(srcs) == 1 && sameFile(srcs[0], dst))
	}
	if !intoDir {
		return []string{dst}, true
	}
	if !j.do(func() error { return os.MkdirAll(dst, 0o755) }) {
		return nil, false
	}
	out := make([]string, len(srcs))
	for i, s := range srcs {
		out[i] = filepath.Join(dst, filepath.Base(s))
	}
	return out, true
}

// sameFile reports whether a and b both exist and are the same file (not
// following symlinks).
func sameFile(a, b string) bool {
	fa, err := os.Lstat(a)
	if err != nil {
		return false
	}
	fb, err := os.Lstat(b)
	return err == nil && os.SameFile(fa, fb)
}

// inside reports whether path is the directory dir or lies under it. The
// file system decides, so a different case of a name or a symlink on the way
// is caught. A dir that is a symlink is never "inside": links are copied as
// links.
func inside(path, dir string) bool {
	di, err := os.Lstat(dir)
	if err != nil || !di.IsDir() {
		return false
	}
	for p := filepath.Clean(path); ; p = filepath.Dir(p) {
		if fi, err := os.Stat(p); err == nil && os.SameFile(fi, di) {
			return true
		}
		if p == filepath.Dir(p) {
			return false
		}
	}
}

// copyTop copies one source after the checks that only make sense at the
// top level.
func (j *Job) copyTop(src, dst string) bool {
	switch {
	case sameFile(src, dst):
		return j.fail(fmt.Errorf("cannot copy %s onto itself", src))
	case inside(dst, src):
		return j.fail(fmt.Errorf("cannot copy %s into itself", src))
	}
	return j.copyItem(src, dst)
}

// copyItem copies src to dst, asking about an existing dst, and reports
// whether everything was copied.
func (j *Job) copyItem(src, dst string) bool {
	var si os.FileInfo
	if !j.do(func() (err error) { si, err = os.Lstat(src); return err }) {
		return false
	}
	if di, err := os.Lstat(dst); err == nil {
		switch {
		case si.IsDir() && di.IsDir():
			return j.copyDir(src, dst, si, true)
		case si.IsDir() || di.IsDir():
			return j.fail(fmt.Errorf("cannot overwrite %s", dst))
		case !j.overwrite(dst):
			return false
		case !j.do(func() error { return os.Remove(dst) }):
			return false
		}
	}
	switch {
	case si.Mode()&os.ModeSymlink != 0:
		return j.do(func() error {
			target, err := os.Readlink(src)
			if err != nil {
				return err
			}
			return os.Symlink(target, dst)
		})
	case si.IsDir():
		return j.copyDir(src, dst, si, false)
	case si.Mode().IsRegular():
		return j.do(func() error { return j.copyFile(src, dst, si) })
	}
	return j.fail(fmt.Errorf("%s: unsupported file type", src))
}

// copyDir copies the contents of src into dst, creating dst unless it
// exists. A new dst gets the mode and time of src after its contents are in.
func (j *Job) copyDir(src, dst string, si os.FileInfo, exists bool) bool {
	if !exists && !j.do(func() error { return os.Mkdir(dst, 0o700) }) {
		return false
	}
	var names []string
	if !j.do(func() (err error) { names, err = readNames(src); return err }) {
		return false
	}
	ok := true
	for _, n := range names {
		if j.Canceled() {
			return false
		}
		ok = j.copyItem(filepath.Join(src, n), filepath.Join(dst, n)) && ok
	}
	if !exists {
		ok = j.do(func() error {
			if err := os.Chmod(dst, si.Mode().Perm()); err != nil {
				return err
			}
			return os.Chtimes(dst, si.ModTime(), si.ModTime())
		}) && ok
	}
	return ok && !j.Canceled()
}

func readNames(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

var errCanceled = errors.New("canceled")

// copyFile copies a regular file in blocks, reporting progress. A copy that
// fails or is canceled is removed and its bytes are taken off the total.
func (j *Job) copyFile(src, dst string, si os.FileInfo) (err error) {
	start := j.progress.Done
	j.progress.File, j.progress.FileDone, j.progress.FileSize = src, 0, si.Size()
	j.Progress(j.progress)
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(dst)
			if j.countBytes {
				j.progress.Done = start
			}
		}
	}()
	buf := make([]byte, blockSize)
	for {
		if j.Canceled() {
			return errCanceled
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			j.progress.FileDone += int64(n)
			if j.countBytes {
				j.progress.Done += int64(n)
			}
			j.Progress(j.progress)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if err := out.Chmod(si.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(dst, si.ModTime(), si.ModTime())
}
```

- [ ] **Step 6: Run tests**

Run: `go test -race ./internal/ops/`
Expected: PASS (все тесты из Step 2).

- [ ] **Step 7: Commit**

```bash
go vet ./... && go test ./...
git add internal/ops
git commit -m "feat(ops): copy files and directories with conflicts, errors and cancel"
```

---

### Task 4: `ops` — перемещение

**Files:**
- Modify: `internal/ops/copy.go`
- Test: `internal/ops/move_test.go`

**Interfaces:**
- Consumes: из Task 3 — `Job`, `do`, `fail`, `overwrite`, `targets(srcs, dst, move)`, `copyItem`, `inside`, `sameFile`, поле `countBytes`, `j.progress`.
- Produces: `func Move(j *Job, srcs []string, dst string) []string` — имена перемещённых источников; `Progress.Done/Total` — число источников. Переменная пакета `var rename = os.Rename` (тесты подменяют её, чтобы получить `EXDEV`).

- [ ] **Step 1: Write the failing tests**

Создать `internal/ops/move_test.go`:

```go
package ops

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestMoveRenames(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	write(t, a, "a", 0o644)
	var s script
	var last Progress
	j := newJob(&s)
	j.Progress = func(p Progress) { last = p }
	done := Move(j, []string{a}, filepath.Join(dir, "b"))
	if strings.Join(done, " ") != "a" || exists(a) || read(t, filepath.Join(dir, "b")) != "a" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if last.Done != 1 || last.Total != 1 {
		t.Fatalf("progress %+v", last)
	}
}

func TestMoveManyIntoDir(t *testing.T) {
	dir := t.TempDir()
	a, d := filepath.Join(dir, "a"), filepath.Join(dir, "d")
	write(t, a, "a", 0o644)
	write(t, filepath.Join(d, "x"), "x", 0o644)
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	if done := Move(newJob(&s), []string{a, d}, out); len(done) != 2 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(a) || exists(d) || read(t, filepath.Join(out, "d", "x")) != "x" {
		t.Fatal("not moved")
	}
}

// crossDevice makes rename fail with EXDEV for the rest of the test.
func crossDevice(t *testing.T) {
	rename = func(a, b string) error { return &os.LinkError{Op: "rename", Old: a, New: b, Err: syscall.EXDEV} }
	t.Cleanup(func() { rename = os.Rename })
}

func TestMoveAcrossVolumesCopiesAndDeletes(t *testing.T) {
	crossDevice(t)
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	write(t, filepath.Join(d, "sub", "x"), "x", 0o640)
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	if done := Move(newJob(&s), []string{d}, out); len(done) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(d) || read(t, filepath.Join(out, "d", "sub", "x")) != "x" {
		t.Fatal("not moved")
	}
}

func TestMoveAcrossVolumesOverwrites(t *testing.T) {
	crossDevice(t)
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{Yes, Skip}}
	done := Move(newJob(&s), srcs, dst)
	if strings.Join(done, " ") != "a" || len(s.asked) != 2 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "new" || exists(srcs[0]) || !exists(srcs[1]) {
		t.Fatal("wrong result")
	}
}

func TestMoveAcrossVolumesCanceledKeepsSource(t *testing.T) {
	crossDevice(t)
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	write(t, filepath.Join(d, "x"), "x", 0o644)
	write(t, filepath.Join(d, "y"), "y", 0o644)
	out := filepath.Join(dir, "out")
	must(t, os.Mkdir(out, 0o755))
	var s script
	j := newJob(&s)
	j.Progress = func(p Progress) {
		if p.FileSize > 0 {
			j.Cancel() // during the first file
		}
	}
	if done := Move(j, []string{d}, out); len(done) != 0 {
		t.Fatalf("done %q", done)
	}
	if !exists(filepath.Join(d, "x")) || !exists(filepath.Join(d, "y")) {
		t.Fatal("source deleted after a canceled copy")
	}
}

func TestMoveConflict(t *testing.T) {
	srcs, dst := conflictDir(t)
	s := script{answers: []Answer{Skip, Yes}}
	done := Move(newJob(&s), srcs, dst)
	if strings.Join(done, " ") != "b" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if read(t, filepath.Join(dst, "a")) != "old" || !exists(srcs[0]) {
		t.Fatal("skipped file touched")
	}
	if read(t, filepath.Join(dst, "b")) != "new" || exists(srcs[1]) {
		t.Fatal("b not moved")
	}
}

func TestMoveOntoExistingDirFails(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "d", "x"), "x", 0o644)
	must(t, os.MkdirAll(filepath.Join(dir, "out", "d"), 0o755))
	s := script{answers: []Answer{Skip}}
	Move(newJob(&s), []string{filepath.Join(dir, "d")}, filepath.Join(dir, "out"))
	if len(s.asked) != 1 || !strings.Contains(s.asked[0], "already exists") {
		t.Fatalf("asked %q", s.asked)
	}
	if !exists(filepath.Join(dir, "d", "x")) {
		t.Fatal("source lost")
	}
}

func TestMoveDirIntoItself(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "d")
	must(t, os.MkdirAll(filepath.Join(d, "sub"), 0o755))
	s := script{answers: []Answer{Skip}}
	done := Move(newJob(&s), []string{d}, filepath.Join(d, "sub"))
	if len(done) != 0 || len(s.asked) != 1 || !strings.Contains(s.asked[0], "into itself") {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(filepath.Join(d, "sub")) {
		t.Fatal("source changed")
	}
}

func TestMoveChangesCaseOnly(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "foo")
	write(t, filepath.Join(d, "x"), "x", 0o644)
	if !exists(filepath.Join(dir, "FOO")) {
		t.Skip("case-sensitive file system")
	}
	var s script
	if done := Move(newJob(&s), []string{d}, filepath.Join(dir, "Foo")); len(done) != 1 || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	names, err := readNames(dir)
	must(t, err)
	if strings.Join(names, " ") != "Foo" {
		t.Fatalf("names %q", names)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ops/`
Expected: FAIL — `undefined: Move`, `undefined: rename`.

- [ ] **Step 3: Implement**

В `internal/ops/copy.go` добавить `"syscall"` в импорты и в конец файла:

```go
var rename = os.Rename

// Move moves srcs like Copy. Within one volume it renames; across volumes it
// copies and deletes the source only after a complete copy. It returns the
// names of the sources moved.
func Move(j *Job, srcs []string, dst string) []string {
	j.progress.Total = int64(len(srcs))
	targets, ok := j.targets(srcs, dst, true)
	if !ok {
		return nil
	}
	var done []string
	for i, src := range srcs {
		if j.Canceled() {
			break
		}
		j.progress.File, j.progress.FileDone, j.progress.FileSize = src, 0, 0
		j.Progress(j.progress)
		if j.moveItem(src, targets[i]) {
			done = append(done, filepath.Base(src))
		}
		j.progress.Done++
		j.Progress(j.progress)
	}
	return done
}

func (j *Job) moveItem(src, dst string) bool {
	if sameFile(src, dst) { // the same path, or a new case of the name
		return j.do(func() error { return rename(src, dst) })
	}
	var si os.FileInfo
	if !j.do(func() (err error) { si, err = os.Lstat(src); return err }) {
		return false
	}
	if inside(dst, src) {
		return j.fail(fmt.Errorf("cannot move %s into itself", src))
	}
	if di, err := os.Lstat(dst); err == nil {
		switch {
		case si.IsDir() && di.IsDir():
			return j.fail(fmt.Errorf("%s already exists", dst))
		case si.IsDir() || di.IsDir():
			return j.fail(fmt.Errorf("cannot overwrite %s", dst))
		case !j.overwrite(dst):
			return false
		}
	}
	crossed := false
	if !j.do(func() error {
		err := rename(src, dst)
		crossed = errors.Is(err, syscall.EXDEV)
		if crossed {
			return nil
		}
		return err
	}) {
		return false
	}
	if !crossed {
		return true
	}
	if _, err := os.Lstat(dst); err == nil && !j.do(func() error { return os.Remove(dst) }) {
		return false // the overwrite was agreed above
	}
	return j.copyItem(src, dst) && j.do(func() error { return os.RemoveAll(src) })
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/ops/`
Expected: PASS. `TestMoveChangesCaseOnly` на APFS по умолчанию выполняется (не SKIP) — проверить `go test -v -run TestMoveChangesCaseOnly ./internal/ops/`.

- [ ] **Step 5: Commit**

```bash
go vet ./... && go test ./...
git add internal/ops
git commit -m "feat(ops): move files, falling back to copy and delete across volumes"
```

---

### Task 5: `ops` — удаление навсегда и в Корзину

**Files:**
- Create: `internal/ops/remove.go`
- Test: `internal/ops/remove_test.go`

**Interfaces:**
- Consumes: из Task 3 — `Job`, `do`, поля `progress`, `deleteAll`; `readNames`.
- Produces:
  - `func Remove(j *Job, paths []string) []string` — удаляет навсегда; непустую папку — только после `NotEmpty` (Yes / All / Skip / Cancel). Возвращает имена удалённых.
  - `var TrashCmd = "/usr/bin/trash"`; `func Trash(j *Job, paths []string) []string` — по одному вызову `TrashCmd path` на путь.
  - `Progress.Done/Total` — число путей.

- [ ] **Step 1: Write the failing tests**

Создать `internal/ops/remove_test.go`:

```go
package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveFilesAndEmptyDirsWithoutAsking(t *testing.T) {
	dir := t.TempDir()
	a, e := filepath.Join(dir, "a"), filepath.Join(dir, "empty")
	write(t, a, "a", 0o644)
	must(t, os.Mkdir(e, 0o755))
	var s script
	var last Progress
	j := newJob(&s)
	j.Progress = func(p Progress) { last = p }
	done := Remove(j, []string{a, e})
	if strings.Join(done, " ") != "a empty" || len(s.asked) != 0 || exists(a) || exists(e) {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if last.Done != 2 || last.Total != 2 {
		t.Fatalf("progress %+v", last)
	}
}

// fullDirs makes two non-empty directories d1 and d2.
func fullDirs(t *testing.T) []string {
	dir := t.TempDir()
	var ds []string
	for _, n := range []string{"d1", "d2"} {
		write(t, filepath.Join(dir, n, "x"), "x", 0o644)
		ds = append(ds, filepath.Join(dir, n))
	}
	return ds
}

func TestRemoveAsksForNonEmptyDirs(t *testing.T) {
	ds := fullDirs(t)
	s := script{answers: []Answer{Skip, Yes}}
	done := Remove(newJob(&s), ds)
	if strings.Join(done, " ") != "d2" || strings.Join(s.asked, ",") != "not empty d1,not empty d2" {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(ds[0]) || exists(ds[1]) {
		t.Fatal("wrong directory deleted")
	}
}

func TestRemoveAllAsksOnce(t *testing.T) {
	ds := fullDirs(t)
	s := script{answers: []Answer{All}}
	if done := Remove(newJob(&s), ds); len(done) != 2 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
}

func TestRemoveCancelStops(t *testing.T) {
	ds := fullDirs(t)
	var s script // Cancel
	j := newJob(&s)
	if done := Remove(j, ds); len(done) != 0 || len(s.asked) != 1 || !j.Canceled() {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !exists(ds[0]) || !exists(ds[1]) {
		t.Fatal("deleted after cancel")
	}
}

func TestRemoveSymlinkToDirKeepsTarget(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "d", "x"), "x", 0o644)
	link := filepath.Join(dir, "link")
	must(t, os.Symlink(filepath.Join(dir, "d"), link))
	var s script
	if done := Remove(newJob(&s), []string{link}); len(done) != 1 || len(s.asked) != 0 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if exists(link) || !exists(filepath.Join(dir, "d", "x")) {
		t.Fatal("wrong thing deleted")
	}
}

func TestTrash(t *testing.T) {
	if _, err := os.Stat(TrashCmd); err != nil {
		t.Skip("no " + TrashCmd)
	}
	a := filepath.Join(t.TempDir(), "tc-trash-test.txt")
	write(t, a, "a", 0o644)
	var s script
	if done := Trash(newJob(&s), []string{a}); len(done) != 1 || len(s.asked) != 0 || exists(a) {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
}

func TestTrashMissingFileAsks(t *testing.T) {
	s := script{answers: []Answer{Skip}}
	missing := filepath.Join(t.TempDir(), "missing")
	if done := Trash(newJob(&s), []string{missing}); len(done) != 0 || len(s.asked) != 1 {
		t.Fatalf("done %q asked %q", done, s.asked)
	}
	if !strings.Contains(s.asked[0], "no such file") {
		t.Fatalf("asked %q", s.asked)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ops/`
Expected: FAIL — `undefined: Remove`, `undefined: Trash`, `undefined: TrashCmd`.

- [ ] **Step 3: Implement**

Создать `internal/ops/remove.go`:

```go
package ops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// TrashCmd moves a file to the Trash so that Finder can put it back
// (macOS 14+).
var TrashCmd = "/usr/bin/trash"

// Remove deletes paths permanently. A non-empty directory is deleted only
// after NotEmpty agrees. It returns the names of the deleted paths.
func Remove(j *Job, paths []string) []string {
	return j.each(paths, j.removeOne)
}

// Trash moves paths to the Trash, one TrashCmd run per path, and returns the
// names of the moved paths.
func Trash(j *Job, paths []string) []string {
	return j.each(paths, func(p string) bool {
		return j.do(func() error {
			if _, err := os.Lstat(p); err != nil {
				return err
			}
			if err := exec.Command(TrashCmd, p).Run(); err != nil {
				return fmt.Errorf("cannot move %s to the Trash: %w", p, err)
			}
			return nil
		})
	})
}

// each runs f on every path, counting paths as progress, and returns the
// names of the paths f succeeded on.
func (j *Job) each(paths []string, f func(string) bool) []string {
	j.progress.Total = int64(len(paths))
	var done []string
	for _, p := range paths {
		if j.Canceled() {
			break
		}
		j.progress.File = p
		j.Progress(j.progress)
		if f(p) {
			done = append(done, filepath.Base(p))
		}
		j.progress.Done++
		j.Progress(j.progress)
	}
	return done
}

func (j *Job) removeOne(path string) bool {
	if fi, err := os.Lstat(path); err == nil && fi.IsDir() && !emptyDir(path) && !j.deleteDir(path) {
		return false
	}
	return j.do(func() error { return os.RemoveAll(path) })
}

func emptyDir(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	names, _ := f.Readdirnames(1)
	return len(names) == 0
}

// deleteDir asks whether to delete the non-empty directory path,
// remembering "All".
func (j *Job) deleteDir(path string) bool {
	if j.deleteAll {
		return true
	}
	switch j.NotEmpty(path) {
	case Yes:
		return true
	case All:
		j.deleteAll = true
		return true
	case Skip:
		return false
	}
	j.Cancel()
	return false
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/ops/`
Expected: PASS. `TestTrash` выполняется (не SKIP) — файл `tc-trash-test.txt` оказывается в `~/.Trash`; это ожидаемо, спека требует реальный вызов `/usr/bin/trash`.

- [ ] **Step 5: Commit**

```bash
go vet ./... && go test ./...
git add internal/ops
git commit -m "feat(ops): delete permanently and move to the Trash"
```

---

### Task 6: F7 — создание папки

**Files:**
- Create: `internal/app/files.go`
- Modify: `internal/app/app.go` (клавиша F7), `internal/app/menu.go` (пункт Make directory)
- Test: `internal/app/files_test.go`

**Interfaces:**
- Consumes: `ui.Dialog`, `ui.NewInput`, `a.report`, `Panel.Reload`, `Panel.Focus`.
- Produces:
  - `func resolve(dir, home, s string) string` — `~`/`~/…` → домашний каталог; относительный путь — от `dir`; путь очищается (`filepath.Clean`), но завершающий `/` сохраняется.
  - `func (a *App) mkdir()` — диалог «Make directory».

- [ ] **Step 1: Write the failing tests**

Создать `internal/app/files_test.go`:

```go
package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ui"
)

func TestResolve(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"b", "/a/b"},
		{"b/../c", "/a/c"},
		{"sub/", "/a/sub/"},
		{"/x//y", "/x/y"},
		{"/", "/"},
		{"~", "/home"},
		{"~/x", "/home/x"},
		{"~x", "/a/~x"},
	} {
		if got := resolve("/a", "/home", c.in); got != c.want {
			t.Errorf("resolve(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func typeText(a *App, s string) {
	for _, r := range s {
		press(a, tcell.KeyRune, r, 0)
	}
}

func TestF7MakesDirectoryAndFocusesIt(t *testing.T) {
	a, dir := newApp(t)
	press(a, tcell.KeyF7, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Make directory" || d.Input == nil {
		t.Fatalf("top %#v", a.modals.Top())
	}
	typeText(a, "new/deep")
	press(a, tcell.KeyEnter, 0, 0)
	if fi, err := os.Stat(filepath.Join(dir, "new", "deep")); err != nil || !fi.IsDir() {
		t.Fatalf("not created: %v", err)
	}
	if !a.modals.Empty() || a.panels[0].Current().Name != "new" {
		t.Fatalf("modals %d current %s", a.modals.Len(), a.panels[0].Current().Name)
	}
	if !a.panels[1].Focus("new") {
		t.Fatal("the other panel was not re-read")
	}
}

func TestF7EmptyNameDoesNothing(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF7, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	if !a.modals.Empty() {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestF7ErrorIsReported(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF7, 0, 0)
	typeText(a, "sub/f.txt/x") // f.txt is a file
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || !d.Danger {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestMenuMakeDirectory(t *testing.T) {
	a, _ := newApp(t)
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyRight, 0, 0) // Files
	for i := 0; i < 5; i++ {       // Help → View → Edit → Copy → Rename/Move → Make directory
		press(a, tcell.KeyDown, 0, 0)
	}
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || d.Title != "Make directory" {
		t.Fatalf("top %#v", a.modals.Top())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/`
Expected: FAIL — `undefined: resolve`; после его появления — F7 не открывает диалог.

- [ ] **Step 3: Implement**

Создать `internal/app/files.go`:

```go
package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/navoznov/terminal-commander/internal/ui"
)

// resolve turns a typed path into an absolute one: "~" is the home directory
// and relative paths start at dir. A trailing "/" is kept: it tells ops that
// the destination is a directory.
func resolve(dir, home, s string) string {
	p := s
	switch {
	case s == "~":
		p = home
	case strings.HasPrefix(s, "~/"):
		p = home + s[1:]
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	p = filepath.Clean(p)
	if strings.HasSuffix(s, "/") && p != "/" {
		p += "/"
	}
	return p
}

// mkdir asks for a name and creates the directory with its parents, then
// puts the cursor on it.
func (a *App) mkdir() {
	p := a.panels[a.active]
	a.modals.Push(&ui.Dialog{
		Title:   "Make directory",
		Lines:   []string{"Create the directory"},
		Input:   ui.NewInput(""),
		Buttons: []string{"OK", "Cancel"},
		Done: func(b int, name string) {
			if b != 0 || name == "" {
				return
			}
			path := resolve(p.Path, a.home, name)
			if err := os.MkdirAll(path, 0o755); err != nil {
				a.report(err)
				return
			}
			a.reloadPanels()
			if rel, err := filepath.Rel(p.Path, filepath.Clean(path)); err == nil && !strings.HasPrefix(rel, "..") {
				p.Focus(strings.Split(rel, string(filepath.Separator))[0])
			}
		},
	})
}

func (a *App) reloadPanels() {
	for _, p := range a.panels {
		a.report(p.Reload())
	}
}
```

В `internal/app/app.go`, в `handleKey`, после `case tcell.KeyF9:` добавить:

```go
	case tcell.KeyF7:
		a.mkdir()
```

В `internal/app/menu.go` заменить строку пункта Make directory:

```go
			{Label: "Make directory", Key: "F7", Action: a.mkdir},
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/app/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
go vet ./... && go test ./...
git add internal/app
git commit -m "feat(app): F7 makes a directory"
```

---

### Task 7: Фоновые операции, окно прогресса, F8 и Shift-F8

**Files:**
- Create: `internal/app/ops.go`
- Modify: `internal/app/app.go` (поля `calls`, `op`; `New`; `Run`; клавиши F8, Shift-F8, F20), `internal/app/files.go` (`subject`, `sources`, `trash`, `remove`), `internal/app/menu.go` (пункты Delete, Delete permanently)
- Test: `internal/app/ops_test.go`

**Interfaces:**
- Consumes: из Task 1 — `ui.Progress`, `Stack.Remove`; из Task 2 — `Panel.Sources`; из Task 3–5 — `ops.Job`, `ops.Answer` и константы, `ops.Progress`, `ops.Remove`, `ops.Trash`, `ops.TrashCmd`; из Task 6 — `a.reloadPanels`.
- Produces:
  - поле `calls chan func()` (буфер 16) — замыкания для UI-горутины; `Run` выбирает между событиями экрана и `calls`.
  - поле `op *op` — операция, идущая сейчас (`nil`, если нет).
  - `func (a *App) run(title string, fileBar bool, work func(*ops.Job) []string)` — запускает `work` в горутине с окном прогресса; по окончании снимает выделение с возвращённых имён в исходной панели и перечитывает обе панели.
  - `func (a *App) ask(d *ui.Dialog, answers []ops.Answer) ops.Answer` — вызывается из горутины операции.
  - `func (a *App) askConflict(dst string) ops.Answer`, `askNotEmpty(dir string) ops.Answer`, `askFail(err error) ops.Answer`.
  - `func subject(paths []string) string`, `func (a *App) sources() []string`, `func (a *App) trash()`, `func (a *App) remove()`.

- [ ] **Step 1: Write the failing tests**

Создать `internal/app/ops_test.go`:

```go
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/ops"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// settle runs the calls posted by the running operation until it ends or
// shows a question dialog.
func settle(t *testing.T, a *App) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for a.op != nil {
		if _, ok := a.modals.Top().(*ui.Dialog); ok {
			return
		}
		select {
		case f := <-a.calls:
			f()
			a.Draw()
		case <-deadline:
			t.Fatal("operation did not finish")
		}
	}
}

// fakeTrash makes F8 delete with rm instead of filling the real Trash.
func fakeTrash(t *testing.T) {
	ops.TrashCmd = "/bin/rm"
	t.Cleanup(func() { ops.TrashCmd = "/usr/bin/trash" })
}

func TestSubject(t *testing.T) {
	if got := subject([]string{"/a/x.txt"}); got != `"x.txt"` {
		t.Fatalf("one: %s", got)
	}
	if got := subject([]string{"/a/x", "/a/y"}); got != "2 files" {
		t.Fatalf("two: %s", got)
	}
}

func TestF8MovesSelectedToTrash(t *testing.T) {
	fakeTrash(t)
	a, dir := newApp(t)
	for _, n := range []string{"a", "b", "c"} {
		must(t, os.WriteFile(filepath.Join(dir, n), nil, 0o644))
	}
	p := a.panels[0]
	must(t, p.Reload())
	p.Selected["a"], p.Selected["b"] = true, true
	press(a, tcell.KeyF8, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != "Move 2 files to Trash?" || d.Danger {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	if !a.modals.Empty() || len(p.Selected) != 0 {
		t.Fatalf("modals %d selected %v", a.modals.Len(), p.Selected)
	}
	if p.Focus("a") || p.Focus("b") || !p.Focus("c") {
		t.Fatal("wrong files deleted or panel not re-read")
	}
}

func TestF8OnUpDirDoesNothing(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Home() // ".."
	press(a, tcell.KeyF8, 0, 0)
	if !a.modals.Empty() {
		t.Fatalf("top %#v", a.modals.Top())
	}
}

func TestShiftF8AsksAboutNonEmptyDir(t *testing.T) {
	for _, k := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyF8, 0, tcell.ModShift),
		tcell.NewEventKey(tcell.KeyF20, 0, 0),
	} {
		a, dir := newApp(t)
		a.panels[0].Focus("sub")
		a.HandleEvent(k)
		a.Draw()
		d, ok := a.modals.Top().(*ui.Dialog)
		if !ok || d.Lines[0] != `Delete "sub" permanently?` || !d.Danger {
			t.Fatalf("%s: top %#v", k.Name(), a.modals.Top())
		}
		press(a, tcell.KeyEnter, 0, 0)
		settle(t, a)
		d, ok = a.modals.Top().(*ui.Dialog)
		if !ok || d.Lines[0] != "Directory sub is not empty." {
			t.Fatalf("%s: top %#v", k.Name(), a.modals.Top())
		}
		press(a, tcell.KeyRune, 'd', 0)
		settle(t, a)
		if !a.modals.Empty() {
			t.Fatalf("%s: top %#v", k.Name(), a.modals.Top())
		}
		if _, err := os.Stat(filepath.Join(dir, "sub")); !os.IsNotExist(err) {
			t.Fatalf("%s: sub still there: %v", k.Name(), err)
		}
	}
}

func TestShiftF8CancelInQuestionKeepsDir(t *testing.T) {
	a, dir := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF8, 0, tcell.ModShift)
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	press(a, tcell.KeyEscape, 0, 0) // Esc in the question = Cancel
	settle(t, a)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "f.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestOperationErrorAsksRetrySkipCancel(t *testing.T) {
	a, _ := newApp(t)
	a.run("Test", false, func(j *ops.Job) []string {
		if j.Fail(os.ErrPermission) == ops.Skip {
			return []string{"skipped"}
		}
		return nil
	})
	settle(t, a)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Error" || !d.Danger || strings.Join(d.Buttons, ",") != "Retry,Skip,Cancel" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, 's', 0)
	settle(t, a)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
}

func TestProgressAppearsAndCancels(t *testing.T) {
	a, _ := newApp(t)
	stopped := make(chan struct{})
	a.run("Test", true, func(j *ops.Job) []string {
		j.Progress(ops.Progress{File: "/x/big", FileDone: 1, FileSize: 4, Done: 1, Total: 2})
		for !j.Canceled() {
			time.Sleep(time.Millisecond)
		}
		close(stopped)
		return nil
	})
	v := a.modals.Top().(*ui.Progress)
	p := a.panels[0].Cursor
	press(a, tcell.KeyEscape, 0, 0) // hidden: ignored
	press(a, tcell.KeyDown, 0, 0)   // never reaches the panel
	if a.panels[0].Cursor != p {
		t.Fatal("key reached the panel")
	}
	deadline := time.After(5 * time.Second)
	for v.Hidden {
		select {
		case f := <-a.calls:
			f()
		case <-deadline:
			t.Fatal("progress window never showed")
		}
	}
	if v.File != "big" || len(v.Bars) != 2 || v.Bars[0] != 0.25 || v.Bars[1] != 0.5 {
		t.Fatalf("view %+v", v)
	}
	select {
	case <-stopped:
		t.Fatal("canceled by a key pressed while hidden")
	default:
	}
	a.Draw()
	press(a, tcell.KeyEscape, 0, 0)
	settle(t, a)
	<-stopped
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
}

func TestMenuDeleteItems(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF9, 0, 0)
	press(a, tcell.KeyRight, 0, 0) // Files
	for i := 0; i < 7; i++ {       // Help → … → Delete → Delete permanently
		press(a, tcell.KeyDown, 0, 0)
	}
	press(a, tcell.KeyEnter, 0, 0)
	if d, ok := a.modals.Top().(*ui.Dialog); !ok || !d.Danger || !strings.HasSuffix(d.Lines[0], "permanently?") {
		t.Fatalf("top %#v", a.modals.Top())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/`
Expected: FAIL — `a.calls undefined`, `a.op undefined`, `undefined: subject`, `a.run undefined`.

- [ ] **Step 3: Implement the runner**

Создать `internal/app/ops.go`:

```go
package app

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navoznov/terminal-commander/internal/ops"
	"github.com/navoznov/terminal-commander/internal/panel"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/ui"
)

// progressDelay is how long an operation runs before its window shows.
const progressDelay = 300 * time.Millisecond

// op is a file operation running in the background.
type op struct {
	job     *ops.Job
	view    *ui.Progress
	src     *panel.Panel
	timer   *time.Timer
	fileBar bool

	mu     sync.Mutex
	last   ops.Progress
	queued atomic.Bool // a progress update waits in a.calls
}

// post runs f on the UI goroutine.
func (a *App) post(f func()) { a.calls <- f }

// run starts work in the background under a progress window titled title,
// with a bar for the current file if fileBar. When work ends, the names it
// returns are unselected in the active panel and both panels are re-read.
func (a *App) run(title string, fileBar bool, work func(*ops.Job) []string) {
	o := &op{src: a.panels[a.active], fileBar: fileBar}
	bars := 1
	if fileBar {
		bars = 2
	}
	o.view = &ui.Progress{Title: title, Bars: make([]float64, bars), Hidden: true, Cancel: func() { o.job.Cancel() }}
	o.job = &ops.Job{
		Progress: func(p ops.Progress) {
			o.mu.Lock()
			o.last = p
			o.mu.Unlock()
			if o.queued.CompareAndSwap(false, true) {
				a.post(func() { o.queued.Store(false); a.showProgress(o) })
			}
		},
		Conflict: a.askConflict,
		NotEmpty: a.askNotEmpty,
		Fail:     a.askFail,
	}
	a.op = o
	a.modals.Push(o.view)
	o.timer = time.AfterFunc(progressDelay, func() { a.post(func() { o.view.Hidden = false }) })
	go func() {
		done := work(o.job)
		a.post(func() { a.finish(o, done) })
	}()
}

func (a *App) showProgress(o *op) {
	o.mu.Lock()
	p := o.last
	o.mu.Unlock()
	o.view.File = filepath.Base(p.File)
	total := fraction(p.Done, p.Total)
	if o.fileBar {
		o.view.Bars = []float64{fraction(p.FileDone, p.FileSize), total}
	} else {
		o.view.Bars = []float64{total}
	}
}

func fraction(n, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(n) / float64(total)
}

func (a *App) finish(o *op, done []string) {
	o.timer.Stop()
	a.modals.Remove(o.view)
	a.op = nil
	for _, n := range done {
		delete(o.src.Selected, n)
	}
	a.reloadPanels()
}

// ask shows d from the operation's goroutine and waits for the answer:
// answers[i] for button i, Cancel for Esc.
func (a *App) ask(d *ui.Dialog, answers []ops.Answer) ops.Answer {
	ch := make(chan ops.Answer, 1)
	d.Danger = true
	d.Done = func(b int, _ string) {
		if b < 0 {
			ch <- ops.Cancel
		} else {
			ch <- answers[b]
		}
	}
	a.post(func() { a.modals.Push(d) })
	return <-ch
}

func (a *App) askConflict(dst string) ops.Answer {
	return a.ask(&ui.Dialog{
		Title:   "Warning",
		Lines:   append([]string{"File already exists:"}, term.Wrap(dst, 60)...),
		Buttons: []string{"Overwrite", "Skip", "Overwrite all", "Skip all", "Cancel"},
	}, []ops.Answer{ops.Yes, ops.Skip, ops.All, ops.SkipAll, ops.Cancel})
}

func (a *App) askNotEmpty(dir string) ops.Answer {
	return a.ask(&ui.Dialog{
		Title:   "Delete",
		Lines:   []string{"Directory " + filepath.Base(dir) + " is not empty.", "Delete it?"},
		Buttons: []string{"Delete", "All", "Skip", "Cancel"},
	}, []ops.Answer{ops.Yes, ops.All, ops.Skip, ops.Cancel})
}

func (a *App) askFail(err error) ops.Answer {
	return a.ask(&ui.Dialog{
		Title:   "Error",
		Lines:   term.Wrap(err.Error(), 60),
		Buttons: []string{"Retry", "Skip", "Cancel"},
	}, []ops.Answer{ops.Yes, ops.Skip, ops.Cancel})
}
```

- [ ] **Step 4: Wire the channel into `App`**

В `internal/app/app.go`:

1. В структуру `App` после `modals ui.Stack` добавить:

```go
	calls      chan func() // work for the UI goroutine, sent by operations
	op         *op         // the running file operation, or nil
```

2. В `New` заменить `a := &App{screen: s, home: home}` на:

```go
	a := &App{screen: s, home: home, calls: make(chan func(), 16)}
```

3. Заменить `Run`:

```go
func (a *App) Run() {
	events := make(chan tcell.Event)
	go a.screen.ChannelEvents(events, nil)
	for !a.quit {
		a.Draw()
		a.screen.Show()
		select {
		case ev, ok := <-events:
			if !ok {
				return // screen finalized
			}
			a.HandleEvent(ev)
		case f := <-a.calls:
			f()
		}
	}
}
```

4. В `handleKey` после `case tcell.KeyF7:` добавить:

```go
	case tcell.KeyF8:
		if ev.Modifiers()&tcell.ModShift != 0 {
			a.remove()
		} else {
			a.trash()
		}
	case tcell.KeyF20: // Shift-F8 in terminals that send it as F20
		a.remove()
```

- [ ] **Step 5: Implement F8 and Shift-F8**

В `internal/app/files.go` добавить в импорты `"fmt"` и `"github.com/navoznov/terminal-commander/internal/ops"`, в конец файла:

```go
// sources returns the paths the active panel's operation works on.
func (a *App) sources() []string {
	p := a.panels[a.active]
	var paths []string
	for _, n := range p.Sources() {
		paths = append(paths, filepath.Join(p.Path, n))
	}
	return paths
}

// subject names the sources in a dialog: `"name"` or `N files`.
func subject(paths []string) string {
	if len(paths) == 1 {
		return `"` + filepath.Base(paths[0]) + `"`
	}
	return fmt.Sprintf("%d files", len(paths))
}

// trash asks and moves the sources to the Trash.
func (a *App) trash() {
	srcs := a.sources()
	if srcs == nil {
		return
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Delete",
		Lines:   []string{"Move " + subject(srcs) + " to Trash?"},
		Buttons: []string{"Delete", "Cancel"},
		Done: func(b int, _ string) {
			if b == 0 {
				a.run("Delete", false, func(j *ops.Job) []string { return ops.Trash(j, srcs) })
			}
		},
	})
}

// remove asks and deletes the sources permanently.
func (a *App) remove() {
	srcs := a.sources()
	if srcs == nil {
		return
	}
	a.modals.Push(&ui.Dialog{
		Title:   "Delete",
		Lines:   []string{"Delete " + subject(srcs) + " permanently?"},
		Buttons: []string{"Delete", "Cancel"},
		Danger:  true,
		Done: func(b int, _ string) {
			if b == 0 {
				a.run("Delete", false, func(j *ops.Job) []string { return ops.Remove(j, srcs) })
			}
		},
	})
}
```

В `internal/app/menu.go` заменить два пункта:

```go
			{Label: "Delete", Key: "F8", Action: a.trash},
			{Label: "Delete permanently", Key: "Shift-F8", Action: a.remove},
```

- [ ] **Step 6: Run tests**

Run: `go test -race ./internal/app/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
go vet ./... && go test -race ./...
git add internal/app
git commit -m "feat(app): run file operations in the background; F8 and Shift-F8 delete"
```

---

### Task 8: F5 / F6, меню, README, ручная проверка

**Files:**
- Modify: `internal/app/files.go` (`transfer`), `internal/app/app.go` (F5, F6), `internal/app/menu.go` (Copy, Rename/Move), `README.md`
- Test: `internal/app/ops_test.go`

**Interfaces:**
- Consumes: `a.sources`, `subject`, `resolve`, `a.run`, `ops.Copy`, `ops.Move`.
- Produces: `func (a *App) transfer(move bool)` — диалог F5 (`move == false`) или F6.

- [ ] **Step 1: Write the failing tests**

В конец `internal/app/ops_test.go`:

```go
// twoDirs puts the right panel into a new directory "dst" next to "sub".
func twoDirs(t *testing.T) (*App, string, string) {
	a, dir := newApp(t)
	dst := filepath.Join(dir, "dst")
	must(t, os.Mkdir(dst, 0o755))
	must(t, a.panels[1].Load(dst))
	must(t, a.panels[0].Reload())
	return a, dir, dst
}

func TestF5CopiesToOtherPanel(t *testing.T) {
	a, dir, dst := twoDirs(t)
	must(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644))
	must(t, a.panels[0].Reload())
	a.panels[0].Focus("a.txt")
	press(a, tcell.KeyF5, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Copy" || d.Lines[0] != `Copy "a.txt" to:` || d.Input.Text != dst || d.Buttons[0] != "Copy" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	if b, err := os.ReadFile(filepath.Join(dst, "a.txt")); err != nil || string(b) != "a" {
		t.Fatalf("copy: %q %v", b, err)
	}
	if !a.modals.Empty() || !a.panels[1].Focus("a.txt") {
		t.Fatal("dialog left open or other panel not re-read")
	}
}

func TestF6RenamesRelativeToPanel(t *testing.T) {
	a, dir := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF6, 0, 0)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Lines[0] != `Rename or move "sub" to:` || d.Buttons[0] != "Move" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	typeText(a, "renamed")
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	if _, err := os.Stat(filepath.Join(dir, "renamed", "f.txt")); err != nil {
		t.Fatal(err)
	}
	if a.panels[0].Focus("sub") {
		t.Fatal("panel not re-read")
	}
}

func TestCopyConflictDialogSkip(t *testing.T) {
	a, dir, dst := twoDirs(t)
	for _, n := range []string{"a", "b"} {
		must(t, os.WriteFile(filepath.Join(dir, n), []byte("new"), 0o644))
	}
	must(t, os.WriteFile(filepath.Join(dst, "a"), []byte("old"), 0o644))
	p := a.panels[0]
	must(t, p.Reload())
	p.Selected["a"], p.Selected["b"] = true, true
	press(a, tcell.KeyF5, 0, 0)
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Warning" || strings.Join(d.Buttons, ",") != "Overwrite,Skip,Overwrite all,Skip all,Cancel" {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, 's', 0)
	settle(t, a)
	if b, _ := os.ReadFile(filepath.Join(dst, "a")); string(b) != "old" {
		t.Fatal("skipped file overwritten")
	}
	if !p.Selected["a"] || p.Selected["b"] {
		t.Fatalf("selected %v: skipped must stay, copied must go", p.Selected)
	}
}

func TestCopyIntoItselfShowsError(t *testing.T) {
	a, _ := newApp(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF5, 0, 0)
	typeText(a, "sub/in")
	press(a, tcell.KeyEnter, 0, 0)
	settle(t, a)
	d, ok := a.modals.Top().(*ui.Dialog)
	if !ok || d.Title != "Error" || !strings.Contains(strings.Join(d.Lines, " "), "into itself") {
		t.Fatalf("top %#v", a.modals.Top())
	}
	press(a, tcell.KeyRune, 'c', 0)
	settle(t, a)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
}

func TestF5CancelDoesNothing(t *testing.T) {
	a, _, dst := twoDirs(t)
	a.panels[0].Focus("sub")
	press(a, tcell.KeyF5, 0, 0)
	press(a, tcell.KeyEscape, 0, 0)
	if a.op != nil || !a.modals.Empty() {
		t.Fatalf("op %v modals %d", a.op, a.modals.Len())
	}
	if _, err := os.Stat(filepath.Join(dst, "sub")); !os.IsNotExist(err) {
		t.Fatal("copied after Cancel")
	}
}

func TestMenuCopyAndMove(t *testing.T) {
	for i, title := range []string{"Copy", "Rename/Move"} {
		a, _ := newApp(t)
		a.panels[0].Focus("sub")
		press(a, tcell.KeyF9, 0, 0)
		press(a, tcell.KeyRight, 0, 0) // Files
		for k := 0; k < 3+i; k++ {     // Help → View → Edit → Copy (→ Rename/Move)
			press(a, tcell.KeyDown, 0, 0)
		}
		press(a, tcell.KeyEnter, 0, 0)
		if d, ok := a.modals.Top().(*ui.Dialog); !ok || d.Title != title {
			t.Fatalf("%s: top %#v", title, a.modals.Top())
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/`
Expected: FAIL — F5 / F6 не открывают диалог (`top <nil>`), пункты меню показывают «Not implemented yet».

- [ ] **Step 3: Implement**

В конец `internal/app/files.go`:

```go
// transfer asks for a destination and copies (F5) or moves (F6) the
// sources there; the other panel's directory is the default.
func (a *App) transfer(move bool) {
	srcs := a.sources()
	if srcs == nil {
		return
	}
	title, prompt, button, do := "Copy", "Copy ", "Copy", ops.Copy
	if move {
		title, prompt, button, do = "Rename/Move", "Rename or move ", "Move", ops.Move
	}
	base := a.panels[a.active].Path
	a.modals.Push(&ui.Dialog{
		Title:   title,
		Lines:   []string{prompt + subject(srcs) + " to:"},
		Input:   ui.NewInput(a.panels[1-a.active].Path),
		Buttons: []string{button, "Cancel"},
		Done: func(b int, text string) {
			if b != 0 || text == "" {
				return
			}
			dst := resolve(base, a.home, text)
			a.run(title, true, func(j *ops.Job) []string { return do(j, srcs, dst) })
		},
	})
}
```

В `internal/app/app.go`, в `handleKey`, перед `case tcell.KeyF7:`:

```go
	case tcell.KeyF5:
		a.transfer(false)
	case tcell.KeyF6:
		a.transfer(true)
```

В `internal/app/menu.go` заменить два пункта:

```go
			{Label: "Copy", Key: "F5", Action: func() { a.transfer(false) }},
			{Label: "Rename/Move", Key: "F6", Action: func() { a.transfer(true) }},
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./...`
Expected: PASS.

- [ ] **Step 5: Update README**

В `README.md`:

1. Заменить абзац под `> [!NOTE]` на:

```markdown
> **Work in progress.** Stages 1–4 of 6 are done: the panels, dialogs, the F9 menu, drive selection, help and file operations (copy, move, make directory, delete to the Trash or permanently). The command line, the viewer and the editor aren't built yet. See [Roadmap](#roadmap).
```

2. В таблицу «Keys» перед строкой `| Esc then 1…0 | F1…F10 |` добавить:

```markdown
| + / - / * | Select or unselect by mask, invert selection |
| Alt-F1 / Alt-F2 (Esc F1 / Esc F2) | Choose the left / right drive |
| F1 | Help |
| F2, F9 | Menu |
| F5 | Copy the selected files, or the file under the cursor, to the other panel |
| F6 | Rename or move |
| F7 | Make a directory |
| F8 | Move to the Trash |
| Shift-F8 | Delete permanently |
```

3. В таблице «Roadmap» поставить `✅` этапам 3 и 4.

- [ ] **Step 6: Manual check**

```bash
make build
mkdir -p /tmp/tc-manual/{left,right} && cd /tmp/tc-manual && dd if=/dev/zero of=left/big bs=1m count=2000 && echo hi > left/small.txt && mkdir -p left/dir/sub && echo x > left/dir/sub/x
tmux new-session -d -s tc -x 100 -y 30 "$OLDPWD/tc /tmp/tc-manual/left" && tmux send-keys -t tc Tab && sleep 0.3
```

Через `tmux send-keys` / `tmux capture-pane -p -t tc` проверить (правую панель перевести в `/tmp/tc-manual/right`: Esc F2 недостаточно — перейти через `..` и Enter на `right`):

- F5 на `big` → диалог `Copy "big" to:` с путём правой панели → Enter → через ~0.3 с окно прогресса с двумя полосами → `Esc` → окно закрылось, в `right` нет `big`.
- F5 на `small.txt` дважды → второй раз диалог «Warning» с пятью кнопками → `o` → перезаписан.
- F6 на `small.txt`, ввести `renamed.txt`, Enter → переименован в левой панели.
- F7 `a/b` → курсор на `a`.
- F8 на `a` → `Move "a" to Trash?` → Enter → папка в Корзине Finder.
- Shift-F8 на `dir` → красный диалог → Enter → «Directory dir is not empty.» → `d` → удалена.

Закрыть: `tmux kill-session -t tc && rm -rf /tmp/tc-manual`.

- [ ] **Step 7: Commit**

```bash
go vet ./... && go test -race ./...
git add internal/app README.md
git commit -m "feat(app): F5 copies and F6 moves files; update README"
```
