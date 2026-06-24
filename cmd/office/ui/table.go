package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/mattn/go-runewidth"
)

type tableStyles struct {
	title      lipgloss.Style
	header     lipgloss.Style
	headerSort lipgloss.Style
	cell       lipgloss.Style
	rowActive  lipgloss.Style
	colActive  lipgloss.Style
	cellActive lipgloss.Style
	rowSelect  lipgloss.Style
	cellSelect lipgloss.Style
	help       lipgloss.Style
}

func defaultTableStyles() tableStyles {
	return tableStyles{
		title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
		header:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("236")).Padding(0, 1),
		headerSort: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Padding(0, 1),
		cell:       lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1),
		rowActive:  lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("238")),
		colActive:  lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("237")),
		cellActive: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62")),
		rowSelect:  lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("22")),
		cellSelect: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("34")),
		help:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	}
}

// DataTable is a spreadsheet-style listContent table for the list pane.
type DataTable struct {
	spec      model.ListSpec
	columns   []model.TableColumn
	items     []model.Item
	order     []int
	cursorRow int
	cursorCol int
	colScroll int
	sortCol   int
	sortAsc   bool
	focused   bool
	ready     bool
	width     int
	height    int
	viewport  viewport.Model
	styles    tableStyles
}

func newDataTable() DataTable {
	t := DataTable{
		sortCol:  -1,
		styles:   defaultTableStyles(),
		viewport: viewport.New(40, 10),
	}
	t.viewport.MouseWheelEnabled = true
	return t
}

func (t *DataTable) SetSize(w, h int) {
	if w < 8 {
		w = 8
	}
	if h < 3 {
		h = 3
	}
	t.width = w
	t.height = h
	t.viewport.Width = w
	t.viewport.Height = h - 2 // title + header
	if t.viewport.Height < 1 {
		t.viewport.Height = 1
	}
	t.refreshViewport()
}

func (t *DataTable) SetFocused(f bool) {
	t.focused = f
	t.refreshViewport()
}

func (t *DataTable) SetData(spec model.ListSpec, items []model.Item) {
	t.spec = spec
	t.items = items
	t.columns = model.BuildColumns(spec.Subject, items)
	t.order = make([]int, len(items))
	for i := range t.order {
		t.order[i] = i
	}
	t.cursorRow = 0
	t.cursorCol = 0
	t.colScroll = 0
	t.ready = true
	if t.sortCol >= 0 {
		t.applySort()
	}
	t.clampCursor()
	t.refreshViewport()
}

func (t *DataTable) Clear() {
	t.ready = false
	t.items = nil
	t.order = nil
	t.columns = nil
	t.cursorRow = 0
	t.cursorCol = 0
	t.viewport.SetContent("")
}

func (t *DataTable) ItemIndex() int {
	if t.cursorRow < 0 || t.cursorRow >= len(t.order) {
		return -1
	}
	return t.order[t.cursorRow]
}

func (t *DataTable) MoveRow(delta int) {
	if !t.ready || len(t.order) == 0 {
		return
	}
	prev := t.cursorRow
	t.cursorRow = clampInt(t.cursorRow+delta, 0, len(t.order)-1)
	t.syncRowScroll(prev)
	t.refreshViewport()
}

func (t *DataTable) MoveCol(delta int) {
	if !t.ready || len(t.columns) == 0 {
		return
	}
	t.cursorCol = clampInt(t.cursorCol+delta, 0, len(t.columns)-1)
	t.ensureColVisible()
	t.refreshViewport()
}

func (t *DataTable) ToggleSort() {
	if !t.ready || len(t.columns) == 0 || len(t.order) == 0 {
		return
	}
	if t.sortCol == t.cursorCol {
		t.sortAsc = !t.sortAsc
	} else {
		t.sortCol = t.cursorCol
		t.sortAsc = true
	}
	t.applySort()
	t.refreshViewport()
}

func (t *DataTable) applySort() {
	if t.sortCol < 0 || t.sortCol >= len(t.columns) {
		return
	}
	key := t.columns[t.sortCol].Key
	sort.SliceStable(t.order, func(i, j int) bool {
		a := model.CellText(t.items[t.order[i]], key)
		b := model.CellText(t.items[t.order[j]], key)
		if t.sortAsc {
			return strings.ToLower(a) < strings.ToLower(b)
		}
		return strings.ToLower(a) > strings.ToLower(b)
	})
}

