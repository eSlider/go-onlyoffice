package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
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
	loadingMore bool
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
	t.viewport.Height = h - 1 // column header
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

// CursorRow is the visual row index in the current table order.
func (t DataTable) CursorRow() int {
	return t.cursorRow
}

// Items returns the backing slice (after appends).
func (t DataTable) Items() []model.Item {
	return t.items
}

// NearEnd reports whether the cursor is within threshold rows of the last row.
func (t DataTable) NearEnd(threshold int) bool {
	if !t.ready || len(t.order) == 0 {
		return false
	}
	return t.cursorRow >= len(t.order)-threshold
}

func (t *DataTable) SetLoadingMore(on bool) {
	t.loadingMore = on
}

// AppendItems adds new rows, skipping duplicate IDs. Returns how many were added.
func (t *DataTable) AppendItems(items []model.Item) int {
	if !t.ready || len(items) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(t.items))
	for _, it := range t.items {
		if it.ID != "" {
			seen[it.ID] = struct{}{}
		}
	}
	added := 0
	for _, it := range items {
		if it.ID != "" {
			if _, ok := seen[it.ID]; ok {
				continue
			}
			seen[it.ID] = struct{}{}
		}
		idx := len(t.items)
		t.items = append(t.items, it)
		t.order = append(t.order, idx)
		added++
	}
	if added > 0 {
		if t.sortCol >= 0 {
			t.applySort()
		}
		t.clampCursor()
		t.refreshViewport()
	}
	return added
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
		case "pgdown", "pgdn", "ctrl+d":
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

func (t DataTable) LoadingMore() bool { return t.loadingMore }

func (t DataTable) View() string {
	if !t.ready {
		return "\nSelect a leaf in the tree (•) and press Enter.\n"
	}
	header := t.renderHeader()
	if t.hasVerticalScrollbar() {
		header = padANSIWidth(header, t.lineContentWidth())
	}
	body := ApplyVerticalScrollbar(
		t.viewport.View(),
		t.width,
		t.viewport.Height,
		t.viewport.TotalLineCount(),
		t.viewport.YOffset,
	)
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func (t DataTable) SortHint() string {
	if t.sortCol < 0 || t.sortCol >= len(t.columns) {
		return ""
	}
	dir := "▲"
	if !t.sortAsc {
		dir = "▼"
	}
	return fmt.Sprintf("sort: %s %s", t.columns[t.sortCol].Title, dir)
}

func (t DataTable) SubjectLabel() string {
	if !t.ready {
		return "List"
	}
	return string(t.spec.Subject)
}

func (t DataTable) ItemCount() int {
	return len(t.items)
}

func (t DataTable) hasVerticalScrollbar() bool {
	return t.ready && t.viewport.TotalLineCount() > t.viewport.Height
}

func (t DataTable) lineContentWidth() int {
	w := t.width
	if t.hasVerticalScrollbar() && w > 1 {
		w--
	}
	return w
}

func (t *DataTable) visibleLayout() (indices []int, widths map[int]int) {
	lay := t.computeLayout()
	return lay.indices, lay.widths
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
		text := truncateCellText(title, w)
		style := t.styles.header
		if t.focused && colIdx == t.cursorCol {
			style = t.styles.headerSort
		}
		cells = append(cells, renderTableCell(style, text, w))
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
		raw := strings.TrimSpace(model.CellText(item, col.Key))
		text := truncateCellText(raw, w)
		cells = append(cells, t.styleCell(row, colIdx, col.Key, selected, item, text, w))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, cells...)
}

func (t *DataTable) styleCell(row, col int, colKey string, selected bool, item model.Item, text string, width int) string {
	if t.spec.Subject == model.SubjectProjects {
		style := t.projectCellStyle(row, col, colKey, selected, item)
		return renderTableCell(style, text, width)
	}

	isRow := row == t.cursorRow
	isCol := col == t.cursorCol
	isCell := t.focused && isRow && isCol

	var base lipgloss.Style
	switch {
	case selected && isCell:
		base = t.styles.cellSelect
	case selected && isRow:
		base = t.styles.rowSelect
	case selected:
		base = t.styles.rowSelect
	case isCell:
		base = t.styles.cellActive
	case t.focused && isRow:
		base = t.styles.rowActive
	case t.focused && isCol:
		base = t.styles.colActive
	default:
		base = t.styles.cell
	}
	return renderTableCell(base, text, width)
}

func (t *DataTable) ensureColVisible() {
	if t.spec.Subject == model.SubjectProjects || t.spec.Subject == model.SubjectUsers {
		t.colScroll = 0
		return
	}
	contentW := t.lineContentWidth()
	vis := pickVisibleColumnIndices(t.columns, t.colScroll, contentW)
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
			vis = pickVisibleColumnIndices(t.columns, t.colScroll, contentW)
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
