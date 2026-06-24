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
	if got := ui.KeyAction("backtab", model.FocusPreview); got != ui.ActionPrevPane {
		t.Fatalf("got %v", got)
	}
}

func TestKeyActionTabNextPane(t *testing.T) {
	if got := ui.KeyAction("tab", model.FocusList); got != ui.ActionNextPane {
		t.Fatalf("got %v", got)
	}
}

func TestKeyActionOpenActions(t *testing.T) {
	if got := ui.KeyAction("a", model.FocusList); got != ui.ActionOpenActions {
		t.Fatalf("got %v", got)
	}
}

func TestLayoutWidths(t *testing.T) {
	menu, list, preview := ui.LayoutWidths(120)
	if menu+list+preview > 120 {
		t.Fatalf("widths exceed total: %d+%d+%d", menu, list, preview)
	}
}
