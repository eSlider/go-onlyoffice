//go:build integration

package onlyoffice

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestIntegrationMinioFallback downloads known stale-S3 files through the local
// MinIO fallback. Requires ONLYOFFICE_URL/USER/PASS (as all integration tests),
// MINIO_ACCESS_KEY/MINIO_SECRET_KEY and MINIO_TEST_FILE_IDS="3785,3859,3666";
// skips when any of those are missing.
func TestIntegrationMinioFallback(t *testing.T) {
	if os.Getenv("MINIO_ACCESS_KEY") == "" || os.Getenv("MINIO_SECRET_KEY") == "" {
		t.Skip("MINIO_ACCESS_KEY/MINIO_SECRET_KEY not set — skipping integration test")
	}
	raw := strings.TrimSpace(os.Getenv("MINIO_TEST_FILE_IDS"))
	if raw == "" {
		t.Skip("MINIO_TEST_FILE_IDS not set — skipping integration test")
	}
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		n, err := c.DownloadFile(ctx, id, io.Discard)
		if err != nil {
			t.Errorf("DownloadFile(%s): %v", id, err)
			continue
		}
		if n == 0 {
			t.Errorf("DownloadFile(%s): 0 bytes", id)
		} else {
			t.Logf("DownloadFile(%s): %d bytes", id, n)
		}
	}
}
