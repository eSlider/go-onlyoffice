package preview_test

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

func TestHTMLToMarkdown(t *testing.T) {
	html := `<h1>Title</h1><p>Body text</p>`
	md, err := preview.HTMLToMarkdown(html)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Title") || !strings.Contains(md, "Body text") {
		t.Fatalf("conversion failed: %q", md)
	}
}
