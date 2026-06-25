package model

import "testing"

func TestTaskStatusLifecycleCycle(t *testing.T) {
	cur := TaskLifecycleOpen
	cur = cur.Next()
	if cur != TaskLifecycleClosed {
		t.Fatalf("got %v", cur)
	}
	cur = cur.Prev()
	if cur != TaskLifecycleOpen {
		t.Fatalf("got %v", cur)
	}
}

func TestFormFieldsFromRawTask(t *testing.T) {
	raw := map[string]any{
		"title":       "Fix",
		"description": "Details",
		"status":      1,
		"projectOwner": map[string]any{"title": "Alpha"},
		"startDate":   "2026-06-01T10:00:00Z",
		"deadline":    "2026-06-10T18:00:00Z",
		"responsibles": []any{
			map[string]any{"id": "u1", "displayName": "Alice"},
		},
	}
	f := FormFieldsFromRaw(KindTask, raw)
	if !f.HasTaskStatus || f.TaskStatus != TaskLifecycleOpen {
		t.Fatalf("status: %+v", f)
	}
	if !f.HasResponsible || f.ResponsibleID != "u1" {
		t.Fatalf("responsible: %+v", f)
	}
	if f.ProjectTitle != "Alpha" {
		t.Fatalf("project: %q", f.ProjectTitle)
	}
	if f.TimingSummary == "" {
		t.Fatal("expected timing summary")
	}
}
