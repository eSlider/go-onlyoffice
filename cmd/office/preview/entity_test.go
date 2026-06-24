package preview_test

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

func TestMailMarkdownConvertsHTMLBody(t *testing.T) {
	raw := map[string]any{
		"subject":  "Weekly update",
		"from":     `"Team" <team@example.com>`,
		"htmlBody": `<p>Hello <strong>world</strong></p><p><a href="https://example.com">Link</a></p>`,
	}
	md := preview.MailMarkdown(raw)
	if !strings.Contains(md, "Weekly update") {
		t.Fatalf("missing subject:\n%s", md)
	}
	if !strings.Contains(md, "team@example.com") {
		t.Fatalf("missing from:\n%s", md)
	}
	if strings.Contains(md, "<strong>") || strings.Contains(md, "<p>") {
		t.Fatalf("expected HTML converted, got raw tags:\n%s", md)
	}
	if !strings.Contains(md, "world") {
		t.Fatalf("missing body text:\n%s", md)
	}
}

func TestMailMarkdownRendersWithColor(t *testing.T) {
	md := preview.MailMarkdown(map[string]any{
		"subject":  "Styled",
		"from":     "a@b.com",
		"htmlBody": `<h2>Heading</h2><p>Plain text</p>`,
	})
	out, err := preview.RenderMarkdown(md, 60)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("empty render output")
	}
	if !strings.Contains(out, "Heading") || !strings.Contains(out, "Plain text") {
		t.Fatalf("missing rendered content:\n%s", out)
	}
}

func TestMailMarkdownPlainBodyFallback(t *testing.T) {
	md := preview.MailMarkdown(map[string]any{
		"subject": "Hi",
		"from":    "a@b.com",
		"body":    "Just plain text",
	})
	if !strings.Contains(md, "Just plain text") {
		t.Fatalf("missing plain body:\n%s", md)
	}
}
