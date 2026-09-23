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

// Default known fixtures for TestIntegrationESFacadeUsesES on the live index.
const (
	defaultESTestQuery     = "Rechnung_986-2025.pdf"
	defaultESTestSubstring = "rechnung 2025"
	defaultESTestFolder    = "522"
)

// TestIntegrationESFacadeUsesES proves that the public search path — `oo search`
// and Client.Files().Search() — really runs against the OnlyOffice
// Elasticsearch backend and not the REST @search endpoint, which only looks at
// file names in the database and is not a Searcher at all (see
// docs/elasticsearch.md). It pins the concrete backend and checks that a known
// document comes back with a non-empty id and folder path.
//
// Requires ONLYOFFICE_ES_URL only — the query never touches the REST API, so no
// OnlyOffice credentials are needed. Skips when it is missing. The fixture is
// overridable with ONLYOFFICE_ES_TEST_QUERY, ONLYOFFICE_ES_TEST_TITLE,
// ONLYOFFICE_ES_TEST_SUBSTRING and ONLYOFFICE_ES_TEST_FOLDER.
func TestIntegrationESFacadeUsesES(t *testing.T) {
	esURL := strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_URL"))
	if esURL == "" {
		t.Skip("ONLYOFFICE_ES_URL not set — skipping Elasticsearch integration test")
	}
	query := firstNonEmpty(strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_TEST_QUERY")), defaultESTestQuery)
	wantTitle := firstNonEmpty(strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_TEST_TITLE")), query)
	substring := firstNonEmpty(strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_TEST_SUBSTRING")), defaultESTestSubstring)
	folder := firstNonEmpty(strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_TEST_FOLDER")), defaultESTestFolder)

	c := NewClient(Credentials{})
	searcher, err := c.Files().Search()
	if err != nil {
		t.Fatalf("Files().Search(): %v", err)
	}
	if got := searcher.Name(); got != ProviderES {
		t.Fatalf("searcher.Name() = %q, want %q (REST @search is not a Searcher)", got, ProviderES)
	}
	if _, ok := searcher.(*ESSearcher); !ok {
		t.Fatalf("searcher = %T, want *ESSearcher (ES backend, not REST)", searcher)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Known file name: multi_match over title, as `oo search <file>` does.
	start := time.Now()
	hits, err := searcher.Search(ctx, SearchQuery{Text: query, Limit: 20})
	if err != nil {
		t.Fatalf("Search(%q): %v", query, err)
	}
	t.Logf("ES facade query %q: %d hits in %s", query, len(hits), time.Since(start))

	known := findHitByTitle(hits, wantTitle)
	if known == nil {
		t.Fatalf("query %q returned %d hits, none titled %q", query, len(hits), wantTitle)
	}
	if strings.TrimSpace(known.ID) == "" {
		t.Errorf("hit %q has empty id", known.Title)
	}
	if len(known.Path) == 0 {
		t.Errorf("hit %q has empty path", known.Title)
	}
	if known.Provider != ProviderES {
		t.Errorf("hit provider = %q, want %q", known.Provider, ProviderES)
	}

	// Substring + folder subtree, as `oo search <terms> --substring --folder N`
	// does: wildcard terms ANDed together, scoped to the folder's subtree.
	start = time.Now()
	subHits, err := searcher.Search(ctx, SearchQuery{Text: substring, Substring: true, FolderID: folder, Limit: 200})
	if err != nil {
		t.Fatalf("substring Search(%q, folder %s): %v", substring, folder, err)
	}
	t.Logf("ES facade substring %q folder %s: %d hits in %s", substring, folder, len(subHits), time.Since(start))
	if len(subHits) == 0 {
		t.Fatalf("substring query %q in folder %s returned no hits", substring, folder)
	}
	if findHitByTitle(subHits, wantTitle) == nil {
		t.Errorf("substring query %q in folder %s did not return %q", substring, folder, wantTitle)
	}
}

// findHitByTitle returns the first hit whose title matches, case-insensitively.
func findHitByTitle(hits []SearchHit, title string) *SearchHit {
	for i := range hits {
		if strings.EqualFold(strings.TrimSpace(hits[i].Title), title) {
			return &hits[i]
		}
	}
	return nil
}

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
