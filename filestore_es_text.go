package onlyoffice

// Own full-text index (epic #34, F6 #42).
//
// The OnlyOffice Elasticsearch index (files_file) only holds extracted content
// for Office formats. FileUtility.CanIndex gates extraction by the server
// setting files.index.formats, whose default is ".pptx|.xlsx|.docx", so PDFs
// are indexed by name only. Instead of patching the server (risky: lost on
// upgrade, forces a full reindex) this file implements a second, independent
// index (default oo_docs_text) that our own pipeline fills from
// internal/docpipe (pdftotext + OCR). The OnlyOffice index is never touched.
//
// See docs/elasticsearch.md for the decision and the trade-offs.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultESTextIndex = "oo_docs_text"

// ESTextConfig configures the own full-text index.
type ESTextConfig struct {
	URL    string // scheme://host:port of the ES HTTP endpoint
	Index  string // index name, default oo_docs_text
	Tenant string // reserved for future multi-tenant data; unused for now
}

// ESTextConfigFromEnv reads ONLYOFFICE_ES_URL and ONLYOFFICE_ES_TEXT_INDEX
// (default oo_docs_text). The library never loads dotfiles — the CLI does that.
func ESTextConfigFromEnv() ESTextConfig {
	return ESTextConfig{
		URL:    strings.TrimRight(strings.TrimSpace(os.Getenv("ONLYOFFICE_ES_URL")), "/"),
		Index:  firstNonEmpty(os.Getenv("ONLYOFFICE_ES_TEXT_INDEX"), defaultESTextIndex),
		Tenant: strings.TrimSpace(os.Getenv("ONLYOFFICE_TENANT")),
	}
}

// TextDoc is one document in the own full-text index. It is keyed by the
// OnlyOffice file id so hits map straight back to Documents entries.
type TextDoc struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	FolderID string `json:"folder,omitempty"`
	Ext      string `json:"ext,omitempty"`
	Content  string `json:"content"`
}

// TextIndex is the storage/search surface for locally extracted document text.
// It complements Searcher: ESSearcher reads OnlyOffice's index, ESTextIndex
// reads ours.
type TextIndex interface {
	Put(ctx context.Context, docs []TextDoc) error
	Delete(ctx context.Context, ids []string) error
	Search(ctx context.Context, q SearchQuery) ([]SearchHit, error)
	Name() string
}

// ESTextIndex is a TextIndex (and Searcher) over a dedicated Elasticsearch
// index filled by TextIndexer.
type ESTextIndex struct {
	cfg  ESTextConfig
	http *http.Client
}

var (
	_ TextIndex = (*ESTextIndex)(nil)
	_ Searcher  = (*ESTextIndex)(nil)
)

// NewESTextIndex returns a searcher/writer for the own full-text index. The URL
// is required; an empty index falls back to oo_docs_text.
func NewESTextIndex(cfg ESTextConfig) (*ESTextIndex, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("onlyoffice: elasticsearch URL is empty (set ONLYOFFICE_ES_URL)")
	}
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	if cfg.Index == "" {
		cfg.Index = defaultESTextIndex
	}
	return &ESTextIndex{cfg: cfg, http: &http.Client{Timeout: 120 * time.Second}}, nil
}

// Name implements Searcher and TextIndex.
func (x *ESTextIndex) Name() string { return "es-text" }

// Index returns the configured index name.
func (x *ESTextIndex) Index() string { return x.cfg.Index }

// esTextMapping pins explicit types: content must stay a plain text field (no
// keyword subfield) and folder/ext stay exact keywords for filters.
const esTextMapping = `{
  "mappings": {
    "properties": {
      "id":      {"type": "keyword"},
      "title":   {"type": "text", "fields": {"keyword": {"type": "keyword", "ignore_above": 512}}},
      "folder":  {"type": "keyword"},
      "ext":     {"type": "keyword"},
      "content": {"type": "text"}
    }
  }
}`

// Ensure creates the index with the explicit mapping. A missing index is
// created; an already existing one is left untouched.
func (x *ESTextIndex) Ensure(ctx context.Context) error {
	status, raw, err := x.do(ctx, http.MethodPut, "/"+x.cfg.Index, []byte(esTextMapping), "application/json")
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}
	if status == http.StatusBadRequest && bytes.Contains(raw, []byte("resource_already_exists_exception")) {
		return nil
	}
	return fmt.Errorf("onlyoffice: create text index %s: %d %s", x.cfg.Index, status, truncate(string(raw), 300))
}

// Put upserts documents via the bulk API and refreshes so they are immediately
// searchable.
func (x *ESTextIndex) Put(ctx context.Context, docs []TextDoc) error {
	if len(docs) == 0 {
		return nil
	}
	status, raw, err := x.do(ctx, http.MethodPost, "/"+x.cfg.Index+"/_bulk?refresh=true", esTextBulkBody(docs), "application/x-ndjson")
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("onlyoffice: bulk index %s: %d %s", x.cfg.Index, status, truncate(string(raw), 400))
	}
	var res esBulkResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return fmt.Errorf("onlyoffice: decode bulk response: %w", err)
	}
	if !res.Errors {
		return nil
	}
	return fmt.Errorf("onlyoffice: bulk index %s: %s", x.cfg.Index, res.firstError())
}

