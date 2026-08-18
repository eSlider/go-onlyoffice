package catalog

import (
	"testing"
)

func TestMatchAgainstOO_emailThenName(t *testing.T) {
	// MatchAgainstOO needs a live client; unit-test the helpers used by matching.
	c := map[string]any{
		"id":        float64(42),
		"isCompany": false,
		"firstName": "Ada",
		"lastName":  "Lovelace",
		"email":     "ada@example.com",
		"commonData": []any{
			map[string]any{"infoType": "Email", "data": "ada.alt@example.com"},
		},
	}
	if contactIDString(c) != "42" {
		t.Fatal(contactIDString(c))
	}
	emails := contactEmails(c)
	if len(emails) < 2 {
		t.Fatalf("%v", emails)
	}
}
