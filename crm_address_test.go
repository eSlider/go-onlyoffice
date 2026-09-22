package onlyoffice

import "testing"

func TestAddressCategoryCode(t *testing.T) {
	cases := map[string]int{
		"Home": 0, "Postal": 1, "Office": 2, "Billing": 3, "Other": 4, "Work": 5,
		"billing": 3, " work ": 5, "3": 3, "5": 5,
		"": 3, "nonsense": 3, "99": 3,
	}
	for in, want := range cases {
		if got := AddressCategoryCode(in); got != want {
			t.Errorf("AddressCategoryCode(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestHasContactAddress(t *testing.T) {
	contact := map[string]any{
		"addresses": []any{
			map[string]any{
				"street": "Lubanas st. 125a-25", "city": "Riga",
				"zip": "LV-1021", "country": "Latvia", "category": float64(3),
			},
		},
	}
	if !HasContactAddress(contact, "  Lubanas St. 125a-25 ", "riga", "lv-1021", "Billing") {
		t.Error("want match (normalized, case-insensitive)")
	}
	if HasContactAddress(contact, "Lubanas st. 125a-25", "Riga", "LV-1021", "Work") {
		t.Error("different category must not match")
	}
	if HasContactAddress(contact, "Lubanas st. 125a-25", "Riga", "00000", "Billing") {
		t.Error("different zip must not match")
	}
	if HasContactAddress(map[string]any{}, "x", "y", "z", "Billing") {
		t.Error("empty contact must not match")
	}
	if got := ContactAddresses(contact); len(got) != 1 {
		t.Fatalf("ContactAddresses = %d rows", len(got))
	}
}
