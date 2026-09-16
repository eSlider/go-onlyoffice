package onlyoffice

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestESTextSearchRequestShape(t *testing.T) {
	got := esTextSearchRequest(SearchQuery{
		Text:       "S1021",
		FolderID:   "634",
		Extensions: []string{".PDF", "pdf"},
		Limit:      5,
	})
	if got.Size != 5 {
		t.Errorf("size = %d, want 5", got.Size)
	}
	mm := got.Query.Bool.Must[0].MultiMatch
	if mm == nil || !reflect.DeepEqual(mm.Fields, []string{"title^2", "content"}) {
		t.Fatalf("multi_match = %+v, want title^2 + content", mm)
	}
	if _, ok := got.Highlight.Fields["content"]; !ok {
		t.Error("content highlight missing")
	}
	if _, ok := got.Highlight.Fields["title"]; !ok {
		t.Error("title highlight missing")
	}
	var folder, exts int
	for _, f := range got.Query.Bool.Filter {
		switch {
		case f.Term != nil && f.Term["folder"] != nil:
			folder++
			if f.Term["folder"] != "634" {
				t.Errorf("folder term = %+v", f.Term)
			}
		case f.Terms != nil:
			exts++
			if !reflect.DeepEqual(f.Terms["ext"], []string{"pdf"}) {
				t.Errorf("ext terms = %+v, want deduped pdf", f.Terms)
			}
		}
	}
	if folder != 1 || exts != 1 {
		t.Errorf("filters folder=%d exts=%d, want 1 each", folder, exts)
	}
}

func TestESTextBulkBody(t *testing.T) {
	body := esTextBulkBody([]TextDoc{
		{ID: "3578", Title: "S1021.pdf", FolderID: "634", Ext: "pdf", Content: "Begleitzettel <S1021> & mehr"},
		{ID: "3579", Title: "S1023.pdf", Ext: "pdf", Content: "x"},
	})
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("bulk body has %d lines, want 4:\n%s", len(lines), body)
	}
	if !strings.Contains(lines[0], `"index"`) || !strings.Contains(lines[0], `"_id":"3578"`) {
		t.Errorf("action line = %q", lines[0])
	}
	if !strings.Contains(lines[1], `"content":"Begleitzettel <S1021> & mehr"`) {
		t.Errorf("source line should keep HTML unescaped, got %q", lines[1])
	}
	if !strings.Contains(lines[2], `"_id":"3579"`) {
		t.Errorf("second action line = %q", lines[2])
	}
}

func TestParseESTextResponse(t *testing.T) {
	raw := []byte(`{
	  "hits": {
	    "total": {"value": 1, "relation": "eq"},
	    "hits": [
	      {
	        "_id": "3578",
	        "_score": 3.21,
	        "_source": {"id": "3578", "title": "2026-07-28-S1021-acme-rechnung.pdf", "folder": "634", "ext": "pdf"},
	        "highlight": {"content": ["Begleitzettel … <em>S1021</em> …"]}
	      }
	    ]
	  }
	}`)
	hits, err := parseESTextResponse(raw)
	if err != nil {
		t.Fatalf("parseESTextResponse: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits = %d, want 1", len(hits))
	}
	h := hits[0]
	if h.ID != "3578" || h.Title != "2026-07-28-S1021-acme-rechnung.pdf" || h.Kind != File {
		t.Errorf("entry = %+v", h.Entry)
	}
	if h.ParentID != "634" || !reflect.DeepEqual(h.Path, []string{"634"}) {
		t.Errorf("path = %v parent = %q", h.Path, h.ParentID)
	}
	if h.Provider != "es-text" {
		t.Errorf("provider = %q", h.Provider)
	}
	if h.Highlight != "Begleitzettel … S1021 …" {
		t.Errorf("highlight = %q, want tags stripped", h.Highlight)
	}
}

func TestNewESTextIndexDefaults(t *testing.T) {
	if _, err := NewESTextIndex(ESTextConfig{}); err == nil {
		t.Error("empty URL: want error")
	}
	x, err := NewESTextIndex(ESTextConfig{URL: "http://es:9200/"})
	if err != nil {
		t.Fatalf("NewESTextIndex: %v", err)
	}
	if x.Index() != defaultESTextIndex {
		t.Errorf("index = %q, want %q", x.Index(), defaultESTextIndex)
	}
	if x.cfg.URL != "http://es:9200" {
		t.Errorf("url = %q, want trimmed", x.cfg.URL)
	}
	if x.Name() != "es-text" {
		t.Errorf("Name() = %q", x.Name())
	}
}

func TestESTextConfigFromEnvIndexDefault(t *testing.T) {
	t.Setenv("ONLYOFFICE_ES_URL", "http://es:9200/")
	t.Setenv("ONLYOFFICE_ES_TEXT_INDEX", "")
	cfg := ESTextConfigFromEnv()
	if cfg.Index != defaultESTextIndex {
		t.Errorf("index = %q, want %q", cfg.Index, defaultESTextIndex)
	}
}

