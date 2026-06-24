package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestDetailDocumentScrollByKey(t *testing.T) {
	d := newDetailPane()
	d.SetSize(50, 20)
	longBody := "# Mail\n\n"
	for i := 0; i < 80; i++ {
		longBody += "line\n"
	}
	d.LoadDocument(model.Item{ID: "1", Kind: model.KindMail, Title: "M"}, longBody, 40)
	d.SetFocused(true)

	if d.DocumentYOffset() != 0 {
		t.Fatalf("expected top offset 0, got %d", d.DocumentYOffset())
	}
	if !d.ScrollDocument("down") {
		t.Fatal("expected scroll down to succeed")
	}
	if d.DocumentYOffset() == 0 {
		t.Fatal("expected offset after scroll down")
	}
	if !d.ScrollDocument("up") {
		t.Fatal("expected scroll up to succeed")
	}
}

func TestDetailDocumentMouseWheel(t *testing.T) {
	d := newDetailPane()
	d.SetSize(50, 20)
	longBody := "# Mail\n\n"
	for i := 0; i < 80; i++ {
		longBody += "line\n"
	}
	d.LoadDocument(model.Item{ID: "1", Kind: model.KindMail, Title: "M"}, longBody, 40)

	d.ScrollDocumentMouse(tea.MouseMsg{
		X: 10, Y: 2, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
	})
	if d.DocumentYOffset() == 0 {
		t.Fatal("expected wheel down to scroll document")
	}
}

func TestDetailPaneXRange(t *testing.T) {
	pw := PaneWidths{
		Menu: 12, List: 72, Detail: 36,
		Visibility: PaneVisibility{Menu: true, List: true, Detail: true},
	}
	start, end := DetailPaneXRange(pw)
	if start != 84 || end != 120 {
		t.Fatalf("range=%d..%d want 84..120", start, end)
	}
}
