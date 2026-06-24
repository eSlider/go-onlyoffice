package model_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestNavTreeHasExpectedRoots(t *testing.T) {
	tree := model.DefaultNavTree()
	roots := tree.RootLabels()
	want := []string{"Projects", "Calendar", "CRM", "Mail", "Users"}
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
	if !tree.IsExpandable(0) {
		t.Fatal("Projects should be expandable")
	}
	tree.ToggleExpand(0)
	if !tree.IsExpanded(0) {
		t.Fatal("Projects should expand")
	}
	tree.ToggleExpand(0)
	if tree.IsExpanded(0) {
		t.Fatal("Projects should collapse")
	}
}

func TestNavLeafReturnsListSpec(t *testing.T) {
	tree := model.DefaultNavTree()
	tree.ToggleExpand(0) // Projects
	tree.ToggleExpand(1) // Browse
	// Find "All projects" leaf cursor
	var found int = -1
	for i := 0; i < tree.VisibleCount(); i++ {
		n, ok := tree.NodeAtVisible(i)
		if ok && n.List != nil && n.List.Subject == model.SubjectProjects {
			found = i
			break
		}
	}
	if found < 0 {
		t.Fatal("all projects leaf not found")
	}
	tree.SetCursor(found)
	spec, ok := tree.CurrentListSpec()
	if !ok || spec.Subject != model.SubjectProjects {
		t.Fatalf("spec=%v ok=%v", spec, ok)
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

func TestActionsForContact(t *testing.T) {
	acts := model.ActionsFor(model.KindContact)
	if len(acts) < 2 {
		t.Fatalf("expected view+delete, got %d", len(acts))
	}
}
