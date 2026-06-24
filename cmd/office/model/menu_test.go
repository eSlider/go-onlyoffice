package model_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestMenuTreeHasExpectedRoots(t *testing.T) {
	tree := model.DefaultMenuTree()
	roots := tree.RootLabels()
	want := []string{"Projects", "Calendar", "CRM", "Mail", "Documents", "Users"}
	if len(roots) != len(want) {
		t.Fatalf("roots=%v want %v", roots, want)
	}
	for i, w := range want {
		if roots[i] != w {
			t.Fatalf("root[%d]=%q want %q", i, roots[i], w)
		}
	}
}

func TestMenuExpandCollapse(t *testing.T) {
	tree := model.DefaultMenuTree()
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

func TestMenuSelectLeafReturnsSubject(t *testing.T) {
	tree := model.DefaultMenuTree()
	// Expand Projects and select "All projects" (first child).
	tree.ToggleExpand(0)
	subj, ok := tree.SelectIndex(1)
	if !ok {
		t.Fatal("expected leaf subject")
	}
	if subj != model.SubjectProjects {
		t.Fatalf("subject=%v want Projects", subj)
	}
}
