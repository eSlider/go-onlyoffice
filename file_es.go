package onlyoffice

// Elasticsearch backend of the unified file client (epic #34, F3 #37).
//
// OnlyOffice full-text search runs on Elasticsearch (index `files_file`, NEST
// client on the server). The REST endpoint GET /api/2.0/files/@search/{query}
// only searches file names in the database, so content search needs a direct
// ES query. The live server is Elasticsearch 7.16.3; the request shape below
// is plain REST and stays stdlib-only, matching the repo's no-extra-deps rule.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Canonical file/search model (epic #34, F1 #35). Declared here because F1 is
// not merged yet; move to file_core.go and drop these when it lands. Keep the
// shape identical to the contract in #35.
type (
	// Kind distinguishes a file from a folder.
	Kind int
	// Entry is a canonical file/folder record.
	Entry struct {
		ID       string
		ParentID string
		Title    string
		Kind     Kind
		Size     int64
		MIME     string
		Created  time.Time
		Modified time.Time
		Version  int
		Provider string
	}
	// SearchQuery is a backend-agnostic search request.
	SearchQuery struct {
		Text       string
		InContent  bool
		FolderID   string
		Extensions []string
		Limit      int
	}
	// SearchHit is a search result entry plus its relevance data.
	SearchHit struct {
		Entry
		Score     float64
		Highlight string
		Path      []string
	}
	// Searcher searches a document store by name and optionally content.
	Searcher interface {
		Search(ctx context.Context, q SearchQuery) ([]SearchHit, error)
		Name() string
	}
)

// Kind values (epic #34).
const (
	File Kind = iota
	Folder
)

const (
	defaultESIndex    = "files_file"
	defaultESLimit    = 20
	maxESLimit        = 200
	maxESResponseSize = 8 << 20
)

// ESConfig configures the direct Elasticsearch searcher.
type ESConfig struct {
	URL    string // scheme://host:port of the ES HTTP endpoint
	Index  string // index name, default files_file
	Tenant string // tenantId filter, empty means all tenants
}

// ESConfigFromEnv reads ONLYOFFICE_ES_URL, ONLYOFFICE_ES_INDEX (default
// files_file) and ONLYOFFICE_TENANT. The library never loads dotfiles — the
// CLI does that.
func ESConfigFromEnv() ESConfig {
	return ESConfig{
		URL:    strings.TrimRight(strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_URL")), "/"),
		Index:  firstNonEmpty(os.Getenv("ONLYOFFICE_ES_INDEX"), defaultESIndex),
		Tenant: strings.TrimSpace(os.Getenv("ONLYOFFICE_TENANT")),
	}
}

// ESSearcher queries OnlyOffice's Elasticsearch index directly for file name
// and document content.
type ESSearcher struct {
	cfg  ESConfig
	http *http.Client
}

// NewESSearcher returns a searcher for the OnlyOffice Elasticsearch index.
// The URL is required; an empty index falls back to files_file.
func NewESSearcher(cfg ESConfig) (*ESSearcher, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("onlyoffice: elasticsearch URL is empty (set ONLYOFFICE_ES_URL)")
	}
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	if cfg.Index == "" {
		cfg.Index = defaultESIndex
	}
	return &ESSearcher{cfg: cfg, http: &http.Client{Timeout: 30 * time.Second}}, nil
}

// Name implements Searcher.
func (s *ESSearcher) Name() string { return "elasticsearch" }

// Search runs a multi_match over title (and, when q.InContent is set,
// document.attachment.content), filtered by tenant and optional folder.
func (s *ESSearcher) Search(ctx context.Context, q SearchQuery) ([]SearchHit, error) {
	q.Text = strings.TrimSpace(q.Text)
	if q.Text == "" {
		return nil, fmt.Errorf("onlyoffice: empty search query")
	}
	body, err := json.Marshal(esSearchRequest(q, s.cfg.Tenant))
	if err != nil {
		return nil, fmt.Errorf("onlyoffice: build elasticsearch query: %w", err)
	}
	endpoint := s.cfg.URL + "/" + s.cfg.Index + "/_search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("onlyoffice: elasticsearch search: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxESResponseSize))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("onlyoffice: elasticsearch search: %d %s", resp.StatusCode, truncate(string(raw), 400))
	}
	return parseESSearchResponse(raw)
}

