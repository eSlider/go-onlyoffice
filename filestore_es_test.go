package onlyoffice

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestESSearchRequestNameOnly(t *testing.T) {
	got := esSearchRequest(SearchQuery{Text: "Rechnung"}, "1")
	if got.Size != defaultESLimit {
		t.Errorf("size = %d, want %d", got.Size, defaultESLimit)
	}
	if !reflect.DeepEqual(got.Source, []string{"id", "title", "folders"}) {
		t.Errorf("_source = %v", got.Source)
	}
	if len(got.Query.Bool.Must) != 1 || got.Query.Bool.Must[0].MultiMatch == nil {
		t.Fatalf("must = %+v, want one multi_match", got.Query.Bool.Must)
	}
	mm := got.Query.Bool.Must[0].MultiMatch
	if mm.Query != "Rechnung" {
		t.Errorf("query = %q", mm.Query)
	}
	if !reflect.DeepEqual(mm.Fields, []string{"title^2"}) {
		t.Errorf("fields = %v, want title only", mm.Fields)
	}
	if _, ok := got.Highlight.Fields["document.attachment.content"]; ok {
		t.Error("content highlight present without InContent")
	}
	if _, ok := got.Highlight.Fields["title"]; !ok {
		t.Error("title highlight missing")
	}
	if len(got.Query.Bool.Filter) != 1 || got.Query.Bool.Filter[0].Term["tenantId"] != int64(1) {
		t.Errorf("tenant filter = %+v, want numeric tenantId=1", got.Query.Bool.Filter)
	}
}

func TestESSearchRequestContentFields(t *testing.T) {
	got := esSearchRequest(SearchQuery{Text: "Mahnung", InContent: true}, "")
	mm := got.Query.Bool.Must[0].MultiMatch
	want := []string{"title^2", "document.attachment.content"}
	if !reflect.DeepEqual(mm.Fields, want) {
		t.Errorf("fields = %v, want %v", mm.Fields, want)
	}
	if _, ok := got.Highlight.Fields["document.attachment.content"]; !ok {
		t.Error("content highlight missing with InContent")
	}
	if len(got.Query.Bool.Filter) != 0 {
		t.Errorf("filter = %+v, want none without tenant/folder", got.Query.Bool.Filter)
	}
}

func TestESSearchRequestFiltersAndLimit(t *testing.T) {
	got := esSearchRequest(SearchQuery{
		Text:       "Storchen",
		FolderID:   "649",
		Extensions: []string{".PDF", "pdf", "docx"},
		Limit:      5000,
	}, "42")
	if got.Size != maxESLimit {
		t.Errorf("size = %d, want cap %d", got.Size, maxESLimit)
	}
	var tenant, folder, wildcards int
	for _, f := range got.Query.Bool.Filter {
		switch {
		case f.Term != nil && f.Term["tenantId"] != nil:
			tenant++
		case f.Nested != nil:
			folder++
			if f.Nested.Path != "folders" || f.Nested.Query.Term["folders.folderId"] != "649" {
				t.Errorf("folder filter = %+v, want nested folders term 649", f.Nested)
			}
		case f.Wildcard != nil:
			wildcards++
		}
	}
	if tenant != 1 || folder != 1 {
		t.Errorf("term filters tenant=%d folder=%d, want 1 each", tenant, folder)
	}
	if wildcards != 2 {
		t.Errorf("wildcard filters = %d, want deduped PDF+docx", wildcards)
	}
}

func TestESSearchRequestSubstringAndsTerms(t *testing.T) {
	got := esSearchRequest(SearchQuery{Text: "Rechnung 2025", Substring: true, FolderID: "522"}, "")
	if got.Query.Bool.Must[0].MultiMatch != nil {
		t.Fatalf("substring must not use multi_match: %+v", got.Query.Bool.Must)
	}
	if len(got.Query.Bool.Must) != 2 {
		t.Fatalf("must = %+v, want two ANDed wildcard terms", got.Query.Bool.Must)
	}
	want := []string{"*rechnung*", "*2025*"}
	for i, m := range got.Query.Bool.Must {
		if m.Wildcard == nil || m.Wildcard["title"] != want[i] {
			t.Errorf("must[%d] = %+v, want title wildcard %q", i, m, want[i])
		}
	}
	if len(got.Query.Bool.Filter) != 1 || got.Query.Bool.Filter[0].Nested == nil {
		t.Errorf("folder filter = %+v, want nested", got.Query.Bool.Filter)
	}
}

