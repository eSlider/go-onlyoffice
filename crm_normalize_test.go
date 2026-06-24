package onlyoffice

import "testing"

func TestNormalizeCompanyName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{" 711media ", "711media"},
		{"711Media", "711media"},
		{"Acme  Corp", "acme corp"},
	}
	for _, tc := range tests {
		if got := NormalizeCompanyName(tc.in); got != tc.want {
			t.Errorf("NormalizeCompanyName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizePersonKey(t *testing.T) {
	if got := NormalizePersonKey(" Jane ", " Doe "); got != "jane doe" {
		t.Fatalf("got %q", got)
	}
}

func TestFixDealTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{" @ 711media", "711media"},
		{"@ 711media", "711media"},
		{"@Acme", "Acme"},
		{"Dev@Acme", "Dev @ Acme"},
		{"Dev  @  Acme", "Dev @ Acme"},
		{"Senior Dev @ Acme", "Senior Dev @ Acme"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := FixDealTitle(tc.in); got != tc.want {
			t.Errorf("FixDealTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStripCompanySuffix(t *testing.T) {
	if got := StripCompanySuffix("Dev @ Acme"); got != "Dev" {
		t.Fatalf("got %q", got)
	}
	if got := StripCompanySuffix("Dev"); got != "Dev" {
		t.Fatalf("got %q", got)
	}
}

func TestContactInfoKey(t *testing.T) {
	if got := ContactInfoKey("Email", "  A@B.COM "); got != "email|a@b.com" {
		t.Fatalf("got %q", got)
	}
}

func TestMemberDisplayKey(t *testing.T) {
	if got := MemberDisplayKey(" 711media "); got != "711media" {
		t.Fatalf("got %q", got)
	}
}

func TestDealTitleForApplication(t *testing.T) {
	if got := DealTitleForApplication("", "711media"); got != "711media" {
		t.Fatalf("got %q", got)
	}
	if got := DealTitleForApplication("Dev", "Acme"); got != "Dev @ Acme" {
		t.Fatalf("got %q", got)
	}
}