func TestTextIndexerIndexEntries(t *testing.T) {
	store := &fakeStore{
		files: map[string][]byte{"1": []byte("PDFBYTES")},
	}
	idx := &fakeIndex{}
	ix := NewTextIndexer(store, idx)
	ix.Extractor = fakeExtractor{prefix: "TEXT "}

	res, err := ix.IndexEntries(context.Background(), []Entry{
		{ID: "1", Title: "Rechnung.PDF", ParentID: "649", Kind: File},
		{ID: "2", Title: "Tabelle.xlsx", ParentID: "649", Kind: File},
		{ID: "3", Title: "Unterordner", Kind: Folder},
	}, IndexOptions{})
	if err != nil {
		t.Fatalf("IndexEntries: %v", err)
	}
	if res.Scanned != 3 || res.Indexed != 1 || res.Skipped != 2 || res.Failed != 0 {
		t.Errorf("result = %+v, want scanned=3 indexed=1 skipped=2 failed=0", res)
	}
	if len(idx.docs) != 1 {
		t.Fatalf("indexed docs = %d, want 1", len(idx.docs))
	}
	got := idx.docs[0]
	want := TextDoc{ID: "1", Title: "Rechnung.PDF", FolderID: "649", Ext: "pdf", Content: "TEXT PDFBYTES"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("doc = %+v, want %+v", got, want)
	}
}

func TestTextIndexerRecordsExtractionFailure(t *testing.T) {
	store := &fakeStore{files: map[string][]byte{"1": []byte("x")}}
	idx := &fakeIndex{}
	ix := NewTextIndexer(store, idx)
	ix.Extractor = failingExtractor{}

	res, err := ix.IndexEntries(context.Background(), []Entry{{ID: "1", Title: "a.pdf", Kind: File}}, IndexOptions{})
	if err != nil {
		t.Fatalf("IndexEntries: %v", err)
	}
	if res.Indexed != 0 || res.Failed != 1 || len(res.Errors) != 1 {
		t.Errorf("result = %+v, want one failure recorded", res)
	}
}

func TestTextIndexerPlanFolder(t *testing.T) {
	store := &fakeStore{dirs: map[string][]Entry{
		"root": {
			{ID: "10", Title: "a.pdf", Kind: File},
			{ID: "11", Title: "sub", Kind: Folder},
		},
		"11": {
			{ID: "12", Title: "b.PDF", Kind: File},
			{ID: "13", Title: "c.xlsx", Kind: File},
		},
	}}
	ix := NewTextIndexer(store, &fakeIndex{})

	flat, err := ix.PlanFolder(context.Background(), "root", IndexOptions{})
	if err != nil {
		t.Fatalf("PlanFolder: %v", err)
	}
	if len(flat) != 1 || flat[0].ID != "10" {
		t.Errorf("flat plan = %+v, want only a.pdf", flat)
	}
	deep, err := ix.PlanFolder(context.Background(), "root", IndexOptions{Recursive: true, Limit: 10})
	if err != nil {
		t.Fatalf("PlanFolder recursive: %v", err)
	}
	if len(deep) != 2 {
		t.Errorf("recursive plan = %d entries, want 2", len(deep))
	}
}

// --- fakes -----------------------------------------------------------------

type fakeStore struct {
	dirs  map[string][]Entry
	files map[string][]byte
	stat  map[string]Entry
}

func (f *fakeStore) Name() string { return "fake" }

func (f *fakeStore) List(_ context.Context, parentID string) ([]Entry, error) {
	return f.dirs[parentID], nil
}

func (f *fakeStore) Stat(_ context.Context, id string) (Entry, error) {
	if e, ok := f.stat[id]; ok {
		return e, nil
	}
	return Entry{}, fmt.Errorf("not found: %s", id)
}

func (f *fakeStore) Download(_ context.Context, id string, w io.Writer) (int64, error) {
	b, ok := f.files[id]
	if !ok {
		return 0, fmt.Errorf("no bytes for %s", id)
	}
	n, err := w.Write(b)
	return int64(n), err
}

func (f *fakeStore) CreateFolder(context.Context, string, string) (Entry, error) {
	return Entry{}, nil
}
func (f *fakeStore) Upload(context.Context, string, string, io.Reader) (Entry, error) {
	return Entry{}, nil
}
func (f *fakeStore) Move(context.Context, []string, string) error { return nil }
func (f *fakeStore) Copy(context.Context, []string, string) error { return nil }
func (f *fakeStore) Rename(context.Context, string, string) error { return nil }
func (f *fakeStore) Delete(context.Context, []string) error       { return nil }

type fakeIndex struct{ docs []TextDoc }

func (f *fakeIndex) Put(_ context.Context, docs []TextDoc) error {
	f.docs = append(f.docs, docs...)
	return nil
}
func (f *fakeIndex) Delete(context.Context, []string) error                   { return nil }
func (f *fakeIndex) Search(context.Context, SearchQuery) ([]SearchHit, error) { return nil, nil }
func (f *fakeIndex) Name() string                                             { return "fake" }

type fakeExtractor struct{ prefix string }

func (f fakeExtractor) Extract(path, _, _ string, _ int) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return f.prefix + string(b), nil
}

type failingExtractor struct{}

func (failingExtractor) Extract(string, string, string, int) (string, error) {
	return "", fmt.Errorf("boom")
}
