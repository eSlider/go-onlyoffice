package model_test

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestNavUsersLeafReturnsListSpec(t *testing.T) {
	tree := model.DefaultNavTree()
	var found int = -1
	for i := 0; i < tree.VisibleCount(); i++ {
		n, ok := tree.NodeAtVisible(i)
		if ok && n.Label == "Users" {
			found = i
			break
		}
	}
	if found < 0 {
		t.Fatal("Users node not found")
	}
	tree.SetCursor(found)
	spec, ok := tree.CurrentListSpec()
	if !ok || spec.Subject != model.SubjectUsers {
		t.Fatalf("users spec=%v ok=%v", spec, ok)
	}
	if tree.IsExpandable(found) {
		t.Fatal("Users should be a direct leaf, not a branch")
	}
}

func TestBuildUserColumnsOmitsIDAndDisplayName(t *testing.T) {
	items := []model.Item{{
		ID: "1", Title: "jdoe", Kind: model.KindUser,
		Raw: map[string]any{
			"id": "1", "userName": "jdoe", "displayName": "John Doe",
			"email": "j@example.com", "registrationDate": "2024-01-02T00:00:00",
		},
	}}
	cols := model.BuildColumns(model.SubjectUsers, items)
	want := []string{"_sel", "userName", "registration", "status", "email"}
	if len(cols) != len(want) {
		t.Fatalf("got %v", columnKeys(cols))
	}
	for i, key := range want {
		if cols[i].Key != key {
			t.Fatalf("col[%d]=%q want %q", i, cols[i].Key, key)
		}
	}
}

func columnKeys(cols []model.TableColumn) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Key
	}
	return out
}
