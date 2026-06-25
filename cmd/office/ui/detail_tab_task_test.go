package ui

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestDetailTabOrderTaskForm(t *testing.T) {
	d := newDetailPane()
	d.SetFocused(true)
	d.LoadForm(model.Item{ID: "9", Kind: model.KindTask, Title: "T"}, model.FormFields{
		PrimaryLabel: "Title", SecondaryLabel: "Description",
		Primary: "Alpha", Secondary: "Beta",
		HasTaskStatus: true, TaskStatus: model.TaskLifecycleOpen,
		HasResponsible: true, ResponsibleID: "u1",
		UserChoices: []model.UserOption{{ID: "u1", Name: "Alice"}},
		ProjectTitle: "Proj", TimingSummary: "2026-01-01 → 2026-02-01",
	})

	stops := []entityField{
		entityFieldPrimary, entityFieldSecondary, entityFieldStatus, entityFieldResponsible,
	}
	for i, want := range stops {
		if d.form.field != want {
			t.Fatalf("stop %d: field=%d want %d", i, d.form.field, want)
		}
		if d.TabForward() {
			t.Fatalf("tab from stop %d should stay in pane", i)
		}
	}
	if d.Zone() != detailZoneActions || d.actionIdx != 0 {
		t.Fatalf("expected Save action, zone=%d idx=%d", d.Zone(), d.actionIdx)
	}
	if d.TabForward() {
		t.Fatal("tab from Save should stay")
	}
	if d.actionIdx != 1 {
		t.Fatalf("expected Close action, idx=%d", d.actionIdx)
	}
}
