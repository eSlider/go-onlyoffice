// Package search provides a client for the produktor SearXNG instance
// at https://search.produktor.io.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://search.produktor.io"

// Client queries the produktor SearXNG instance.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a client with sensible defaults.
func NewClient() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Options are optional SearXNG query parameters.
type Options struct {
	Categories string // e.g. "general", "images"
	Engines    string // e.g. "duckduckgo,wikipedia"
	Language   string // e.g. "en", "de"
	Page       int    // 1-based page number
	SafeSearch int    // 0=off, 1=moderate, 2=strict
}

// Result is one search hit.
type Result struct {
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Engine        string   `json:"engine"`
	Engines       []string `json:"engines"`
	Score         float64  `json:"score"`
	Category      string   `json:"category"`
	PublishedDate *string  `json:"publishedDate"`
	Thumbnail     string   `json:"thumbnail"`
}

// Response is the JSON envelope from /search?format=json.
type Response struct {
	Query               string     `json:"query"`
	Results             []Result   `json:"results"`
	Suggestions         []string   `json:"suggestions"`
	Answers             []any      `json:"answers"`
	Corrections         []any      `json:"corrections"`
	Infoboxes           []any      `json:"infoboxes"`
	UnresponsiveEngines [][]string `json:"unresponsive_engines"`
}

// Search runs a query and decodes the JSON response.
func (c *Client) Search(ctx context.Context, query string, opts *Options) (*Response, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search: empty query")
	}

	base := strings.TrimRight(c.baseURL(), "/")
	u, err := url.Parse(base + "/search")
	if err != nil {
		return nil, fmt.Errorf("search: parse url: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	q.Set("format", "json")
	if opts != nil {
		if opts.Categories != "" {
			q.Set("categories", opts.Categories)
		}
		if opts.Engines != "" {
			q.Set("engines", opts.Engines)
		}
		if opts.Language != "" {
			q.Set("language", opts.Language)
		}
		if opts.Page > 0 {
			q.Set("pageno", fmt.Sprintf("%d", opts.Page))
		}
		if opts.SafeSearch > 0 {
			q.Set("safesearch", fmt.Sprintf("%d", opts.SafeSearch))
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("search: new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-produktor-search/1.0")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("search: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("search: decode json: %w", err)
	}
	return &out, nil
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
