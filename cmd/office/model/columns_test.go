package model

import "testing"

func TestBuildProjectColumns(t *testing.T) {
	items := []Item{{
		ID: "1", Title: "Alpha", Kind: KindProject,
		Raw: map[string]any{
			"taskCount": 3, "taskCountTotal": 5,
			"documentsCount": 2, "participantCount": 4,
		},
	}}
	cols := BuildColumns(SubjectProjects, items)
	want := []string{"_sel", "id", "status", "title", "tasks", "documents", "users"}
	if len(cols) != len(want) {
		t.Fatalf("got %d columns, want %d", len(cols), len(want))
	}
	for i, key := range want {
		if cols[i].Key != key {
			t.Fatalf("col[%d]=%q want %q", i, cols[i].Key, key)
		}
	}
	for _, c := range cols {
		if c.Key == "subtitle" {
			t.Fatalf("unexpected column %q", c.Key)
		}
	}
}

func TestCellTextProjectStatus(t *testing.T) {
	open := Item{Kind: KindProject, Raw: map[string]any{"status": 0}}
	if got := CellText(open, "status"); got != "🟢 Open" {
		t.Fatalf("got %q", got)
	}
	closed := Item{Kind: KindProject, Raw: map[string]any{"status": 2}}
	if got := CellText(closed, "status"); got != "🔴 Closed" {
		t.Fatalf("got %q", got)
	}
	paused := Item{Kind: KindProject, Raw: map[string]any{"status": 1}}
	if got := CellText(paused, "status"); got != "🟡 Paused" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildProjectColumnsStatusHeaderIsIcon(t *testing.T) {
	cols := BuildColumns(SubjectProjects, nil)
	for _, c := range cols {
		if c.Key == "status" && c.Title != "●" {
			t.Fatalf("status header=%q want ●", c.Title)
		}
	}
}

func TestFormatProjectTasksOpenClosed(t *testing.T) {
	raw := map[string]any{"taskCount": 3, "taskCountTotal": 8}
	if got := formatProjectTasks(raw); got != "3/5" {
		t.Fatalf("got %q want 3/5", got)
	}
}

func TestCellTextProjectDocumentsUsers(t *testing.T) {
	it := Item{Raw: map[string]any{"documentsCount": 7, "participantCount": 2}}
	if got := CellText(it, "documents"); got != "7" {
		t.Fatalf("documents: %q", got)
	}
	if got := CellText(it, "users"); got != "2" {
		t.Fatalf("users: %q", got)
	}
}

func TestBuildColumnsIncludesSelectionAndCoreFields(t *testing.T) {
	items := []Item{
		{ID: "1", Title: "Alpha", Kind: KindTask, Raw: map[string]any{"status": "Open"}},
	}
	cols := BuildColumns(SubjectTasks, items)
	if cols[0].Key != "_sel" {
		t.Fatalf("first column should be selection, got %q", cols[0].Key)
	}
}