// Delete removes documents by OnlyOffice file id. A missing index means there
// is nothing to delete.
func (x *ESTextIndex) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	body, err := json.Marshal(map[string]any{"query": map[string]any{"terms": map[string]any{"id": ids}}})
	if err != nil {
		return err
	}
	status, raw, err := x.do(ctx, http.MethodPost, "/"+x.cfg.Index+"/_delete_by_query?refresh=true", body, "application/json")
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status >= 400 {
		return fmt.Errorf("onlyoffice: delete from %s: %d %s", x.cfg.Index, status, truncate(string(raw), 400))
	}
	return nil
}

// Search runs a multi_match over title (boosted) and content, with optional
// folder and extension filters. A missing index yields no hits, not an error.
func (x *ESTextIndex) Search(ctx context.Context, q SearchQuery) ([]SearchHit, error) {
	q.Text = strings.TrimSpace(q.Text)
	if q.Text == "" {
		return nil, fmt.Errorf("onlyoffice: empty search query")
	}
	body, err := json.Marshal(esTextSearchRequest(q))
	if err != nil {
		return nil, fmt.Errorf("onlyoffice: build elasticsearch query: %w", err)
	}
	status, raw, err := x.do(ctx, http.MethodPost, "/"+x.cfg.Index+"/_search", body, "application/json")
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status >= 400 {
		return nil, fmt.Errorf("onlyoffice: search %s: %d %s", x.cfg.Index, status, truncate(string(raw), 400))
	}
	return parseESTextResponse(raw)
}

// do sends one request and returns the status and body (bounded). The caller
// decides which statuses are errors.
func (x *ESTextIndex) do(ctx context.Context, method, path string, body []byte, contentType string) (int, []byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, x.cfg.URL+path, r)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := x.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("onlyoffice: elasticsearch %s: %w", method, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxESResponseSize))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, raw, nil
}

// esTextBulkBody renders the NDJSON bulk payload. Pure, so it is unit-tested.
func esTextBulkBody(docs []TextDoc) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, d := range docs {
		_ = enc.Encode(map[string]any{"index": map[string]any{"_id": d.ID}})
		_ = enc.Encode(d)
	}
	return b.Bytes()
}

// esTextSearchRequest builds the own-index query. Pure, so it is unit-tested.
func esTextSearchRequest(q SearchQuery) esRequest {
	limit := q.Limit
	if limit <= 0 {
		limit = defaultESLimit
	}
	if limit > maxESLimit {
		limit = maxESLimit
	}
	fields := []string{"title^2", "content"}
	must := []esClause{{MultiMatch: &esMultiMatch{Query: q.Text, Fields: fields}}}

	var filter []esClause
	if f := strings.TrimSpace(q.FolderID); f != "" {
		filter = append(filter, esClause{Term: map[string]any{"folder": f}})
	}
	if exts := normalizeExtensions(q.Extensions); len(exts) > 0 {
		filter = append(filter, esClause{Terms: map[string]any{"ext": exts}})
	}

	return esRequest{
		Size:      limit,
		Source:    []string{"id", "title", "folder", "ext"},
		Query:     esQuery{Bool: esBool{Must: must, Filter: filter}},
		Highlight: esHighlight{PreTags: []string{"<em>"}, PostTags: []string{"</em>"}, Fields: map[string]struct{}{"title": {}, "content": {}}},
	}
}

// esBulkResponse is the subset of an ES bulk response we consume.
type esBulkResponse struct {
	Errors bool `json:"errors"`
	Items  []map[string]struct {
		ID     string `json:"_id"`
		Status int    `json:"status"`
		Error  *struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"error"`
	} `json:"items"`
}

// firstError returns a compact description of the first failed bulk item.
func (r esBulkResponse) firstError() string {
	for _, item := range r.Items {
		for op, res := range item {
			if res.Error != nil {
				return fmt.Sprintf("%s %s: %s %s", op, res.ID, res.Error.Type, res.Error.Reason)
			}
		}
	}
	return "unknown bulk error"
}

// esTextResponse is the subset of an own-index search response we consume.
type esTextResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID     string              `json:"_id"`
			Score  float64             `json:"_score"`
			Source TextDoc             `json:"_source"`
			HL     map[string][]string `json:"highlight"`
		} `json:"hits"`
	} `json:"hits"`
}

// parseESTextResponse converts an own-index search response into SearchHit
// values. Pure, so it is unit-tested.
func parseESTextResponse(raw []byte) ([]SearchHit, error) {
	var r esTextResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("onlyoffice: decode elasticsearch response: %w", err)
	}
	hits := make([]SearchHit, 0, len(r.Hits.Hits))
	for _, h := range r.Hits.Hits {
		id := h.Source.ID
		if id == "" {
			id = h.ID
		}
		parent := h.Source.FolderID
		var path []string
		if parent != "" {
			path = []string{parent}
		}
		hits = append(hits, SearchHit{
			Entry: Entry{
				ID:       id,
				ParentID: parent,
				Title:    h.Source.Title,
				Kind:     File,
				Provider: "es-text",
			},
			Score:     h.Score,
			Highlight: esHighlightText(h.HL),
			Path:      path,
		})
	}
	return hits, nil
}
