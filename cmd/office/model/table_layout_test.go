package model

import "testing"

func TestTableFlexLayoutFor(t *testing.T) {
	proj, ok := TableFlexLayoutFor(SubjectProjects)
	if !ok || proj.FlexColumnKey != "title" || proj.MinFlexWidth != 12 {
		t.Fatalf("projects layout=%+v ok=%v", proj, ok)
	}
	users, ok := TableFlexLayoutFor(SubjectUsers)
	if !ok || users.FlexColumnKey != "email" {
		t.Fatalf("users layout=%+v", users)
	}
	_, ok = TableFlexLayoutFor(SubjectTasks)
	if ok {
		t.Fatal("tasks should use scrolling layout")
	}
}
