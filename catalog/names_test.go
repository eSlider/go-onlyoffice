package catalog

import "testing"

func TestCleanPersonNamesStripsCompanyParen(t *testing.T) {
	f, l := CleanPersonNames("John", "Smith (Acme)", "John Smith (Acme)", "Acme", nil)
	if f != "John" || l != "Smith" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNamesStripsDashCompany(t *testing.T) {
	f, l := CleanPersonNames("Jens", "Meyer - Acme", "", "Acme", nil)
	if f != "Jens" || l != "Meyer" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNamesFromEmail(t *testing.T) {
	f, l := CleanPersonNames("david.patzke@acme.example", "-", "", "Acme",
		[]string{"david.patzke@acme.example"})
	if f != "David" || l != "Patzke" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNamesLastIsCompany(t *testing.T) {
	f, l := CleanPersonNames("Thorsten", "Acme", "", "Acme GmbH & Co. KG", nil)
	if f != "Thorsten" || l != "-" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestFormatProjectTitle(t *testing.T) {
	if got := FormatProjectTitle("DE", "Acme", "Golang"); got != "DE | Acme | Golang" {
		t.Fatalf("got %q", got)
	}
	if got := FormatProjectTitle("", "Acme", ""); got != "Acme" {
		t.Fatalf("got %q", got)
	}
}
