package onlyoffice

// Document conversion via the OnlyOffice DocumentServer converter.
//
// The DocumentServer (the same engine behind the portal's "Download as PDF")
// converts any office format. From a portal-reachable host the converter is
// exposed at "<portal>/ds-vpath/converter" (reverse proxy) or directly at
// "http://<docs-server>:8083/converter" (legacy path: /ConvertService.ashx).
//
// Flow: PresignedURI(fileId) → Convert(docsBase, secret, req) → download
// result.FileURL. The JWT is HS256 signed with the DocumentServer's
// services.CoAuthoring.secret (NOT storage.fs.secretString).

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// PresignedURI returns a short-lived, fetchable download URI for a portal file
// (GET /api/2.0/files/file/{fileId}/presigneduri). The DocumentServer can fetch
// it without the caller's session, so it is the input for Convert.
func (c *Client) PresignedURI(ctx context.Context, fileID string) (string, error) {
	if fileID == "" {
		return "", fmt.Errorf("file id is required")
	}
	raw, err := c.getJSON(ctx, fmt.Sprintf("/api/2.0/files/file/%s/presigneduri", url.PathEscape(fileID)))
	if err != nil {
		return "", err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(resp, &s); err == nil && s != "" {
		return s, nil
	}
	// Some builds return an object instead of a bare string.
	var o map[string]any
	if err := json.Unmarshal(resp, &o); err == nil {
		for _, k := range []string{"uri", "url", "Uri", "Url"} {
			if v, ok := o[k].(string); ok && v != "" {
				return v, nil
			}
		}
	}
	return "", fmt.Errorf("presigneduri: unexpected response %s", truncate(string(resp), 200))
}

// ConvertRequest is the DocumentServer converter body.
type ConvertRequest struct {
	URL        string `json:"url"`
	OutputType string `json:"outputtype"`
	FileType   string `json:"filetype,omitempty"`
	Key        string `json:"key"`
	Title      string `json:"title,omitempty"`
}

// ConvertResult is the DocumentServer converter reply.
type ConvertResult struct {
	FileURL    string `json:"fileUrl"`
	FileType   string `json:"fileType"`
	Percent    int    `json:"percent"`
	EndConvert bool   `json:"endConvert"`
	Error      *int   `json:"error,omitempty"`
}

// SignJWT builds an HS256 JWT with the given payload (stdlib only).
func SignJWT(secret string, payload any) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt secret is empty")
	}
	hb, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString
	signing := enc(hb) + "." + enc(pb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	return signing + "." + enc(mac.Sum(nil)), nil
}

// ConvertDocument asks a DocumentServer to convert req.URL into req.OutputType.
// docsBase is e.g. "https://portal/ds-vpath" or "http://localhost:8083";
// secret is the DocumentServer CoAuthoring JWT secret. Passes the JWT both as
// the AuthorizationJwt header and as a body token.
func (c *Client) ConvertDocument(ctx context.Context, docsBase, secret string, req ConvertRequest) (*ConvertResult, error) {
	if strings.TrimSpace(docsBase) == "" {
		return nil, fmt.Errorf("docs base url is required")
	}
	if req.URL == "" {
		return nil, fmt.Errorf("source url is required")
	}
	if req.OutputType == "" {
		return nil, fmt.Errorf("outputtype is required")
	}
	if req.Key == "" {
		return nil, fmt.Errorf("conversion key is required")
	}
	jwt, err := SignJWT(secret, req)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(docsBase, "/") + "/converter"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("AuthorizationJwt", "Bearer "+jwt)
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("converter request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("converter: %d %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out ConvertResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("converter decode: %w (%s)", err, truncate(string(raw), 200))
	}
	if out.Error != nil {
		return &out, fmt.Errorf("converter error %d", *out.Error)
	}
	if out.FileURL == "" {
		return &out, fmt.Errorf("converter returned no fileUrl")
	}
	return &out, nil
}

// DownloadURLTo streams an absolute URL (no portal auth) into dst.
func (c *Client) DownloadURLTo(ctx context.Context, rawurl string, dst io.Writer) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("download: %d", resp.StatusCode)
	}
	return io.Copy(dst, resp.Body)
}
