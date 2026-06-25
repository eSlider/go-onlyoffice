package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

func sampleItems() []model.Item {
	return []model.Item{
		{ID: "2", Title: "Bravo", Subtitle: "b", Kind: model.KindTask, Raw: map[string]any{"status": "Open"}},
		{ID: "1", Title: "Alpha", Subtitle: "a", Kind: model.KindTask, Raw: map[string]any{"status": "Done"}},
	}
}

func TestDataTableSortByTitle(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(80, 12)
	tbl.SetData(model.ListSpec{Subject: model.SubjectTasks}, sampleItems())
	tbl.cursorCol = indexOfColumn(tbl, "title")
	tbl.ToggleSort()
	if model.CellText(tbl.items[tbl.order[0]], "title") != "Alpha" {
		t.Fatalf("expected Alpha first after sort, order=%v", tbl.order)
	}
}

func TestDataTableMoveColScrolls(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(24, 10)
	items := make([]model.Item, 1)
	items[0] = model.Item{ID: "1", Title: "One", Raw: map[string]any{
		"alpha": "a", "beta": "b", "gamma": "c", "delta": "d",
	}}
	tbl.SetData(model.ListSpec{Subject: model.SubjectContacts}, items)
	last := len(tbl.columns) - 1
	tbl.cursorCol = last
	tbl.ensureColVisible()
	if tbl.colScroll == 0 && last > 2 {
		t.Fatalf("expected horizontal scroll for wide table, colScroll=%d", tbl.colScroll)
	}
}

func TestDataTableViewHighlightsActiveCell(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(80, 12)
	tbl.SetFocused(true)
	tbl.SetData(model.ListSpec{Subject: model.SubjectTasks}, sampleItems())
	tbl.cursorRow = 1
	tbl.cursorCol = 2
	view := tbl.View()
	if !strings.Contains(view, "Alpha") {
		t.Fatal("view missing row data")
	}
	if !strings.Contains(view, "Title") {
		t.Fatal("view missing header")
	}
}

func TestDataTableFillsPaneWidth(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(80, 12)
	tbl.SetData(model.ListSpec{Subject: model.SubjectTasks}, sampleItems())
	indices, widths := tbl.visibleLayout()
	sum := 0
	for _, i := range indices {
		sum += widths[i]
	}
	if sum != 80 {
		t.Fatalf("visible columns width sum=%d want 80", sum)
	}
}

func TestDistributeColumnWidthsFillsTotal(t *testing.T) {
	cols := []model.TableColumn{
		{Key: "_sel", Width: 3},
		{Key: "id", Width: 10},
		{Key: "title", Width: 20},
	}
	indices := []int{0, 1, 2}
	w := distributeColumnWidths(33, 80, indices, cols)
	sum := w[0] + w[1] + w[2]
	if sum != 80 {
		t.Fatalf("sum=%d want 80", sum)
	}
	if w[2] <= 20 {
		t.Fatalf("title should expand, got %d", w[2])
	}
}

func TestProjectTableTitleAbsorbsWidth(t *testing.T) {
	cols := model.BuildColumns(model.SubjectProjects, nil)
	lay := layoutProjectTable(cols, 120)
	if len(lay.indices) != len(cols) {
		t.Fatalf("expected all %d columns visible, got %d", len(cols), len(lay.indices))
	}
	sum := 0
	titleW := 0
	fixedSum := 0
	titleIdx := indexOfColumnKey(cols, "title")
	for _, i := range lay.indices {
		sum += lay.widths[i]
		if i == titleIdx {
			titleW = lay.widths[i]
			continue
		}
		fixedSum += lay.widths[i]
	}
	if sum != 120 {
		t.Fatalf("sum=%d want 120", sum)
	}
	if titleW != 120-fixedSum {
		t.Fatalf("title width=%d want remainder %d", titleW, 120-fixedSum)
	}
	if titleW < 40 {
		t.Fatalf("title should absorb extra width, got %d", titleW)
	}
}

func TestProjectTableFillsPaneWidth(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(80, 12)
	tbl.SetData(model.ListSpec{Subject: model.SubjectProjects}, []model.Item{
		{ID: "1", Title: "Alpha", Kind: model.KindProject, Raw: map[string]any{"status": 0}},
	})
	indices, widths := tbl.visibleLayout()
	if len(indices) != len(tbl.columns) {
		t.Fatalf("expected all columns visible, got %d/%d", len(indices), len(tbl.columns))
	}
	sum := 0
	for _, i := range indices {
		sum += widths[i]
	}
	if sum != 80 {
		t.Fatalf("visible columns width sum=%d want 80", sum)
	}
}

