package model_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestSelectionToggle(t *testing.T) {
	items := []model.Item{
		{ID: "1", Title: "A"},
		{ID: "2", Title: "B"},
	}
	sel := model.NewSelection()
	if sel.Count() != 0 {
		t.Fatalf("initial count=%d", sel.Count())
	}
	sel.Toggle(&items, 0)
	if !items[0].Selected || sel.Count() != 1 {
		t.Fatalf("after toggle 0: selected=%v count=%d", items[0].Selected, sel.Count())
	}
	sel.Toggle(&items, 0)
	if items[0].Selected || sel.Count() != 0 {
		t.Fatalf("after second toggle: selected=%v count=%d", items[0].Selected, sel.Count())
	}
}

func TestSelectionClearOnSubjectChange(t *testing.T) {
	items := []model.Item{{ID: "1", Title: "A", Selected: true}}
	sel := model.NewSelection()
	sel.Toggle(&items, 0)
	sel.Clear()
	if sel.Count() != 0 {
		t.Fatalf("count=%d after clear", sel.Count())
	}
}

func TestNextFocusPane(t *testing.T) {
	if got := model.NextFocusPane(model.FocusMenu); got != model.FocusList {
		t.Fatalf("menu->list got %v", got)
	}
	if got := model.NextFocusPane(model.FocusPreview); got != model.FocusMenu {
		t.Fatalf("preview->menu got %v", got)
	}
}
