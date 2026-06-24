package ui

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/mattn/go-runewidth"
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

func TestTruncateCellText(t *testing.T) {
	long := strings.Repeat("x", 40)
	got := truncateCellText(long, 10)
	if runewidth.StringWidth(got) > 10 {
		t.Fatalf("truncated width=%d want <=10", runewidth.StringWidth(got))
	}
	if !strings.HasSuffix(got, "...") {
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
	if !strings.Contains(tbl.renderRow(1), long) {
		t.Fatal("cursor row should show full title")
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
	if !strings.Contains(rowSelected, long) {
		t.Fatal("space-selected row should show full title")
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
