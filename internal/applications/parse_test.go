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

// TestBuildSummaryFitAssessment covers the RE2-safe replacement of a previously
// lookahead-based regex that would panic at MustCompile time on Go regexp.
func TestBuildSummaryFitAssessment(t *testing.T) {
	app := Data{Position: "SRE", Company: "Acme", Folder: "/tmp"}
	withNextSection := "## Fit Assessment\nGreat fit overall.\nStrong background.\n## Next section\nUnrelated."
	summary := buildSummary(app, withNextSection)
	if !contains(summary, "Fit Assessment:") {
		t.Fatalf("missing header in summary:\n%s", summary)
	}
	if !contains(summary, "Great fit overall.") {
		t.Fatalf("missing body in summary:\n%s", summary)
	}
	if contains(summary, "Unrelated.") {
		t.Fatalf("bled into next section:\n%s", summary)
	}
	eof := "## Fit Assessment\nOnly content at EOF."
	summary = buildSummary(app, eof)
	if !contains(summary, "Only content at EOF.") {
		t.Fatalf("EOF case missing body:\n%s", summary)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
