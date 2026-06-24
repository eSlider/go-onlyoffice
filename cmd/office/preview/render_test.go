package preview_test

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

func TestRenderMarkdown(t *testing.T) {
	out, err := preview.RenderMarkdown("# Hello\n\nWorld", 40)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("empty render output")
	}
	if !strings.Contains(strings.ToLower(out), "hello") {
		t.Fatalf("expected rendered heading: %q", out)
	}
}

func TestRenderMarkdownEmpty(t *testing.T) {
	out, err := preview.RenderMarkdown("", 40)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty, got %q", out)
	}
}
