package model

import (
	"testing"
	"time"
)

func TestTaskStatusLabelFromInt(t *testing.T) {
	if got := TaskStatusLabel(1); got != "Open" {
		t.Fatalf("got %q", got)
	}
	if got := TaskStatusLabel(2); got != "Closed" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatRelativeDeadlineFuture(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(2 * time.Hour)
	got := FormatRelativeDeadlineAt(deadline.Format(time.RFC3339), now)
	if got != "in 2 hours" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatRelativeDeadlineDays(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(26 * time.Hour)
	got := FormatRelativeDeadlineAt(deadline.Format(time.RFC3339), now)
	if got != "in 1 day" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildTaskColumnsOmitsSubtitle(t *testing.T) {
	items := []Item{{
		ID: "1", Title: "Fix bug", Kind: KindTask, Subtitle: "1",
		Raw: map[string]any{"status": 1, "deadline": "2026-06-25T12:00:00Z"},
	}}
	cols := BuildColumns(SubjectTasks, items)
	for _, c := range cols {
		if c.Key == "subtitle" {
			t.Fatalf("unexpected subtitle column")
		}
	}
	if CellText(items[0], "status") != "Open" {
		t.Fatalf("status not humanized")
	}
}
