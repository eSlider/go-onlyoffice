package applications

import (
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
