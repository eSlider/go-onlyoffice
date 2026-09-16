//go:build integration

package onlyoffice

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestIntegrationESTextIndex verifies the own full-text index end to end
// against a live Elasticsearch: create the index with its mapping, index a
// document, find it by content (and reject it via a folder filter and after
// deletion), then drop the throwaway index.
//
// Requires ONLYOFFICE_ES_URL (a reachable ES endpoint — in the current setup a
// tunnel to the ES inside the OnlyOffice VM, see docs/elasticsearch.md). It
// does not need OnlyOffice credentials because no file is downloaded: the
// TextIndexer write path is covered by unit tests with a fake extractor.
func TestIntegrationESTextIndex(t *testing.T) {
	esURL := strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_URL"))
	if esURL == "" {
		t.Skip("ONLYOFFICE_ES_URL not set — skipping Elasticsearch integration test")
	}
	stamp := time.Now().UTC().Format("20060102150405")
	idx, err := NewESTextIndex(ESTextConfig{URL: esURL, Index: "oo_docs_text_it_" + stamp})
	if err != nil {
		t.Fatalf("NewESTextIndex: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		_, _, _ = idx.do(cleanupCtx, http.MethodDelete, "/"+idx.Index(), nil, "")
	})

	if err := idx.Ensure(ctx); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	// Ensure is idempotent.
	if err := idx.Ensure(ctx); err != nil {
		t.Fatalf("Ensure (second): %v", err)
	}

	token := "gotes" + stamp
	doc := TextDoc{
		ID:       "3578",
		Title:    "2026-07-28-S1021-Edelweiss-rechnung.pdf",
		FolderID: "634",
		Ext:      "pdf",
		Content:  "Begleitzettel SGB XI — Rechnung " + token,
	}
	if err := idx.Put(ctx, []TextDoc{doc}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	hits, err := idx.Search(ctx, SearchQuery{Text: token})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != "3578" {
		t.Fatalf("content search hits = %+v, want doc 3578", hits)
	}
	if !strings.Contains(hits[0].Highlight, token) {
		t.Errorf("highlight = %q, want token", hits[0].Highlight)
	}

	if hits, err := idx.Search(ctx, SearchQuery{Text: token, FolderID: "999"}); err != nil {
		t.Fatalf("Search with folder filter: %v", err)
	} else if len(hits) != 0 {
		t.Errorf("folder filter returned %d hits, want 0", len(hits))
	}
	if hits, err := idx.Search(ctx, SearchQuery{Text: token, Extensions: []string{"docx"}}); err != nil {
		t.Fatalf("Search with ext filter: %v", err)
	} else if len(hits) != 0 {
		t.Errorf("ext filter returned %d hits, want 0", len(hits))
	}

	if err := idx.Delete(ctx, []string{"3578"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if hits, err := idx.Search(ctx, SearchQuery{Text: token}); err != nil {
		t.Fatalf("Search after delete: %v", err)
	} else if len(hits) != 0 {
		t.Errorf("after delete search returned %d hits, want 0", len(hits))
	}
}
