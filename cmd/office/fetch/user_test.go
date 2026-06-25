package fetch

import (
	"fmt"
	"testing"
)

func TestUserProfileUpdateBodyOmitsStatus(t *testing.T) {
	body := userProfileUpdateBody(true, nil)
	if _, ok := body["status"]; ok {
		t.Fatal("profile update must not include status")
	}
	if body["isAdmin"] != true {
		t.Fatal("expected isAdmin true")
	}
}

func TestUserProfileUpdateBodyPartialAdminModules(t *testing.T) {
	body := userProfileUpdateBody(false, []string{"documents", "crm"})
	if body["isAdmin"] != false {
		t.Fatal("expected partial admin")
	}
	mods, ok := body["listAdminModules"].([]string)
	if !ok || len(mods) != 2 {
		t.Fatalf("modules=%v", body["listAdminModules"])
	}
}

func TestIsSuspendedUserError(t *testing.T) {
	err := fmt.Errorf(`PUT JSON /api/2.0/people/x: 500 {"error":{"message":"The user is suspended"}}`)
	if !isSuspendedUserError(err) {
		t.Fatal("expected suspended detection")
	}
}
