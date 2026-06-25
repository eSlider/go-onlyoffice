package ui

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestUserFormViewShowsIDAndACL(t *testing.T) {
	f := NewEntityFormForTest()
	f.SetSize(60, 30)
	f.Load(model.KindUser, "42", model.FormFields{
		HasUserEdit: true,
		UserEnabled: true,
		UserACL: model.UserACLState{
			FullAccess: false,
			Modules: map[string]bool{
				"documents": true,
			},
		},
		GroupsText: "Admins, Devs",
	})
	view := f.View()
	for _, want := range []string{"User 42", "ID", "42", "Account", "Password", "Full access", "Documents", "Groups", "Admins"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestUserFormFieldCount(t *testing.T) {
	f := NewEntityFormForTest()
	f.Load(model.KindUser, "1", model.FormFields{HasUserEdit: true, UserACL: model.UserACLFromRaw(nil)})
	if got := f.FieldCount(); got != 2+len(model.UserACLDefs) {
		t.Fatalf("field count=%d want %d", got, len(model.UserACLDefs)+2)
	}
}

func TestLayoutUserTableEmailAbsorbsWidth(t *testing.T) {
	cols := model.BuildColumns(model.SubjectUsers, nil)
	flex, ok := model.TableFlexLayoutFor(model.SubjectUsers)
	if !ok {
		t.Fatal("expected flex layout")
	}
	lay := layoutFlexTable(cols, 100, flex.FlexColumnKey, flex.MinFlexWidth)
	sum := 0
	emailW := 0
	for _, i := range lay.indices {
		sum += lay.widths[i]
		if cols[i].Key == "email" {
			emailW = lay.widths[i]
		}
	}
	if sum != 100 {
		t.Fatalf("sum=%d want 100", sum)
	}
	if emailW < 30 {
		t.Fatalf("email width=%d should absorb extra space", emailW)
	}
}
