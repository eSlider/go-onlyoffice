package ui_test

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/eslider/go-onlyoffice/cmd/office/ui"
)

func TestEntityFormLoadAndView(t *testing.T) {
	form := ui.NewEntityFormForTest()
	form.SetSize(40, 16)
	form.Load(model.KindTask, "42", model.FormFields{
		PrimaryLabel: "Title", SecondaryLabel: "Description",
		Primary: "Fix bug", Secondary: "Details here",
	})
	view := form.View()
	if !strings.Contains(view, "Fix bug") {
		t.Fatal("view missing title")
	}
}

func TestDetailPaneFormAndActions(t *testing.T) {
	d := ui.NewDetailPaneForTest()
	d.SetSize(50, 20)
	d.SetFocused(true)
	d.LoadForm(model.Item{ID: "1", Kind: model.KindTask, Title: "T"}, model.FormFields{
		PrimaryLabel: "Title", SecondaryLabel: "Description",
		Primary: "Hello", Secondary: "World",
	})
	view := d.View()
	if !strings.Contains(view, "Save") {
		t.Fatal("expected Save action button")
	}
	if !strings.Contains(view, "Hello") {
		t.Fatal("expected form content")
	}
}

func TestDetailPaneDocumentMode(t *testing.T) {
	d := ui.NewDetailPaneForTest()
	d.SetSize(50, 20)
	d.LoadDocument(model.Item{ID: "9", Kind: model.KindFile, Title: "a.txt"}, "# Doc\n\nbody", 40)
	view := d.View()
	if !strings.Contains(view, "Download") {
		t.Fatal("expected Download action")
	}
}
