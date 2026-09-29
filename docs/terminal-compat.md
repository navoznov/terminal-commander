# Совместимость клавиш с терминалами

Проверено `tc --keytest` 2026-09-28.

## Встроенный терминал Orca

| Клавиша | Результат |
|---|---|
| F1…F10 | работают |
| Shift-F8 | работает |
| `Esc` затем цифра | F1…F10 |
| `Esc` затем `.` (или `ю` в русской раскладке) | Alt-. — скрытые файлы |
| `Esc` затем F1 / F2 | Alt-F1 / Alt-F2 |
| Option-F1, Option-F2, Option-. | не доходят: Option не отправляется как Meta |
| Control-1, Control-2 | не доходят (терминал не передаёт Control+цифра) |
| Control-Fn-↑ (Control-PgUp) | не доходит — скорее всего, его перехватывает сама Orca («предыдущая вкладка»); проверить в iTerm2 |

Выводы:
- для Option-сочетаний используйте `Esc`-префикс или включите «Option как Meta» в терминале
  (iTerm2: Profiles → Keys → Left Option → `Esc+`; Terminal.app: «Use Option as Meta key»;
  VS Code: `terminal.integrated.macOptionIsMeta: true`);
- режим панели переключается Control-T или через меню F9 → Left/Right → Brief/Full;
- на уровень выше — Backspace или Enter на `..`.
- Control-Enter доходит отдельно от Enter только в терминалах с протоколом
  kitty / CSI-u (tcell включает его сам, если терминал умеет); вставить имя
  в командную строку везде можно через Control-J или `Esc` `Enter`.

## iTerm2, VS Code, JetBrains, Terminal.app

Не проверялись.
