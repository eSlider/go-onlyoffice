package model

import "testing"

func TestClassifyCalendarRow(t *testing.T) {
	kind, label := ClassifyCalendarRow(map[string]any{
		"title": "Standup",
		"start": "2026-06-24T10:00:00Z",
	})
	if kind != KindEvent || label != "Event" {
		t.Fatalf("got kind=%s label=%q", kind, label)
	}
	kind, label = ClassifyCalendarRow(map[string]any{"title": "Work"})
	if kind != KindCalendar || label != "Calendar" {
		t.Fatalf("got kind=%s label=%q", kind, label)
	}
}

func TestBuildCalendarColumnsIncludesType(t *testing.T) {
	items := []Item{{
		ID: "1", Title: "Meet", Kind: KindEvent,
		Raw: map[string]any{"type": "Event", "start": "2026-06-24T10:00:00Z"},
	}}
	cols := BuildColumns(SubjectCalendar, items)
	found := false
	for _, c := range cols {
		if c.Key == "type" {
			found = true
		}
		if c.Key == "subtitle" {
			t.Fatal("subtitle column should not appear")
		}
	}
	if !found {
		t.Fatal("expected type column")
	}
}
