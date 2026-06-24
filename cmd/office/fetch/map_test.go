package fetch_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/fetch"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestItemsFromMaps(t *testing.T) {
	rows := []map[string]any{
		{"id": float64(1), "title": "Alpha", "status": "Open"},
		{"id": float64(2), "title": "Beta", "status": "Closed"},
	}
	items := fetch.ItemsFromMaps(rows, model.KindTask, fetch.TaskItemFields)
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].ID != "1" || items[0].Title != "Alpha" {
		t.Fatalf("item0=%+v", items[0])
	}
	if items[0].Kind != model.KindTask {
		t.Fatalf("kind=%v", items[0].Kind)
	}
	if items[1].Subtitle != "Closed" {
		t.Fatalf("subtitle=%q", items[1].Subtitle)
	}
}