// esSearchRequest builds the ES query body. Pure, so it is unit-tested.
func esSearchRequest(q SearchQuery, tenant string) esRequest {
	limit := q.Limit
	if limit <= 0 {
		limit = defaultESLimit
	}
	if limit > maxESLimit {
		limit = maxESLimit
	}
	fields := []string{"title^2"}
	if q.InContent {
		fields = append(fields, "document.attachment.content")
	}
	must := []esClause{{MultiMatch: &esMultiMatch{Query: q.Text, Fields: fields}}}

	var filter []esClause
	if t := strings.TrimSpace(tenant); t != "" {
		filter = append(filter, esClause{Term: map[string]any{"tenantId": numericOrString(t)}})
	}
	if f := strings.TrimSpace(q.FolderID); f != "" {
		filter = append(filter, esClause{Term: map[string]any{"folders.folderId": f}})
	}
	for _, ext := range normalizeExtensions(q.Extensions) {
		filter = append(filter, esClause{Wildcard: map[string]any{"title": "*." + ext}})
	}

	highlightFields := map[string]struct{}{"title": {}}
	if q.InContent {
		highlightFields["document.attachment.content"] = struct{}{}
	}
	return esRequest{
		Size:      limit,
		Source:    []string{"id", "title", "folders"},
		Query:     esQuery{Bool: esBool{Must: must, Filter: filter}},
		Highlight: esHighlight{PreTags: []string{"<em>"}, PostTags: []string{"</em>"}, Fields: highlightFields},
	}
}

// normalizeExtensions lowercases, trims leading dots and drops empties.
func normalizeExtensions(exts []string) []string {
	out := make([]string, 0, len(exts))
	seen := map[string]bool{}
	for _, e := range exts {
		e = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(e), ".")))
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	return out
}

// numericOrString keeps an integer-looking filter value numeric (tenantId is
// a long) and leaves anything else as a string (folderId is a text token).
func numericOrString(s string) any {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return s
}

// esRequest is the subset of the ES query DSL this client emits.
type esRequest struct {
	Size      int         `json:"size"`
	Source    []string    `json:"_source"`
	Query     esQuery     `json:"query"`
	Highlight esHighlight `json:"highlight"`
}

type esQuery struct {
	Bool esBool `json:"bool"`
}

type esBool struct {
	Must   []esClause `json:"must,omitempty"`
	Filter []esClause `json:"filter,omitempty"`
}

type esClause struct {
	MultiMatch *esMultiMatch  `json:"multi_match,omitempty"`
	Term       map[string]any `json:"term,omitempty"`
	Wildcard   map[string]any `json:"wildcard,omitempty"`
}

type esMultiMatch struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
}

type esHighlight struct {
	PreTags  []string            `json:"pre_tags,omitempty"`
	PostTags []string            `json:"post_tags,omitempty"`
	Fields   map[string]struct{} `json:"fields"`
}

// esResponse is the subset of an ES search response we consume.
type esResponse struct {
	Took int `json:"took"`
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		Hits []esResponseHit `json:"hits"`
	} `json:"hits"`
}

type esResponseHit struct {
	ID     string  `json:"_id"`
	Score  float64 `json:"_score"`
	Source struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Folders []struct {
			FolderID string `json:"folderId"`
			ID       int    `json:"id"`
		} `json:"folders"`
	} `json:"_source"`
	Highlight map[string][]string `json:"highlight"`
}

// parseESSearchResponse converts an ES search response into SearchHit values.
// Pure, so it is unit-tested.
func parseESSearchResponse(raw []byte) ([]SearchHit, error) {
	var r esResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("onlyoffice: decode elasticsearch response: %w", err)
	}
	hits := make([]SearchHit, 0, len(r.Hits.Hits))
	for _, h := range r.Hits.Hits {
		id := strconv.Itoa(h.Source.ID)
		if h.Source.ID == 0 {
			id = h.ID
		}
		var parent string
		path := make([]string, 0, len(h.Source.Folders))
		for i, f := range h.Source.Folders {
			path = append(path, f.FolderID)
			if i == 0 {
				parent = f.FolderID
			}
		}
		hits = append(hits, SearchHit{
			Entry: Entry{
				ID:       id,
				ParentID: parent,
				Title:    h.Source.Title,
				Kind:     File,
				Provider: "elasticsearch",
			},
			Score:     h.Score,
			Highlight: esHighlightText(h.Highlight),
			Path:      path,
		})
	}
	return hits, nil
}

var esHighlightTag = regexp.MustCompile(`</?em[^>]*>`)

// esHighlightText flattens a highlight map into one plain-text snippet,
// preferring the content fragment over the title.
func esHighlightText(hl map[string][]string) string {
	for _, key := range []string{"document.attachment.content", "title"} {
		frags := hl[key]
		if len(frags) == 0 {
			continue
		}
		clean := make([]string, 0, len(frags))
		for _, f := range frags {
			clean = append(clean, esHighlightTag.ReplaceAllString(f, ""))
		}
		return strings.Join(clean, " … ")
	}
	return ""
}
