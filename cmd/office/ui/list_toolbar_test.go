package ui

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestListToolbarShowsFilterAndActions(t *testing.T) {
	b := newListToolbar()
	b.SetWidth(60)
	view := b.View(ListToolbarMeta{
		Subject: "projects",
		Count:   3,
	})
	if !strings.Contains(view, "projects (3)") {
		t.Fatalf("missing subject: %q", view)
	}
	if !strings.Contains(view, "💾") || !strings.Contains(view, "🗑") {
		t.Fatalf("missing action icons: %q", view)
	}
}

func TestListToolbarActionsEnabledFromMeta(t *testing.T) {
	b := newListToolbar()
	on := ListToolbarMeta{SaveEnabled: true, DeleteEnabled: true}
	off := ListToolbarMeta{}
	if !b.IsActionEnabled(model.ActionSave, on) {
		t.Fatal("save should be enabled")
	}
	if b.IsActionEnabled(model.ActionSave, off) {
		t.Fatal("save should be disabled")
	}
	if !b.IsActionEnabled(model.ActionDelete, on) {
		t.Fatal("delete should be enabled")
	}
}

func TestCanSaveFromListRequiresSelection(t *testing.T) {
	m := Model{
		items: []model.Item{
			{ID: "1", Title: "A", Kind: model.KindProject, Selected: true},
		},
	}
	if m.canSaveFromList() {
		t.Fatal("save should require matching loaded detail")
	}
	m.detail = newDetailPane()
	m.detail.SetSize(40, 20)
	m.detail.LoadForm(model.Item{ID: "1", Kind: model.KindProject}, model.FormFields{
		PrimaryLabel: "Title", SecondaryLabel: "Description",
		Primary: "A", HasStatus: true,
	})
	if !m.canSaveFromList() {
		t.Fatal("save should be enabled with one selected row and loaded detail")
	}
}
