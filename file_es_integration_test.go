//go:build integration

package onlyoffice

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TestIntegrationESSearch uploads a throwaway workbook and verifies that the
// direct Elasticsearch search finds it by file name and by content.
//
// The content index (document.attachment.content) is only populated for Office
// formats (docx/xlsx/pptx), so the fixture is an xlsx whose cell carries a
// unique token. Requires ONLYOFFICE_ES_URL (a reachable ES endpoint — in the
// current setup a tunnel to the ES inside the OnlyOffice VM, see
// docs/elasticsearch.md) plus the regular REST credentials for the upload.
// Skips when either is missing.
func TestIntegrationESSearch(t *testing.T) {
	esURL := strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_URL"))
	if esURL == "" {
		t.Skip("ONLYOFFICE_ES_URL not set — skipping Elasticsearch integration test")
	}
	c := liveClient(t)
	t.Cleanup(func() { cleanupTestProjects(t, c) })

	stamp := time.Now().UTC().Format("20060102-150405")
	nameToken := "goesname" + stamp
	contentToken := "goescontent" + stamp

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	project, err := c.CreateProject(NewProjectRequest{
		Title:       testProjectPrefix + "es-" + stamp,
		Description: "go-onlyoffice elasticsearch integration",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.ID == nil {
		t.Fatal("created project without id")
	}
	pid := strconv.Itoa(*project.ID)

	title := nameToken + ".xlsx"
	localPath := filepath.Join(t.TempDir(), title)
	book := excelize.NewFile()
	if err := book.SetCellValue("Sheet1", "A1", "OnlyOffice Elasticsearch content fixture "+contentToken); err != nil {
		t.Fatalf("SetCellValue: %v", err)
	}
	if err := book.SaveAs(localPath); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	entry, err := c.UploadProjectFile(ctx, pid, localPath)
	if err != nil {
		t.Fatalf("UploadProjectFile: %v", err)
	}
	fileID := strconv.Itoa(int(FileEntryNumericID(entry)))
	if fileID == "0" {
		t.Fatalf("upload returned no file id: %+v", entry)
	}

	es, err := NewESSearcher(ESConfig{
		URL:    esURL,
		Index:  os.Getenv("ONLYOFFICE_ES_INDEX"),
		Tenant: os.Getenv("ONLYOFFICE_TENANT"),
	})
	if err != nil {
		t.Fatalf("NewESSearcher: %v", err)
	}

	// Indexing is asynchronous on the server; poll until the file shows up.
	// The server's title analyzer splits on whitespace, so the name query is
	// the full file name token (including extension), as a user would type it.
	nameHit := waitForHit(t, ctx, es, SearchQuery{Text: title}, fileID)
	if nameHit.Title != title {
		t.Errorf("name hit title = %q, want %q", nameHit.Title, title)
	}
	contentHit := waitForHit(t, ctx, es, SearchQuery{Text: contentToken, InContent: true}, fileID)
	if contentHit.Highlight == "" {
		t.Error("content hit has no highlight fragment")
	}
	if !strings.Contains(contentHit.Title, nameToken) {
		t.Errorf("content hit title = %q, want the uploaded workbook", contentHit.Title)
	}

	// The content token is absent from the title, so a name-only search must
	// not return the file — this proves the content field is really queried.
	if hits := searchQuiet(t, es, SearchQuery{Text: contentToken}); len(hits) != 0 {
		t.Errorf("name-only search for content token returned %d hits, want 0", len(hits))
	}
}

// waitForHit polls ES until the file with fileID appears and returns that hit.
func waitForHit(t *testing.T, ctx context.Context, s *ESSearcher, q SearchQuery, fileID string) SearchHit {
	t.Helper()
	var lastErr error
	for {
		hits, err := s.Search(ctx, q)
		if err != nil {
			lastErr = err
		} else {
			for _, h := range hits {
				if h.ID == fileID {
					return h
				}
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("search %q: file %s not indexed in time (last err: %v)", q.Text, fileID, lastErr)
		case <-time.After(3 * time.Second):
		}
	}
}

func searchQuiet(t *testing.T, s *ESSearcher, q SearchQuery) []SearchHit {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	hits, err := s.Search(ctx, q)
	if err != nil {
		t.Fatalf("Search(%q): %v", q.Text, err)
	}
	return hits
}
