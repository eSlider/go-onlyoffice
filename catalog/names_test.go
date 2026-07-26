package catalog

import "testing"

func TestCleanPersonNames_stripsCompanyParen(t *testing.T) {
	f, l := CleanPersonNames("Jörg", "Thomsen (WhereGroup)", "Jörg Thomsen (WhereGroup)", "WhereGroup", nil)
	if f != "Jörg" || l != "Thomsen" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNames_stripsDashCompany(t *testing.T) {
	f, l := CleanPersonNames("Jens", "Schaefermeyer - WhereGroup", "", "WhereGroup", nil)
	if f != "Jens" || l != "Schaefermeyer" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNames_fromEmail(t *testing.T) {
	f, l := CleanPersonNames("david.patzke@wheregroup.com", "-", "", "WhereGroup",
		[]string{"david.patzke@wheregroup.com"})
	if f != "David" || l != "Patzke" {
		t.Fatalf("got %q %q", f, l)
	}
}

func TestCleanPersonNames_lastIsCompany(t *testing.T) {
	f, l := CleanPersonNames("Thorsten", "WhereGroup", "", "WhereGroup GmbH & Co. KG", nil)
	if f != "Thorsten" || l != "-" {
		t.Fatalf("got %q %q", f, l)
	}
}
