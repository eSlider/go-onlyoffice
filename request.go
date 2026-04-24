package onlyoffice

// Low-level Request/Query primitive used by the typed helpers in this
// package. Prefer the domain-specific methods (CreateProject, GetTasks, …)
// or the untyped helpers in http.go (ResponseArray, ResponseObject, …) — this
// file is kept for API compatibility and for rare callers that want to hit
// arbitrary endpoints with arbitrary Params/Body.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

// Request for OnlyOffice API.
type Request struct {
	Uri    string  // URI is the path to the API endpoint e.g. /api/2.0/project.json
	Method string  // HTTP method; defaults to GET when empty.
	Params any     // Params is serialised to the query string via go-querystring.
	Body   any     // Body is marshalled to JSON unless it is []byte or string.
	Token  *string // Explicit Authorization header; overrides the cached token.
	NoAuth bool    // Skip automatic authentication.
	Debug  bool    // Kept for backwards compatibility; no longer affects behaviour.
}

// GetMethod returns Method, defaulting to GET when unset.
func (r Request) GetMethod() string {
	if r.Method == "" {
		return http.MethodGet
	}
	return r.Method
}

// Query the OnlyOffice API.
//
//   - If request.Method is empty it defaults to GET.
//   - If request.Body is non-nil it is marshalled to JSON (unless it is
//     already []byte or string, which are passed through verbatim).
//   - If request.Token is nil the cached session token is used (and
//     refreshed as needed unless NoAuth is set).
//   - If request.Token is non-nil it is used verbatim as the Authorization
//     header.
//   - If request.NoAuth is true then no token is fetched — the caller is
//     responsible for authenticating requests (used internally by Auth()).
func (c *Client) Query(request Request, result interface{}) error {
	url := c.credentials.Url + request.Uri

	if request.Params != nil {
		v, err := query.Values(request.Params)
		if err != nil {
			return err
		}
		url = fmt.Sprintf("%s?%s", url, v.Encode())
	}

	rdr, err := requestBodyReader(request.Body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(request.GetMethod(), url, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Pragma", "no-cache")

	if !request.NoAuth {
		if err := c.ensureToken(); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}
	switch {
	case request.Token != nil:
		req.Header.Set("Authorization", *request.Token)
	case c.token != nil:
		req.Header.Set("Authorization", c.token.Value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if result == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// requestBodyReader normalises Query() body input into an io.Reader.
// []byte and string are passed through verbatim; everything else is
// marshalled to JSON. Returns (nil, nil) for a nil body.
func requestBodyReader(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	switch b := body.(type) {
	case []byte:
		return bytes.NewReader(b), nil
	case string:
		return strings.NewReader(b), nil
	default:
		j, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		return bytes.NewReader(j), nil
	}
}

// ToJson for Credentials returns a JSON-encoded payload suitable for the
// authentication endpoint. Returns nil on marshal error (which never happens
// for this struct).
func (c Credentials) ToJson() []byte {
	b, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	return b
}

// MetaResponse is the shape of every envelope returned by the OnlyOffice API.
// It is embedded inline into the per-endpoint response structs via the
// `json:",inline"` convention used throughout this package.
type MetaResponse struct {
	Count      int `json:"count"`
	Total      int `json:"total"`
	Status     int `json:"status"`
	StatusCode int `json:"statusCode"`
}

// Permissions is a common subset of boolean permissions embedded into
// several entity types.
type Permissions struct {
	CanEdit   *bool `json:"canEdit,omitempty"`
	CanDelete *bool `json:"canDelete,omitempty"`
}

// Token is the authentication token returned by /api/2.0/authentication.json.
type Token struct {
	Value   string `json:"token"`
	Expires Time   `json:"expires"`
}

// Time wraps time.Time with the OnlyOffice "yyyy-MM-ddTHH:mm:ss.fffffffzzz"
// wire format used by several endpoints.
type Time time.Time

// String formats the time in OnlyOffice's expected ISO-8601 variant (without
// fractional seconds or timezone).
func (t Time) String() string {
	return time.Time(t).Format("2006-01-02T15:04:05")
}

// Before reports whether t is before u.
func (t Time) Before(u Time) bool {
	return time.Time(t).Before(time.Time(u))
}

// After reports whether t is after u.
func (t Time) After(u Time) bool {
	return time.Time(t).After(time.Time(u))
}

// UnmarshalJSON decodes the OnlyOffice wire format into Time.
func (r *Time) UnmarshalJSON(data []byte) error {
	data = data[1 : len(data)-1] // trim surrounding quotes
	t, err := time.Parse("2006-01-02T15:04:05.0000000-07:00", string(data))
	if err != nil {
		return err
	}
	*r = Time(t)
	return nil
}

// MarshalJSON emits the OnlyOffice-friendly short form.
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, time.Time(t).Format("2006-01-02T15:04:05"))), nil
}
