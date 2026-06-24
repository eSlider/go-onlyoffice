package preview_test

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

func TestContactMarkdownCompany(t *testing.T) {
	md := preview.ContactMarkdown(map[string]any{
		"displayName": "Acme GmbH",
		"isCompany":   true,
		"about":       "Widget supplier",
	})
	if !strings.Contains(md, "Acme GmbH") {
		t.Fatalf("title missing: %q", md)
	}
	if !strings.Contains(md, "Company") {
		t.Fatalf("type label missing: %q", md)
	}
	if !strings.Contains(md, "Widget supplier") {
		t.Fatalf("about missing: %q", md)
	}
}

func TestOpportunityMarkdown(t *testing.T) {
	md := preview.OpportunityMarkdown(map[string]any{
		"title":       "Big Deal",
		"description": "Annual contract",
		"bidValue":    float64(50000),
		"bidCurrency": "EUR",
	})
	if !strings.Contains(md, "Big Deal") {
		t.Fatalf("title missing: %q", md)
	}
	if !strings.Contains(md, "50000") || !strings.Contains(md, "EUR") {
		t.Fatalf("bid missing: %q", md)
	}
}

func TestMailMarkdown(t *testing.T) {
	md := preview.MailMarkdown(map[string]any{
		"subject": "Hello",
		"from":    "Alice <alice@example.com>",
		"body":    "<p>Hi there</p>",
	})
	if !strings.Contains(md, "Hello") {
		t.Fatalf("subject missing: %q", md)
	}
	if !strings.Contains(md, "alice@example.com") {
		t.Fatalf("from missing: %q", md)
	}
	if strings.Contains(md, "<p>") {
		t.Fatalf("html not stripped: %q", md)
	}
	if !strings.Contains(md, "Hi there") {
		t.Fatalf("body text missing: %q", md)
	}
}

func TestEventMarkdown(t *testing.T) {
	md := preview.EventMarkdown(map[string]any{
		"title": "Standup",
		"start": "2026-06-24T09:00:00",
		"end":   "2026-06-24T09:15:00",
	})
	if !strings.Contains(md, "Standup") {
		t.Fatalf("title missing: %q", md)
	}
	if !strings.Contains(md, "2026-06-24") {
		t.Fatalf("dates missing: %q", md)
	}
}

func TestTaskMarkdown(t *testing.T) {
	md := preview.TaskMarkdown(map[string]any{
		"title":       "Deploy",
		"status":      "Open",
		"deadline":    "2026-07-01",
		"description": "Roll out v2",
	})
	if !strings.Contains(md, "Deploy") {
		t.Fatalf("title missing: %q", md)
	}
	if !strings.Contains(md, "Open") {
		t.Fatalf("status missing: %q", md)
	}
}
