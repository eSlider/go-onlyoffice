package ui

import (
	"strings"
	"testing"
)

func TestScrollbarMetricsHiddenWhenFits(t *testing.T) {
	show, _, _ := ScrollbarMetrics(5, 10, 0)
	if show {
		t.Fatal("expected no scrollbar when content fits")
	}
}

func TestScrollbarMetricsVisibleWhenOverflow(t *testing.T) {
	show, start, end := ScrollbarMetrics(100, 10, 50)
	if !show || end <= start {
		t.Fatalf("expected thumb, got %d..%d", start, end)
	}
}

func TestApplyVerticalScrollbarAddsColumn(t *testing.T) {
	view := "line1\nline2\nline3"
	out := ApplyVerticalScrollbar(view, 12, 3, 10, 0)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("lines=%d", len(lines))
	}
	if !strings.HasSuffix(lines[0], "█") && !strings.HasSuffix(lines[0], "│") {
		t.Fatalf("expected scrollbar glyph: %q", lines[0])
	}
}
