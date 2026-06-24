package model_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestFilterItemsByTitle(t *testing.T) {
	items := []model.Item{
		{ID: "1", Title: "Alpha task"},
		{ID: "2", Title: "Beta task"},
	}
	got := model.FilterItems(items, "alpha")
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterItemsEmptyQueryReturnsAll(t *testing.T) {
	items := []model.Item{{ID: "1", Title: "A"}, {ID: "2", Title: "B"}}
	got := model.FilterItems(items, "  ")
	if len(got) != 2 {
		t.Fatalf("got len=%d want 2", len(got))
	}
}
