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

func TestKeyActionListToggleSelect(t *testing.T) {
	if got := ui.KeyAction(" ", model.FocusList); got != ui.ActionToggleSelect {
		t.Fatalf("got %v", got)
	}
}

func TestKeyActionPreviewNoOp(t *testing.T) {
	if got := ui.KeyAction(" ", model.FocusPreview); got != ui.ActionNone {
		t.Fatalf("got %v", got)
	}
}

func TestKeyActionTabNextPane(t *testing.T) {
	if got := ui.KeyAction("tab", model.FocusList); got != ui.ActionNextPane {
		t.Fatalf("got %v", got)
	}
}

func TestLayoutWidths(t *testing.T) {
	menu, list, preview := ui.LayoutWidths(120)
	if menu+list+preview > 120 {
		t.Fatalf("widths exceed total: %d+%d+%d", menu, list, preview)
	}
	if menu < 20 || list < 30 || preview < 30 {
		t.Fatalf("widths too narrow: %d %d %d", menu, list, preview)
	}
}