func TestProjectRowRenderWidth(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	tbl := newDataTable()
	tbl.SetSize(120, 20)
	tbl.SetData(model.ListSpec{Subject: model.SubjectProjects}, []model.Item{
		{ID: "abc12345", Title: "My Project Title", Kind: model.KindProject, Raw: map[string]any{"status": 0}},
	})
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	_, widths := tbl.visibleLayout()
	want := 0
	for _, w := range widths {
		want += w
	}
	row := re.ReplaceAllString(tbl.renderRow(0), "")
	got := runewidth.StringWidth(row)
	if got < want-2 || got > want+2 {
		t.Fatalf("row display width=%d want ~%d", got, want)
	}
}

func TestProjectRowStaysSingleLineWhenSelected(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	long := strings.Repeat("X", 200)
	tbl := newDataTable()
	tbl.SetSize(120, 20)
	tbl.SetData(model.ListSpec{Subject: model.SubjectProjects}, []model.Item{
		{ID: "1", Title: long, Kind: model.KindProject, Raw: map[string]any{"status": 0}, Selected: true},
	})
	if strings.Contains(tbl.renderRow(0), "\n") {
		t.Fatal("selected project row must not wrap to multiple lines")
	}
}

func TestProjectRowStyleUniformBackground(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	tbl := newDataTable()
	item := model.Item{Raw: map[string]any{"status": 0}}
	base := tbl.projectBaseStyle(item)
	titleStyled := tbl.projectStatusStyle("title", item, base).Render("Open Proj")
	statusStyled := tbl.projectStatusStyle("status", item, base).Render(model.ProjectStatusCell(item.Raw))
	if !strings.Contains(titleStyled, "\x1b[") {
		t.Fatalf("title cell should be styled: %q", titleStyled)
	}
	if !strings.Contains(statusStyled, "\x1b[") {
		t.Fatalf("status cell should be styled: %q", statusStyled)
	}
	if titleStyled == statusStyled {
		t.Fatal("status and title cells should use different foreground styles")
	}
}

func TestTruncateCellText(t *testing.T) {
	long := strings.Repeat("x", 40)
	got := truncateCellText(long, 10)
	if runewidth.StringWidth(got) > 10 {
		t.Fatalf("truncated width=%d want <=10", runewidth.StringWidth(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
}

func TestDataTableTruncatesNonSelectedRows(t *testing.T) {
	long := strings.Repeat("A", 60)
	items := []model.Item{
		{ID: "1", Title: "Short"},
		{ID: "2", Title: long},
	}
	tbl := newDataTable()
	tbl.SetSize(50, 10)
	tbl.SetData(model.ListSpec{Subject: model.SubjectTasks}, items)
	tbl.cursorRow = 0

	if strings.Contains(tbl.renderRow(1), long) {
		t.Fatal("non-cursor row should truncate long title")
	}

	tbl.cursorRow = 1
	if !strings.Contains(tbl.renderRow(1), strings.Repeat("A", 10)) {
		t.Fatal("cursor row should show title text within column")
	}
}

func TestDataTableRowWidthMatchesColumns(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(80, 12)
	tbl.SetData(model.ListSpec{Subject: model.SubjectTasks}, sampleItems())
	tbl.SetFocused(true)
	_, widths := tbl.visibleLayout()
	want := 0
	for _, w := range widths {
		want += w
	}
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	row := re.ReplaceAllString(tbl.renderRow(0), "")
	got := runewidth.StringWidth(row)
	if got < want-2 || got > want+2 {
		t.Fatalf("row display width=%d want ~%d", got, want)
	}
}

func TestDataTableShowsFullTextWhenSpaceSelected(t *testing.T) {
	long := strings.Repeat("B", 60)
	items := []model.Item{
		{ID: "1", Title: "Short"},
		{ID: "2", Title: long, Selected: true},
	}
	tbl := newDataTable()
	tbl.SetSize(50, 10)
	tbl.SetData(model.ListSpec{Subject: model.SubjectTasks}, items)
	tbl.cursorRow = 0

	rowSelected := tbl.renderRow(1)
	if !strings.Contains(rowSelected, strings.Repeat("B", 10)) {
		t.Fatal("space-selected row should show title text within column")
	}
}

func indexOfColumn(tbl DataTable, key string) int {
	for i, c := range tbl.columns {
		if c.Key == key {
			return i
		}
	}
	return 0
}

func indexOfColumnKey(cols []model.TableColumn, key string) int {
	for i, c := range cols {
		if c.Key == key {
			return i
		}
	}
	return 0
}
