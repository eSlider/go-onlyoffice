package preview_test

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

func TestCSVToMarkdownTable(t *testing.T) {
	csv := "name,score\nAlice,10\nBob,20\n"
	md, err := preview.CSVToMarkdownTable([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "name") || !strings.Contains(md, "Alice") {
		t.Fatalf("table missing data: %q", md)
	}
	if !strings.Contains(md, "|") {
		t.Fatalf("expected pipe table: %q", md)
	}
}

func TestJSONToMarkdown(t *testing.T) {
	md, err := preview.JSONToMarkdown([]byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "```") || !strings.Contains(md, `"a"`) {
		t.Fatalf("expected fenced json: %q", md)
	}
}
