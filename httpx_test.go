package onlyoffice

// Unit tests for the form/multipart helpers and a few CRM/Calendar/Subtask
// methods. These do not require a real OnlyOffice server — they spin up a
// net/http/httptest.Server that emulates the expected endpoints.

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeServer wires a minimal OnlyOffice-like API: it always returns the same
// token on /api/2.0/authentication.json and dispatches remaining requests via
// the caller-supplied handler.
func fakeServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/2.0/authentication.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"token":"TEST_TOKEN","expires":"2099-01-01T00:00:00Z"}}`)
	})
	mux.HandleFunc("/", h)
	return httptest.NewServer(mux)
}

// newTestClient builds a Client pointed at the test server with a primed token
// so helpers skip the auth round-trip when the test does not exercise it.
func newTestClient(srv *httptest.Server) *Client {
	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})
	c.token = &Token{Value: "TEST_TOKEN", Expires: Time(time.Now().Add(time.Hour))}
	return c
}

func TestAddSubtaskFormEncoded(t *testing.T) {
	var gotPath, gotAuth, gotCT, gotBody string
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"id":99,"title":"child"}}`)
	})
	defer srv.Close()
	c := newTestClient(srv)

	out, err := c.AddSubtask(context.Background(), "42", "child")
	if err != nil {
		t.Fatalf("AddSubtask: %v", err)
	}
	if gotPath != "/api/2.0/project/task/42.json" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAuth != "TEST_TOKEN" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if !strings.HasPrefix(gotCT, "application/x-www-form-urlencoded") {
		t.Errorf("content-type = %q", gotCT)
	}
	if gotBody != "title=child" {
		t.Errorf("body = %q", gotBody)
	}
	if fm, ok := out["title"].(string); !ok || fm != "child" {
		t.Errorf("out.title = %v", out["title"])
	}
}

func TestAddEventUsesDefaultCalendar(t *testing.T) {
	var gotPath, gotBody string
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":[{"id":"e1","name":"hi"}]}`)
	})
	defer srv.Close()
	c := newTestClient(srv)
	c.SetDefaults(Defaults{CalendarID: "7"})

	ev, err := c.AddEvent(context.Background(), "", "hi", "2025-01-01T10:00:00Z", "2025-01-01T11:00:00Z", "desc", false)
	if err != nil {
		t.Fatalf("AddEvent: %v", err)
	}
	if gotPath != "/api/2.0/calendar/7/event.json" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, "name=hi") || !strings.Contains(gotBody, "isAllDayLong=false") {
		t.Errorf("body = %q", gotBody)
	}
	if ev["id"] != "e1" {
		t.Errorf("event id = %v", ev["id"])
	}
}

func TestAddEventRequiresCalendar(t *testing.T) {
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server must not be hit without calendar id")
	})
	defer srv.Close()
	c := newTestClient(srv)
	if _, err := c.AddEvent(context.Background(), "", "t", "s", "e", "", false); err == nil {
		t.Fatal("expected error when no calendar id and no default")
	}
}

func TestListContactsTotal(t *testing.T) {
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.0/crm/contact/filter.json" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("filterValue") != "acme" {
			t.Errorf("filterValue = %q", r.URL.Query().Get("filterValue"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":[{"id":1,"displayName":"ACME"}],"total":7}`)
	})
	defer srv.Close()
	c := newTestClient(srv)

	items, total, err := c.ListContacts(context.Background(), 50, 0, "acme")
	if err != nil {
		t.Fatalf("ListContacts: %v", err)
	}
	if total != 7 {
		t.Errorf("total = %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d", len(items))
	}
}

func TestUploadOpportunityFileMultipart(t *testing.T) {
	tmp := t.TempDir() + "/a.txt"
	if err := writeFile(tmp, "hello"); err != nil {
		t.Fatal(err)
	}
	var gotField, gotFilename, gotBody string
	srv := fakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("content-type = %q", ct)
		}
		_, params, err := mime.ParseMediaType(ct)
		if err != nil {
			t.Fatal(err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		part, err := mr.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		gotField = part.FormName()
		gotFilename = part.FileName()
		b, _ := io.ReadAll(part)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"id":1,"title":"a.txt"}}`)
	})
	defer srv.Close()
	c := newTestClient(srv)

	out, err := c.UploadOpportunityFile(context.Background(), "123", tmp)
	if err != nil {
		t.Fatalf("UploadOpportunityFile: %v", err)
	}
	if gotField != "file" || gotFilename != "a.txt" || gotBody != "hello" {
		t.Errorf("multipart = field=%q filename=%q body=%q", gotField, gotFilename, gotBody)
	}
	if out["title"] != "a.txt" {
		t.Errorf("out.title = %v", out["title"])
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

func TestResponseFieldMissing(t *testing.T) {
	_, err := responseField(json.RawMessage(`{"other":1}`), "response")
	if err == nil {
		t.Fatal("expected error on missing field")
	}
}

func TestAuthenticateContextUsesCacheWhenFresh(t *testing.T) {
	var authHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/2.0/authentication.json", func(w http.ResponseWriter, r *http.Request) {
		authHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"token":"FRESH_TOKEN","expires":"2099-01-01T00:00:00.0000000-00:00"}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})

	if err := c.AuthenticateContext(context.Background()); err != nil {
		t.Fatalf("first AuthenticateContext: %v", err)
	}
	if authHits != 1 {
		t.Errorf("expected 1 auth hit after first call, got %d", authHits)
	}
	if c.token == nil || c.token.Value != "FRESH_TOKEN" {
		t.Errorf("token not cached: %+v", c.token)
	}
	if err := c.AuthenticateContext(context.Background()); err != nil {
		t.Fatalf("second AuthenticateContext: %v", err)
	}
	if authHits != 1 {
		t.Errorf("cache bypassed: expected 1 auth hit, got %d", authHits)
	}
}

func TestInvalidateTokenForcesReauth(t *testing.T) {
	var authHits int
	mux := http.NewServeMux()
	mux.HandleFunc("/api/2.0/authentication.json", func(w http.ResponseWriter, r *http.Request) {
		authHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"response":{"token":"T","expires":"2099-01-01T00:00:00.0000000-00:00"}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})

	if err := c.AuthenticateContext(context.Background()); err != nil {
		t.Fatalf("auth: %v", err)
	}
	c.InvalidateToken()
	if c.token != nil {
		t.Fatalf("token still cached after Invalidate: %+v", c.token)
	}
	if err := c.AuthenticateContext(context.Background()); err != nil {
		t.Fatalf("re-auth: %v", err)
	}
	if authHits != 2 {
		t.Errorf("expected 2 auth hits after invalidate, got %d", authHits)
	}
}

func TestAuthenticateContextRespectsCancellation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/2.0/authentication.json", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			_, _ = io.WriteString(w, `{"response":{"token":"T","expires":"2099-01-01T00:00:00.0000000-00:00"}}`)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.AuthenticateContext(ctx); err == nil {
		t.Fatal("expected error on context timeout")
	}
}