func (t *DataTable) PageScroll(delta int) {
	if delta > 0 {
		t.viewport.ViewDown()
	} else {
		t.viewport.ViewUp()
	}
}

func (t *DataTable) Update(msg tea.Msg) tea.Cmd {
	if !t.focused || !t.ready {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "pgdown", "pgdn", "f", "ctrl+d":
			t.viewport.ViewDown()
		case "pgup", "b", "ctrl+u":
			t.viewport.ViewUp()
		}
	case tea.MouseMsg:
		var cmd tea.Cmd
		t.viewport, cmd = t.viewport.Update(msg)
		return cmd
	}
	return nil
}

func (t DataTable) View() string {
	if !t.ready {
		return "List\n\nSelect a leaf node in the tree\n(marked with •) and press Enter.\n"
	}
	title := t.styles.title.Render(fmt.Sprintf("%s (%d)", t.spec.Subject, len(t.items)))
	sortHint := ""
	if t.sortCol >= 0 && t.sortCol < len(t.columns) {
		dir := "▲"
		if !t.sortAsc {
			dir = "▼"
		}
		sortHint = t.styles.help.Render(fmt.Sprintf("  sort: %s %s", t.columns[t.sortCol].Title, dir))
	}
	header := t.renderHeader()
	body := t.viewport.View()
	content := lipgloss.JoinVertical(lipgloss.Left, title+sortHint, header, body)
	return lipgloss.NewStyle().Width(t.width).Render(content)
}

func (t *DataTable) visibleLayout() (indices []int, widths map[int]int) {
	widths = make(map[int]int)
	if len(t.columns) == 0 || t.width <= 0 {
		return nil, widths
	}
	indices = t.pickVisibleColumnIndices()
	if len(indices) == 0 {
		return indices, widths
	}
	minSum := 0
	for _, i := range indices {
		minSum += t.columns[i].Width
	}
	widths = distributeColumnWidths(minSum, t.width, indices, t.columns)
	return indices, widths
}

func (t *DataTable) pickVisibleColumnIndices() []int {
	var out []int
	used := 0
	for colIdx := t.colScroll; colIdx < len(t.columns); colIdx++ {
		w := t.columns[colIdx].Width
		if len(out) > 0 && used+w > t.width {
			break
		}
		out = append(out, colIdx)
		used += w
	}
	if len(out) == 0 {
		colIdx := t.colScroll
		if colIdx < 0 || colIdx >= len(t.columns) {
			colIdx = 0
		}
		out = []int{colIdx}
	}
	return out
}

// distributeColumnWidths expands or shrinks visible columns to exactly fill total width.
func distributeColumnWidths(minSum, total int, indices []int, cols []model.TableColumn) map[int]int {
	out := make(map[int]int, len(indices))
	if len(indices) == 0 {
		return out
	}
	if total < len(indices) {
		total = len(indices)
	}
	if minSum <= 0 {
		each := total / len(indices)
		if each < 1 {
			each = 1
		}
		for _, i := range indices {
			out[i] = each
		}
		fixColumnWidthSum(out, indices, total)
		return out
	}
	for _, i := range indices {
		out[i] = cols[i].Width
	}
	if minSum >= total {
		for _, i := range indices {
			out[i] = cols[i].Width * total / minSum
			if out[i] < 1 {
				out[i] = 1
			}
		}
		fixColumnWidthSum(out, indices, total)
		return out
	}
	extra := total - minSum
	flex := make([]int, 0, len(indices))
	for _, i := range indices {
		switch cols[i].Key {
		case "title", "subtitle", "description", "displayName", "primaryEmail", "from", "to", "tasks":
			flex = append(flex, i)
		}
	}
	if len(flex) == 0 {
		flex = append(flex, indices...)
	}
	flexSum := 0
	for _, i := range flex {
		flexSum += cols[i].Width
	}
	if flexSum <= 0 {
		flexSum = len(flex)
	}
	for _, i := range flex {
		out[i] += extra * cols[i].Width / flexSum
	}
	fixColumnWidthSum(out, indices, total)
	return out
}

func fixColumnWidthSum(widths map[int]int, indices []int, total int) {
	if len(indices) == 0 {
		return
	}
	sum := 0
	for _, i := range indices {
		sum += widths[i]
	}
	widths[indices[len(indices)-1]] += total - sum
	if widths[indices[len(indices)-1]] < 1 {
		widths[indices[len(indices)-1]] = 1
	}
}

