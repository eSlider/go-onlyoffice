package onlyoffice

import "testing"

func TestIsCompany(t *testing.T) {
	if IsCompany(map[string]any{"isCompany": true}) != true {
		t.Fatal("true row not detected")
	}
	if IsCompany(map[string]any{"isCompany": false}) {
		t.Fatal("false row detected as company")
	}
	if IsCompany(map[string]any{}) {
		t.Fatal("missing field detected as company")
	}
	if IsCompany(nil) {
		t.Fatal("nil row detected as company")
	}
}

func TestNumericIDLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"9", "10", true},
		{"1747", "1748", true},
		{"abc", "abd", true},
		{"10", "9", false},
		{" 12 ", "13", true},
		{"x1", "2", false}, // non-numeric falls back lexical: "x1" > "2"
	}
	for _, c := range cases {
		if got := NumericIDLess(c.a, c.b); got != c.want {
			t.Errorf("NumericIDLess(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestSortIDs(t *testing.T) {
	ids := []string{"20", "3", "100", "1"}
	SortIDs(ids)
	want := "1 3 20 100"
	got := ""
	for i, id := range ids {
		if i > 0 {
			got += " "
		}
		got += id
	}
	if got != want {
		t.Fatalf("SortIDs=%q want %q", got, want)
	}
}

func TestContactID(t *testing.T) {
	if ContactID(map[string]any{"id": float64(42)}) != "42" {
		t.Fatal("numeric id formatting broken")
	}
}

func TestHistoryEntityConstants(t *testing.T) {
	if HistoryEntityOpportunity != "opportunity" || HistoryEntityCase != "case" {
		t.Fatal("history entity whitelist drifted from live-verified values")
	}
}