func TestEscapeWildcard(t *testing.T) {
	cases := map[string]string{"*rechnung*": "rechnung", "a?b\\c": "abc", "plain": "plain"}
	for in, want := range cases {
		if got := escapeWildcard(in); got != want {
			t.Errorf("escapeWildcard(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestESSearchRequestRejectsEmptyTextAtSearch(t *testing.T) {
	s, err := NewESSearcher(ESConfig{URL: "http://localhost:9200"})
	if err != nil {
		t.Fatalf("NewESSearcher: %v", err)
	}
	if _, err := s.Search(t.Context(), SearchQuery{Text: "  "}); err == nil {
		t.Error("empty query: want error")
	}
}

func TestNewESSearcherRequiresURL(t *testing.T) {
	if _, err := NewESSearcher(ESConfig{}); err == nil {
		t.Error("empty URL: want error")
	}
	s, err := NewESSearcher(ESConfig{URL: "http://es:9200/"})
	if err != nil {
		t.Fatalf("NewESSearcher: %v", err)
	}
	if s.cfg.Index != defaultESIndex {
		t.Errorf("index = %q, want %q", s.cfg.Index, defaultESIndex)
	}
	if s.cfg.URL != "http://es:9200" {
		t.Errorf("url = %q, want trimmed", s.cfg.URL)
	}
	if s.Name() != "elasticsearch" {
		t.Errorf("Name() = %q", s.Name())
	}
}

func TestNormalizeExtensions(t *testing.T) {
	got := normalizeExtensions([]string{" .PDF ", "pdf", "", "xlsx"})
	want := []string{"pdf", "xlsx"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeExtensions = %v, want %v", got, want)
	}
}

func TestParseESSearchResponse(t *testing.T) {
	raw := []byte(`{
	  "took": 12,
	  "hits": {
	    "total": {"value": 2, "relation": "eq"},
	    "hits": [
	      {
	        "_id": "2395",
	        "_score": 7.31,
	        "_source": {"id": 2395, "title": "Rechnung-4711.pdf",
	          "folders": [{"folderId": "438", "id": 0}, {"folderId": "11", "id": 0}]},
	        "highlight": {
	          "title": ["<em>Rechnung</em>-4711.pdf"],
	          "document.attachment.content": ["… Zahlung der <em>Rechnung</em> …"]
	        }
	      },
	      {
	        "_id": "2318",
	        "_score": 6.02,
	        "_source": {"id": 2318, "title": "Mahnung.pdf", "folders": []},
	        "highlight": {"title": ["<em>Mahnung</em>.pdf"]}
	      }
	    ]
	  }
	}`)
	hits, err := parseESSearchResponse(raw)
	if err != nil {
		t.Fatalf("parseESSearchResponse: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits = %d, want 2", len(hits))
	}
	h0 := hits[0]
	if h0.ID != "2395" || h0.Title != "Rechnung-4711.pdf" || h0.Kind != File {
		t.Errorf("hit0 entry = %+v", h0.Entry)
	}
	// folders is root → leaf; the immediate parent is the last entry.
	if h0.ParentID != "11" || !reflect.DeepEqual(h0.Path, []string{"438", "11"}) {
		t.Errorf("hit0 path = %v parent = %q, want parent 11", h0.Path, h0.ParentID)
	}
	if h0.Score != 7.31 {
		t.Errorf("hit0 score = %v", h0.Score)
	}
	if h0.Highlight != "… Zahlung der Rechnung …" {
		t.Errorf("hit0 highlight = %q, want content fragment", h0.Highlight)
	}
	if hits[1].Highlight != "Mahnung.pdf" {
		t.Errorf("hit1 highlight = %q, want title without tags", hits[1].Highlight)
	}
	if hits[1].ParentID != "" || len(hits[1].Path) != 0 {
		t.Errorf("hit1 path = %v", hits[1].Path)
	}
}

func TestESSearchRequestJSONShape(t *testing.T) {
	got := esSearchRequest(SearchQuery{Text: "Rechnung", InContent: true}, "1")
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := back["query"].(map[string]any)["bool"]; !ok {
		t.Errorf("query.bool missing: %s", b)
	}
}
