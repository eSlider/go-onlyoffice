---
name: office-tui-table
description: >-
  Build and fix Bubble Tea list tables in cmd/office (DataTable, column layout,
  lipgloss cell rendering, pane width). Use when changing office center-pane
  tables, column widths, ANSI styling, or list toolbar alignment.
---
# office TUI table (`cmd/office/ui`)

## Architecture

| Piece | File | Role |
|-------|------|------|
| `DataTable` | `table.go` | cursor, sort, viewport, header/row render |
| Column layout | `table_layout.go` | flex vs horizontal-scroll policies |
| Flex policy | `model/table_layout.go` | per-subject flex column key + min width |
| Cell render | `table_render.go` | bubbles/table pattern + ANSI-safe padding |
| Subject row style | `table_projects.go` | optional per-subject cell styles |
| Pane sizing | `scroll.go` | `paneContentWidth`, `paneLipglossWidth` |
| Scrollbar pad | `scrollbar.go` | `padANSIWidth` for styled lines |

## Layout policies

1. **Flex (all columns visible)** — register in `model.TableFlexLayoutFor`:
   - fixed columns keep `TableColumn.Width`
   - one `FlexColumnKey` (e.g. `title`, `email`) absorbs remainder
   - `layoutFlexTable` shrinks fixed cols down to `minFixedColumnWidth` before flex min

2. **Scrolling (default)** — `layoutScrollingTable`:
   - horizontal `colScroll`; subset of columns visible
   - `distributeColumnWidths` fills pane width

Add a new flex subject: one entry in `TableFlexLayoutFor` only — do **not** copy `layoutProjectTable`/`layoutUserTable` per subject.

## Cell rendering (required pattern)

Never `style.Width(w).Padding(0,1).Render(text)` on styled cells — padding doubles display width and breaks `JoinHorizontal`.

Use `renderTableCell` from `table_render.go` (matches `charmbracelet/bubbles/table`):

```go
renderTableCell(outerStyle, plainText, columnWidth)
```

- truncate **plain** text with `runewidth.Truncate` inside `renderTableCell`
- inner box: `Width(innerW).Inline(true)` where `innerW = width - outer.GetHorizontalFrameSize()`
- outer style carries padding, background, foreground

## ANSI rules

- **Never** `runewidth.Truncate` / `padDisplayWidth` on strings that already contain lipgloss ANSI codes — corrupts escapes and shows garbage like lone `ID`.
- Use `padANSIWidth` (`charmbracelet/x/ansi`) for styled lines (headers, toolbar, scrollbar rows).
- Do not wrap an already full-width styled header in another `lipgloss.Width()` — pad once.

## Pane width

Lipgloss bordered panes render **2 cells wider** than `Style.Width`:

- `paneLipglossWidth(rendered) = rendered - 2` for `.Width()` on pane style
- `paneContentWidth(rendered) = rendered - 4` for table/viewport inner size (border + padding)

Pass `paneContentWidth(pw.List)` to `DataTable.SetSize` and list toolbar.

## Tests to add when touching tables

- `layoutFlexTable` width sum equals pane width
- `TestProjectTableHeaderFitsPaneWidth` / per-line width at 68, 40, 30
- `TestPadANSIWidthPreservesStyledLine`
- `TestThreePaneRenderedWidthMatchesTerminal` top border + table lines ≤ inner width

## Anti-patterns

- Per-subject duplicate layout files (`table_foo.go` with copy-pasted shrink loops)
- Skipping truncation on selected rows (causes multi-line rows in viewport)
- Using `bubbles/table` widget directly — office needs column cursor, multi-select, subject-specific row styles
