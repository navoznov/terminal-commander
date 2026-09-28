# Совместимость клавиш с терминалами

Проверено `tc --keytest` 2026-09-28.

## Встроенный терминал Orca

| Клавиша | Результат |
|---|---|
| F1…F10 | работают |
| Shift-F8 | работает |
| `Esc` затем цифра | F1…F10 |
| `Esc` затем `.` | Alt-. |
| `Esc` затем F1 / F2 | Alt-F1 / Alt-F2 |
| Option-F1, Option-F2, Option-. | не доходят: Option не отправляется как Meta |
| Control-1, Control-2 | не доходят (терминал не передаёт Control+цифра) |
| Control-Fn-↑ (Control-PgUp) | не доходит (Control-↑ перехватывает Mission Control) |

Выводы:
- для Option-сочетаний используйте `Esc`-префикс или включите «Option как Meta» в терминале;
- режим панели переключается Control-T (и через меню F9 на этапе 3);
- на уровень выше — Backspace или Enter на `..`.

## iTerm2, VS Code, JetBrains, Terminal.app

Не проверялись.
