//go:build integration

package onlyoffice

import (
	"context"
	"strings"
	"testing"
)

// TestIntegrationFolderPath resolves the real Fibu EDL folder chain
// (project root 522 → Eingangsrechnungen 647 → 2025 649) to titles.
func TestIntegrationFolderPath(t *testing.T) {
	creds := GetEnvironmentCredentials()
	if strings.TrimSpace(creds.Url) == "" || strings.TrimSpace(creds.User) == "" {
		t.Skip("no ONLYOFFICE_URL/USER credentials")
	}
	c := NewClient(creds)
	ctx := context.Background()

	path := c.FolderPath(ctx, []string{"522", "647", "649"})
	if len(path) != 3 {
		t.Fatalf("FolderPath returned %v, want 3 segments", path)
	}
	for i, seg := range path {
		if strings.TrimSpace(seg) == "" {
			t.Errorf("segment %d empty: %v", i, path)
		}
	}
	full := c.UniquePath(ctx, []string{"522", "647", "649"}, "Rechnung-x.pdf")
	if !strings.HasSuffix(full, "Rechnung-x.pdf") || !strings.Contains(full, "/") {
		t.Errorf("UniquePath = %q, want a slash-joined path ending in the file", full)
	}
	t.Logf("path=%v full=%q", path, full)
}
