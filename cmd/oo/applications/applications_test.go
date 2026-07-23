package applications

import (
	"os"
	"path/filepath"
	"testing"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

func TestDealTitleForApplication(t *testing.T) {
	if got := onlyoffice.DealTitleForApplication("", "711media"); got != "711media" {
		t.Fatalf("got %q", got)
	}
	if got := onlyoffice.DealTitleForApplication("Dev", "Acme"); got != "Dev @ Acme" {
		t.Fatalf("got %q", got)
	}
}

func TestLooksLikeAppFolder(t *testing.T) {
	cases := map[string]bool{
		"uae-009-dizzaract-senior-backend-devops-abu-dhabi":    true,
		"djinni-001-devops-engineer-windows-cloud-justmarkets": true,
		"linkedin-elastic-senior-go-intake-observability":      true, // no NNN but long
		"alex-staff-cloud-platform-engineer":                   true,
		"http-client":                                          false,
		"node_modules":                                         false,
		"tools":                                                false,
		"README":                                               false,
		"uae-009":                                              false, // too short
	}
	for name, want := range cases {
		if got := looksLikeAppFolder(name); got != want {
			t.Errorf("looksLikeAppFolder(%q)=%v want %v", name, got, want)
		}
	}
}

func TestDiscoverSkipsNodeModules(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "uae-009-dizzaract-senior-backend-devops-abu-dhabi")
	junk := filepath.Join(root, "node_modules", "http-client")
	tools := filepath.Join(root, "tools", "gmail-oo-reconcile")
	for _, d := range []string{appDir, junk, tools} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "README.md"), []byte("# x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(filepath.Dir(got[0])) != filepath.Base(appDir) {
		t.Fatalf("Discover=%v want only %s", got, appDir)
	}
}

func TestHasContactInfoHelper(t *testing.T) {
	contact := map[string]any{
		"commonData": []any{
			map[string]any{"infoType": "Email", "data": "a@b.com"},
		},
	}
	if !onlyoffice.HasContactInfo(contact, "Email", "a@b.com") {
		t.Fatal("expected match")
	}
	if onlyoffice.HasContactInfo(contact, "Email", "other@b.com") {
		t.Fatal("unexpected match")
	}
}
