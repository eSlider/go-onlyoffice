package ui_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/eslider/go-onlyoffice/cmd/office/ui"
)

func TestKeyActionMenuDown(t *testing.T) {
	if got := ui.KeyAction("j", model.FocusMenu); got != ui.ActionMoveDown {
		t.Fatalf("got %v", got)
	}
}

func TestKeyActionShiftTabPrevPane(t *testing.T) {
	if got := ui.KeyAction("shift+tab", model.FocusList); got != ui.ActionPrevPane {
		t.Fatalf("got %v", got)
	}
}

func TestKeyActionTabOnPreviewDoesNotSwitchPane(t *testing.T) {
	if got := ui.KeyAction("tab", model.FocusPreview); got != ui.ActionNone {
		t.Fatalf("tab on preview should stay in pane, got %v", got)
	}
}

func TestKeyActionSortAndColumns(t *testing.T) {
	if got := ui.KeyAction("s", model.FocusList); got != ui.ActionSort {
		t.Fatalf("got %v", got)
	}
}

func TestKeyActionFilter(t *testing.T) {
	if got := ui.KeyAction("f", model.FocusList); got != ui.ActionFilter {
		t.Fatalf("got %v", got)
	}
}

func TestLayoutWidths(t *testing.T) {
	menu, list, preview := ui.LayoutWidthsLegacy(120)
	if menu+list+preview != 120 {
		t.Fatalf("widths exceed total: %d+%d+%d", menu, list, preview)
	}
}
