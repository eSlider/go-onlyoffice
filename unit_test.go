package onlyoffice

// Pure unit tests — no network, no fake vendor HTTP servers.
// Protocol-level behaviour is covered by *_integration_test.go (build-tagged).

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestResponseFieldMissing(t *testing.T) {
	if _, err := responseField(json.RawMessage(`{"other":1}`), "response"); err == nil {
		t.Fatal("expected error on missing field")
	}
}

func TestResponseFieldPresent(t *testing.T) {
	raw, err := responseField(json.RawMessage(`{"response":[1,2,3],"other":9}`), "response")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "[1,2,3]" {
		t.Errorf("got %s", string(raw))
	}
}

func TestRequestBodyReaderNil(t *testing.T) {
	r, err := requestBodyReader(nil)
	if err != nil {
		t.Fatal(err)
	}
	if r != nil {
		t.Errorf("expected nil reader for nil body, got %T", r)
	}
}

func TestRequestBodyReaderBytes(t *testing.T) {
	r, err := requestBodyReader([]byte(`{"k":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal("nil reader")
	}
}

func TestRequestBodyReaderStruct(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	r, err := requestBodyReader(payload{Name: "x"})
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 64)
	n, _ := r.Read(buf)
	if string(buf[:n]) != `{"name":"x"}` {
		t.Errorf("got %q", string(buf[:n]))
	}
}

func TestProjectStringNilSafe(t *testing.T) {
	var p Project
	if got := p.String(); got != "" {
		t.Errorf("nil Title should yield empty string, got %q", got)
	}
	title := "my project"
	p.Title = &title
	if got := p.String(); got != "my project" {
		t.Errorf("got %q", got)
	}
	titleWithPercent := "100% coverage"
	p.Title = &titleWithPercent
	if got := p.String(); got != "100% coverage" {
		t.Errorf("Sprintf format-string regression: got %q", got)
	}
}

func TestAuthenticateContextRespectsCancellation(t *testing.T) {
	// Point the client at a routable-but-unresponsive endpoint (TEST-NET-1
	// per RFC 5737) and cancel the context almost immediately. The test
	// verifies ctx plumbing, not OnlyOffice protocol — no vendor mock.
	c := NewClient(Credentials{Url: "http://192.0.2.1:9", User: "u", Password: "p"})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := c.AuthenticateContext(ctx); err == nil {
		t.Fatal("expected error on context timeout against unreachable endpoint")
	}
}

func TestInvalidateTokenIsIdempotent(t *testing.T) {
	c := NewClient(Credentials{Url: "http://example.invalid", User: "u", Password: "p"})
	c.InvalidateToken()
	c.InvalidateToken()
	if c.token != nil {
		t.Fatal("token should remain nil after double invalidate")
	}
}

func TestGetEnvironmentCredentialsAliases(t *testing.T) {
	t.Setenv("ONLYOFFICE_URL", "")
	t.Setenv("ONLYOFFICE_HOST", "https://example/")
	t.Setenv("ONLYOFFICE_USER", "")
	t.Setenv("ONLYOFFICE_NAME", "alice")
	t.Setenv("ONLYOFFICE_PASS", "")
	t.Setenv("ONLYOFFICE_PASSWORD", "s3cret")
	c := GetEnvironmentCredentials()
	if c.Url != "https://example" {
		t.Errorf("Url alias not applied / trailing slash not trimmed: %q", c.Url)
	}
	if c.User != "alice" {
		t.Errorf("User alias not applied: %q", c.User)
	}
	if c.Password != "s3cret" {
		t.Errorf("Password alias not applied: %q", c.Password)
	}
}

func TestGetEnvironmentDefaultsFallbacks(t *testing.T) {
	t.Setenv("ONLYOFFICE_CALENDAR_ID", "")
	t.Setenv("ONLYOFFICE_PROJECT_ID", "")
	t.Setenv("ONLYOFFICE_CALENDAR_PROJECT_ID", "")
	d := GetEnvironmentDefaults()
	if d.CalendarID != "1" || d.ProjectID != "33" {
		t.Errorf("defaults: %+v", d)
	}
	t.Setenv("ONLYOFFICE_CALENDAR_PROJECT_ID", "7")
	if got := GetEnvironmentDefaults().ProjectID; got != "7" {
		t.Errorf("CalendarProjectId alias: %q", got)
	}
}
