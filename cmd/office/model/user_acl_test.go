package model

import "testing"

func TestUserACLFromRawFullAccess(t *testing.T) {
	raw := map[string]any{
		"isAdmin":          true,
		"listAdminModules": []any{"crm"},
	}
	acl := UserACLFromRaw(raw)
	if !acl.FullAccess {
		t.Fatal("expected full access")
	}
	if !acl.ACLModuleOn("documents") {
		t.Fatal("full access should show all modules on")
	}
}

func TestUserACLAPIPayloadPartial(t *testing.T) {
	acl := UserACLState{
		FullAccess: false,
		Modules: map[string]bool{
			"documents": true,
			"projects":  true,
		},
	}
	isAdmin, mods := acl.APIPayload()
	if isAdmin {
		t.Fatal("expected partial admin")
	}
	if len(mods) != 2 {
		t.Fatalf("mods=%v", mods)
	}
}

func TestUserIsEnabled(t *testing.T) {
	if !UserIsEnabled(map[string]any{"status": EmployeeStatusActive}) {
		t.Fatal("active user should be enabled")
	}
	if UserIsEnabled(map[string]any{"status": EmployeeStatusTerminated}) {
		t.Fatal("terminated user should be disabled")
	}
}
