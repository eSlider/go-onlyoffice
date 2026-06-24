package ui

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestDetailTabOrderProjectForm(t *testing.T) {
	d := newDetailPane()
	d.SetFocused(true)
	d.LoadForm(model.Item{ID: "1", Kind: model.KindProject, Title: "P"}, model.FormFields{
		PrimaryLabel: "Title", SecondaryLabel: "Description",
		Primary: "Alpha", Secondary: "Beta", HasStatus: true,
		Status: model.ProjectLifecycleOpen,
	})

	if d.tabStop != 0 || d.form.field != entityFieldPrimary {
		t.Fatalf("start on title, tabStop=%d field=%d", d.tabStop, d.form.field)
	}
	if d.TabForward() {
		t.Fatal("tab from title should stay in pane")
	}
	if d.form.field != entityFieldSecondary {
		t.Fatalf("second stop should be description, field=%d", d.form.field)
	}
	if d.TabForward() {
		t.Fatal("tab from description should stay in pane")
	}
	if d.form.field != entityFieldStatus {
		t.Fatalf("third stop should be status, field=%d", d.form.field)
	}
	if d.TabForward() {
		t.Fatal("tab from status should stay in pane")
	}
	if d.Zone() != detailZoneActions || d.actionIdx != 0 {
		t.Fatalf("fourth stop should be Save, zone=%d idx=%d", d.Zone(), d.actionIdx)
	}
	if d.TabForward() {
		t.Fatal("tab from Save should stay in pane")
	}
	if d.actionIdx != 1 {
		t.Fatalf("fifth stop should be Delete, idx=%d", d.actionIdx)
	}
	if !d.TabForward() {
		t.Fatal("tab from Delete should leave pane")
	}
	if d.tabStop != 0 {
		t.Fatalf("after leaving pane tab order resets, tabStop=%d", d.tabStop)
	}
}

func TestDetailShiftTabFromTitleLeavesPane(t *testing.T) {
	d := newDetailPane()
	d.SetFocused(true)
	d.LoadForm(model.Item{ID: "1", Kind: model.KindProject}, model.FormFields{
		PrimaryLabel: "Title", SecondaryLabel: "Description",
	})
	if !d.TabBackward() {
		t.Fatal("shift+tab from title should leave pane")
	}
	if d.tabStop != d.maxTabStop() {
		t.Fatalf("should park on last stop, tabStop=%d max=%d", d.tabStop, d.maxTabStop())
	}
}
