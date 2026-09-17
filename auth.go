package onlyoffice

// Authentication primitives: token lifecycle, eager / context-aware auth, and
// token invalidation. Split out of http.go so auth concerns live together.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Auth authenticates using the given credentials and returns a fresh token.
// Most callers should not call this directly — use Authenticate /
// AuthenticateContext, or let Query fetch a token lazily. Kept exported for
// backwards compatibility.
func (c *Client) Auth(creds *Credentials) (*Token, error) {
	t := &Token{}
	return t, c.Query(Request{
		Uri:    "/api/2.0/authentication.json",
		Method: http.MethodPost,
		Body:   creds,
		NoAuth: true,
	}, &struct {
		MetaResponse `json:",inline"`
		Response     *Token `json:"response"`
	}{
		Response: t,
	})
}

// Authenticate validates credentials and primes the token. Library users may
// call this eagerly to surface auth errors at startup; otherwise the token
// is fetched lazily on the first request.
//
// Prefer AuthenticateContext in long-running jobs — it honours cancellation.
func (c *Client) Authenticate() error { return c.ensureToken() }

// AuthenticateContext is the context-aware variant of Authenticate. If the
// cached token is still valid it returns immediately; otherwise it performs
// a POST to /api/2.0/authentication.json that is cancellable via ctx.
//
// Transient answers from the edge (openresty 429/502/503/504) are retried with
// the same deterministic policy as every other request (see retry.go), because
// the server rate-limits authentication and the integration suite otherwise
// fails with a raw HTML 429 page.
//
// This is the recommended entry point for long-running syncs (cron,
// watchers) because it guarantees that a stalled auth call will not block
// the caller past its deadline.
func (c *Client) AuthenticateContext(ctx context.Context) error {
	if c.tokenValid() {
		return nil
	}
	return DoRetry(ctx, DefaultRetryPolicy(), func() error {
		return c.authenticateOnce(ctx)
	})
}

// authenticateOnce performs a single authentication POST. Callers must handle
// retries; use AuthenticateContext.
func (c *Client) authenticateOnce(ctx context.Context) error {
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
		return statusError(resp.StatusCode, retryAfterOf(resp), "auth: %d %s", resp.StatusCode, truncate(string(raw), 400))
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

// tokenValid reports whether the cached token is present and not expired.
func (c *Client) tokenValid() bool {
	return c.token != nil && !time.Time(c.token.Expires).Before(time.Now())
}

// ensureToken refreshes the authentication token when missing or expired.
// Mirrors the logic inline in Query() but is safe to call from helpers that
// bypass the typed Request abstraction. It shares AuthenticateContext so the
// transient-retry policy applies to every code path.
func (c *Client) ensureToken() error {
	if c.tokenValid() {
		return nil
	}
	return c.AuthenticateContext(context.Background())
}

// authHeader returns the value for the Authorization header, ensuring a token.
func (c *Client) authHeader() (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}
	return c.token.Value, nil
}
