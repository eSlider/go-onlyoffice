package onlyoffice

import "testing"

func TestCleanPersonNames_stripsCompanyParen(t *testing.T) {
	f, l := CleanPersonNames("Jane", "Doe (Acme)", "Jane Doe (Acme)", "Acme", nil)
	if f != "Jane" || l != "Doe" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNames_stripsDashCompany(t *testing.T) {
	f, l := CleanPersonNames("Jane", "Doe - Acme", "", "Acme", nil)
	if f != "Jane" || l != "Doe" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNames_fromEmail(t *testing.T) {
	f, l := CleanPersonNames("jane.doe@example.com", "-", "", "Acme",
		[]string{"jane.doe@example.com"})
	if f != "Jane" || l != "Doe" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNames_lastIsCompany(t *testing.T) {
	f, l := CleanPersonNames("Jane", "Acme", "", "Acme GmbH & Co. KG", nil)
	if f != "Jane" || l != "-" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestFormatProjectTitle(t *testing.T) {
	got := FormatProjectTitle("DE", "Acme", "Map App")
	if got != "DE | Acme | Map App" {
		t.Fatalf("got %q", got)
	}
}