func (t *DataTable) renderHeader() string {
	indices, widths := t.visibleLayout()
	cells := make([]string, 0, len(indices))
	for _, colIdx := range indices {
		col := t.columns[colIdx]
		w := widths[colIdx]
		title := col.Title
		if t.sortCol == colIdx {
			if t.sortAsc {
				title += " ▲"
			} else {
				title += " ▼"
			}
		}
		text := runewidth.Truncate(title, w, "…")
		style := t.styles.header
		if t.focused && colIdx == t.cursorCol {
			style = t.styles.headerSort
		}
		cells = append(cells, style.Width(w).MaxWidth(w).Render(text))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
}

func (t *DataTable) refreshViewport() {
	if !t.ready {
		return
	}
	rows := make([]string, 0, len(t.order))
	for row := 0; row < len(t.order); row++ {
		rows = append(rows, t.renderRow(row))
	}
	t.viewport.SetContent(strings.Join(rows, "\n"))
	t.syncRowScroll(t.cursorRow)
}

func (t *DataTable) renderRow(row int) string {
	item := t.items[t.order[row]]
	selected := item.Selected
	indices, widths := t.visibleLayout()
	cells := make([]string, 0, len(indices))
	for _, colIdx := range indices {
		col := t.columns[colIdx]
		w := widths[colIdx]
		text := runewidth.Truncate(model.CellText(item, col.Key), w, "…")
		cells = append(cells, t.styleCell(row, colIdx, selected, text, w))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
}

func (t *DataTable) styleCell(row, col int, selected bool, text string, width int) string {
	base := t.styles.cell.Width(width).MaxWidth(width)
	isRow := row == t.cursorRow
	isCol := col == t.cursorCol
	isCell := t.focused && isRow && isCol

	switch {
	case selected && isCell:
		return t.styles.cellSelect.Width(width).MaxWidth(width).Render(text)
	case selected && isRow:
		return t.styles.rowSelect.Width(width).MaxWidth(width).Render(text)
	case selected:
		return t.styles.rowSelect.Width(width).MaxWidth(width).Render(text)
	case isCell:
		return t.styles.cellActive.Width(width).MaxWidth(width).Render(text)
	case t.focused && isRow:
		return t.styles.rowActive.Width(width).MaxWidth(width).Render(text)
	case t.focused && isCol:
		return t.styles.colActive.Width(width).MaxWidth(width).Render(text)
	default:
		return base.Render(text)
	}
}

func (t *DataTable) ensureColVisible() {
	vis := t.pickVisibleColumnIndices()
	if len(vis) == 0 {
		return
	}
	first, last := vis[0], vis[len(vis)-1]
	if t.cursorCol < first {
		t.colScroll = t.cursorCol
	}
	if t.cursorCol > last {
		t.colScroll = t.cursorCol
		for t.cursorCol >= 0 {
			vis = t.pickVisibleColumnIndices()
			if len(vis) == 0 {
				break
			}
			last = vis[len(vis)-1]
			if t.cursorCol <= last {
				break
			}
			if t.colScroll < len(t.columns)-1 {
				t.colScroll++
			} else {
				break
			}
		}
	}
}

func (t *DataTable) syncRowScroll(prev int) {
	if t.cursorRow < t.viewport.YOffset {
		t.viewport.YOffset = t.cursorRow
	} else if t.cursorRow >= t.viewport.YOffset+t.viewport.Height {
		t.viewport.YOffset = t.cursorRow - t.viewport.Height + 1
	}
	if t.viewport.YOffset < 0 {
		t.viewport.YOffset = 0
	}
	maxOff := len(t.order) - t.viewport.Height
	if maxOff < 0 {
		maxOff = 0
	}
	if t.viewport.YOffset > maxOff {
		t.viewport.YOffset = maxOff
	}
	_ = prev
}

func (t *DataTable) clampCursor() {
	if len(t.order) == 0 {
		t.cursorRow = 0
	} else {
		t.cursorRow = clampInt(t.cursorRow, 0, len(t.order)-1)
	}
	if len(t.columns) == 0 {
		t.cursorCol = 0
	} else {
		t.cursorCol = clampInt(t.cursorCol, 0, len(t.columns)-1)
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// tableKeyMatches helps avoid importing key in app.go for sort-only bindings.
func (t *DataTable) UpdateItems(items []model.Item) {
	t.items = items
	t.clampCursor()
	t.refreshViewport()
}
