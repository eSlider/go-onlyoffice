package onlyoffice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mailsSyncMock serves a two-page inbox: page 1 has two list items (one
// reporting hasAttachments but omitting the attachment array, forcing the
// full-record fetch), page 2 is empty. The full record for message 102
// carries one attachment whose body is served by download.ashx.
func newMailSyncTestServer(t *testing.T, msgsPage1 string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.0/authentication.json":
			http.SetCookie(w, &http.Cookie{Name: "sessionid", Value: "abc", Path: "/"})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"response":{"token":"tok","expires":"2099-01-01T00:00:00.0000000+00:00"}}`))

		case r.URL.Path == "/api/2.0/mail/messages":
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("page") > "1" {
				_, _ = w.Write([]byte(`{"response":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"response":[` + msgsPage1 + `]}`))

		case r.URL.Path == "/api/2.0/mail/messages/102":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"response":{
				"id":102,"subject":"Full record","from":"\"A\" <a@b.com>",
				"date":"2026-08-22T10:15:00+02:00","folder":1,"isNew":false,
				"hasAttachments":true,
				"attachments":[{"id":77,"fileName":"report.pdf","size":3}]}}`))

		case r.URL.Path == "/addons/mail/httphandlers/download.ashx":
			if r.Header.Get("Cookie") == "" {
				http.Error(w, "missing cookie", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte("PDF!"))

		default:
			http.NotFound(w, r)
		}
	}))
}

func TestFetchMailFolderHydratesAndDownloads(t *testing.T) {
	page1 := `
		{"id":101,"subject":"Plain","from":"x@y.z","date":"2026-08-21T09:00:00Z",
		 "folder":1,"isNew":true,"hasAttachments":false},
		{"id":102,"subject":"With attachment (list item)","from":"a@b.com",
		 "date":"2026-08-22T10:15:00+02:00","folder":1,"isNew":false,
		 "hasAttachments":true}
	`
	srv := newMailSyncTestServer(t, page1)
	defer srv.Close()

	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgs, err := c.FetchMailFolder(ctx, MailFolderInbox, MailSyncOptions{FetchBodies: true})
	if err != nil {
		t.Fatalf("FetchMailFolder: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}

	first := msgs[0]
	if first.ID != 101 || first.Subject != "Plain" || !first.IsNew {
		t.Fatalf("first = %+v", first)
	}
	if first.Date.IsZero() || first.Date.Year() != 2026 {
		t.Fatalf("first date = %v", first.Date)
	}
	if first.HasAttachments {
		t.Fatalf("first should have no attachments")
	}

	second := msgs[1]
	if !second.HasAttachments || len(second.Attachments) != 1 {
		t.Fatalf("second attachments = %+v", second.Attachments)
	}
	att := second.Attachments[0]
	if att.ID != "77" || att.Name != "report.pdf" || att.Size != 3 || string(att.Body) != "PDF!" {
		t.Fatalf("attachment = %+v", att)
	}
	if second.Date.Location() == time.UTC && second.Date.Hour() != 8 {
		t.Fatalf("second date = %v (want +02:00 offset preserved)", second.Date)
	}
}

func TestFetchMailFolderLimitAndStartIndex(t *testing.T) {
	var items []string
	for i := 1; i <= 5; i++ {
		items = append(items, `{"id":`+string(rune('0'+i))+`,"subject":"m`+string(rune('0'+i))+`",
			"from":"x@y.z","date":"2026-08-20T00:00:00Z","folder":1}`)
	}
	srv := newMailSyncTestServer(t, strings.Join(items, ","))
	defer srv.Close()

	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := c.FetchMailFolder(ctx, MailFolderInbox, MailSyncOptions{StartIndex: 1, Limit: 2})
	if err != nil {
		t.Fatalf("FetchMailFolder: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d messages, want 2", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("ids = %d,%d want 2,3", got[0].ID, got[1].ID)
	}
}

func TestParseMailTime(t *testing.T) {
	fractions := "2026-08-22T10:15:00.1234567+02:00"
	if parseMailTime(fractions).IsZero() {
		t.Fatalf("RFC3339 with 7-digit fraction failed: %q", fractions)
	}
	if parseMailTime("2026-08-22T10:15:00").IsZero() {
		t.Fatal("second-precision form failed")
	}
	if !parseMailTime("").IsZero() || !parseMailTime("garbage").IsZero() {
		t.Fatal("unparseable input must yield zero time")
	}
}
