package applications

import "testing"

func TestExtractField(t *testing.T) {
	text := "**Position**: Senior Dev\n**Company**: Acme\n"
	if g := extract(text, `(?i)\*\*Position\*\*:\s*(.+?)(?:\s{2,}|\n)`); g != "Senior Dev" {
		t.Fatalf("position: %q", g)
	}
	if g := extract(text, `(?i)\*\*Company\*\*:\s*(.+?)(?:\s{2,}|\n)`); g != "Acme" {
		t.Fatalf("company: %q", g)
	}
}

func TestParseSalary(t *testing.T) {
	if v := parseSalary("€50,000"); v < 40000 {
		t.Fatalf("annual: %v", v)
	}
	if v := parseSalary("400 / day"); v != 400*220 {
		t.Fatalf("daily: %v", v)
	}
}
