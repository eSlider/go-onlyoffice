package onlyoffice

// Lightweight JSON/form/multipart helpers complementing the typed Query()
// abstraction in onlyoffice.go. These are used by the Calendar, CRM, and
// subtask helpers that talk to non-JSON endpoints (form-encoded or multipart).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ensureToken refreshes the authentication token when missing or expired.
// Mirrors the logic inline in Query() but is safe to call from helpers that
// bypass the typed Request abstraction.
func (c *Client) ensureToken() error {
	if c.token != nil && !time.Time(c.token.Expires).Before(time.Now()) {
		return nil
	}
	tok, err := c.Auth(c.credentials)
	if err != nil {
		return err
	}
	c.token = tok
	return nil
}

// authHeader returns the value for the Authorization header, ensuring a token.
func (c *Client) authHeader() (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}
	return c.token.Value, nil
}

// Authenticate validates credentials and primes the token. Library users may
// call this eagerly to surface auth errors at startup; otherwise the token is
// fetched lazily on the first request.
//
// Prefer AuthenticateContext in long-running jobs — it honours cancellation.
func (c *Client) Authenticate() error { return c.ensureToken() }

// AuthenticateContext is the context-aware variant of Authenticate. If the
// cached token is still valid it returns immediately; otherwise it performs a
// POST to /api/2.0/authentication.json that is cancellable via ctx.
//
// This is the recommended entry point for long-running syncs (cron, watchers)
// because it guarantees that a stalled auth call will not block the caller
// past its deadline.
func (c *Client) AuthenticateContext(ctx context.Context) error {
	if c.token != nil && !time.Time(c.token.Expires).Before(time.Now()) {
		return nil
	}
	body, err := json.Marshal(c.credentials)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+"/api/2.0/authentication.json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("auth: %d %s", resp.StatusCode, truncate(string(raw), 400))
	}
	var env struct {
		Response *Token `json:"response"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("auth decode: %w", err)
	}
	if env.Response == nil || env.Response.Value == "" {
		return fmt.Errorf("auth: empty token in response")
	}
	c.token = env.Response
	return nil
}

// InvalidateToken clears the cached authentication token. The next request
// (or call to Authenticate / AuthenticateContext) will re-authenticate.
//
// Use this to recover from a mid-sync 401 when the server has revoked or
// rotated the session while the Expires timestamp still looks fresh locally.
func (c *Client) InvalidateToken() { c.token = nil }

// baseURL returns the configured base URL without trailing slash.
func (c *Client) baseURL() string {
	return strings.TrimRight(c.credentials.Url, "/")
}

// truncate crops s to n runes, appending "..." when truncated. Used in error
// messages to keep OnlyOffice HTML payloads readable.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// responseField extracts a top-level JSON key (for example "response") from a
// raw OnlyOffice envelope. Returns an error when the key is missing.
func responseField(raw json.RawMessage, key string) (json.RawMessage, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	v, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("response missing %q", key)
	}
	return v, nil
}

// ResponseArray executes a GET and returns the "response" field as []map.
// Returns (nil, nil) when the field is JSON null.
func (c *Client) ResponseArray(ctx context.Context, path string) ([]map[string]any, error) {
	raw, err := c.getJSON(ctx, path)
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	if string(resp) == "null" {
		return nil, nil
	}
	var list []map[string]any
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// getJSON issues an authenticated GET and returns the raw response body.
func (c *Client) getJSON(ctx context.Context, path string) (json.RawMessage, error) {
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s: %d %s", path, resp.StatusCode, truncate(string(raw), 400))
	}
	return raw, nil
}

// postForm issues an authenticated POST with application/x-www-form-urlencoded body.
func (c *Client) postForm(ctx context.Context, path string, fields url.Values) (json.RawMessage, error) {
	return c.formRequest(ctx, http.MethodPost, path, fields)
}

// putForm issues an authenticated PUT with application/x-www-form-urlencoded body.
func (c *Client) putForm(ctx context.Context, path string, fields url.Values) (json.RawMessage, error) {
	return c.formRequest(ctx, http.MethodPut, path, fields)
}

func (c *Client) formRequest(ctx context.Context, method, path string, fields url.Values) (json.RawMessage, error) {
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, strings.NewReader(fields.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s form %s: %d %s", method, path, resp.StatusCode, truncate(string(raw), 400))
	}
	return raw, nil
}

// deleteReq issues an authenticated DELETE.
func (c *Client) deleteReq(ctx context.Context, path string) (json.RawMessage, error) {
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL()+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DELETE %s: %d %s", path, resp.StatusCode, truncate(string(raw), 400))
	}
	return raw, nil
}

// SelfUserID returns the ID of the authenticated user (people/@self), cached.
func (c *Client) SelfUserID(ctx context.Context) (string, error) {
	if c.selfID != "" {
		return c.selfID, nil
	}
	raw, err := c.getJSON(ctx, "/api/2.0/people/@self.json")
	if err != nil {
		return "", err
	}
	var env struct {
		Response struct {
			ID string `json:"id"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	c.selfID = env.Response.ID
	return c.selfID, nil
}

// uploadMultipart posts a single file to path under the given form field name.
func (c *Client) uploadMultipart(ctx context.Context, path, fieldName, filePath string) (json.RawMessage, error) {
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upload %s: %d %s", path, resp.StatusCode, truncate(string(raw), 400))
	}
	return raw, nil
}
