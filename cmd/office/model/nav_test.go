package model_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestNavTreeHasExpectedRoots(t *testing.T) {
	tree := model.DefaultNavTree()
	roots := tree.RootLabels()
	want := []string{"Projects", "Tasks", "By project", "Calendar", "CRM", "Mail", "Users"}
	if len(roots) != len(want) {
		t.Fatalf("roots=%v want %v", roots, want)
	}
	for i, w := range want {
		if roots[i] != w {
			t.Fatalf("root[%d]=%q want %q", i, roots[i], w)
		}
	}
}

func TestNavExpandCollapse(t *testing.T) {
	tree := model.DefaultNavTree()
	var crm int = -1
	for i := 0; i < tree.VisibleCount(); i++ {
		n, ok := tree.NodeAtVisible(i)
		if ok && n.Label == "CRM" {
			crm = i
			break
		}
	}
	if crm < 0 {
		t.Fatal("CRM node not found")
	}
	if !tree.IsExpandable(crm) {
		t.Fatal("CRM should be expandable")
	}
	tree.ToggleExpand(crm)
	if !tree.IsExpanded(crm) {
		t.Fatal("CRM should expand")
	}
	tree.ToggleExpand(crm)
	if tree.IsExpanded(crm) {
		t.Fatal("CRM should collapse")
	}
}

func TestNavCalendarLeafReturnsListSpec(t *testing.T) {
	tree := model.DefaultNavTree()
	var found int = -1
	for i := 0; i < tree.VisibleCount(); i++ {
		n, ok := tree.NodeAtVisible(i)
		if ok && n.Label == "Calendar" {
			found = i
			break
		}
	}
	if found < 0 {
		t.Fatal("Calendar node not found")
	}
	tree.SetCursor(found)
	spec, ok := tree.CurrentListSpec()
	if !ok || spec.Subject != model.SubjectCalendar {
		t.Fatalf("spec=%v ok=%v", spec, ok)
	}
}

func TestNavProjectsLeafReturnsListSpec(t *testing.T) {
	tree := model.DefaultNavTree()
	tree.SetCursor(0)
	spec, ok := tree.CurrentListSpec()
	if !ok || spec.Subject != model.SubjectProjects {
		t.Fatalf("projects spec=%v ok=%v", spec, ok)
	}
}

func TestNavTasksLeafReturnsListSpec(t *testing.T) {
	tree := model.DefaultNavTree()
	var found int = -1
	for i := 0; i < tree.VisibleCount(); i++ {
		n, ok := tree.NodeAtVisible(i)
		if ok && n.Label == "Tasks" {
			found = i
			break
		}
	}
	if found < 0 {
		t.Fatal("Tasks node not found")
	}
	tree.SetCursor(found)
	spec, ok := tree.CurrentListSpec()
	if !ok || spec.Subject != model.SubjectTasks {
		t.Fatalf("tasks spec=%v ok=%v", spec, ok)
	}
}

func TestPrevFocusPane(t *testing.T) {
	if got := model.PrevFocusPane(model.FocusList); got != model.FocusMenu {
		t.Fatalf("got %v", got)
	}
	if got := model.PrevFocusPane(model.FocusMenu); got != model.FocusPreview {
		t.Fatalf("got %v", got)
	}
}

func TestNavFilterByLabel(t *testing.T) {
	tree := model.DefaultNavTree()
	tree.SetFilter("inbox")
	found := false
	for i := 0; i < tree.VisibleCount(); i++ {
		n, ok := tree.NodeAtVisible(i)
		if ok && n.Label == "Inbox" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Inbox should remain visible when filtering mail")
	}
	for i := 0; i < tree.VisibleCount(); i++ {
		n, ok := tree.NodeAtVisible(i)
		if ok && n.Label == "Calendar" {
			t.Fatal("Calendar should be hidden when filtering inbox")
		}
	}
}

func TestNavClearFilterRestoresTree(t *testing.T) {
	tree := model.DefaultNavTree()
	before := tree.VisibleCount()
	tree.SetFilter("zzz-no-match")
	if tree.VisibleCount() >= before {
		t.Fatalf("filter should shrink visible nodes")
	}
	tree.ClearFilter()
	if tree.VisibleCount() != before {
		t.Fatalf("clear filter should restore visible count")
	}
}

func TestActionsForContact(t *testing.T) {
	acts := model.ActionsFor(model.KindContact)
	if len(acts) != 1 || acts[0].ID != model.ActionDelete {
		t.Fatalf("expected delete only, got %v", acts)
	}
}
