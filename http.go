package onlyoffice

// Transport-layer helpers used by the untyped domain methods (CRM, calendar,
// tasks, files). Authentication lives in auth.go; the typed Request/Query
// abstraction lives in request.go. These helpers deliberately share the
// `*Client` state with the typed path so token refresh, base URL, and
// self-id caching are handled once.

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
)

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

// ResponseObject executes a GET and decodes the "response" field into a map.
// Returns (nil, nil) when the field is JSON null or absent. Companion to
// ResponseArray — factored out to eliminate the ~15 identical decode blocks
// in crm.go / tasks.go / calendar.go.
func (c *Client) ResponseObject(ctx context.Context, path string) (map[string]any, error) {
	raw, err := c.getJSON(ctx, path)
	if err != nil {
		return nil, err
	}
	return unmarshalResponseObject(raw)
}

// postFormObject issues an authenticated POST (form-encoded) and decodes the
// "response" field into a map.
func (c *Client) postFormObject(ctx context.Context, path string, fields url.Values) (map[string]any, error) {
	raw, err := c.postForm(ctx, path, fields)
	if err != nil {
		return nil, err
	}
	return unmarshalResponseObject(raw)
}

// putFormObject issues an authenticated PUT (form-encoded) and decodes the
// "response" field into a map.
func (c *Client) putFormObject(ctx context.Context, path string, fields url.Values) (map[string]any, error) {
	raw, err := c.putForm(ctx, path, fields)
	if err != nil {
		return nil, err
	}
	return unmarshalResponseObject(raw)
}

// deleteObject issues an authenticated DELETE and decodes the "response"
// field into a map.
func (c *Client) deleteObject(ctx context.Context, path string) (map[string]any, error) {
	raw, err := c.deleteReq(ctx, path)
	if err != nil {
		return nil, err
	}
	return unmarshalResponseObject(raw)
}

// unmarshalResponseObject extracts the "response" field from a raw OnlyOffice
// envelope and decodes it into map[string]any. Returns (nil, nil) for a null
// response and (nil, err) when the field is missing or malformed.
func unmarshalResponseObject(raw json.RawMessage) (map[string]any, error) {
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 || string(resp) == "null" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
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

// putJSON issues an authenticated PUT with application/json body.
func (c *Client) putJSON(ctx context.Context, path string, body any) (json.RawMessage, error) {
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	var rdr io.Reader
	switch b := body.(type) {
	case nil:
		rdr = strings.NewReader("{}")
	case []byte:
		rdr = bytes.NewReader(b)
	case string:
		rdr = strings.NewReader(b)
	default:
		buf, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL()+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
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
		return nil, fmt.Errorf("PUT JSON %s: %d %s", path, resp.StatusCode, truncate(string(raw), 400))
	}
	return raw, nil
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
