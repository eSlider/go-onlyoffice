package ui

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
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
	tbl.SetData(model.ListSpec{Subject: model.SubjectTasks}, items)
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

func indexOfColumn(tbl DataTable, key string) int {
	for i, c := range tbl.columns {
		if c.Key == key {
			return i
		}
	}
	return 0
}
