package onlyoffice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResolveMailFolder(t *testing.T) {
	tests := []struct {
		in   string
		want int
		err  bool
	}{
		{"", MailFolderInbox, false},
		{"inbox", MailFolderInbox, false},
		{"SENT", MailFolderSent, false},
		{"4", MailFolderTrash, false},
		{"nope", 0, true},
	}
	for _, tc := range tests {
		got, err := ResolveMailFolder(tc.in)
		if tc.err {
			if err == nil {
				t.Fatalf("%q: expected error", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("%q: got %d err=%v", tc.in, got, err)
		}
	}
}

func TestMailMessagesPath(t *testing.T) {
	path := mailMessagesPath(MailMessagesFilter{Folder: 1}, 1, 10)
	if path != "/api/2.0/mail/messages?count=10&folder=1" {
		t.Fatalf("page 1: got %q", path)
	}
	path = mailMessagesPath(MailMessagesFilter{Folder: 1}, 3, 25)
	if path != "/api/2.0/mail/messages?count=25&folder=1&page=3" {
		t.Fatalf("page 3: got %q", path)
	}
}

func TestParseMailAddress(t *testing.T) {
	tests := []struct {
		raw         string
		wantName    string
		wantAddress string
	}{
		{`"LinkedIn Jobbenachrichtigungen" <jobalerts-noreply@linkedin.com>`, "LinkedIn Jobbenachrichtigungen", "jobalerts-noreply@linkedin.com"},
		{`"Bitfinex" <no-reply@bitfinex.com>`, "Bitfinex", "no-reply@bitfinex.com"},
		{"eslider@gmail.com", "", "eslider@gmail.com"},
		{`"Glassdoor-Jobs" <noreply@glassdoor.com>`, "Glassdoor-Jobs", "noreply@glassdoor.com"},
		{"", "", ""},
	}
	for _, tc := range tests {
		name, addr := ParseMailAddress(tc.raw)
		if name != tc.wantName || addr != tc.wantAddress {
			t.Fatalf("%q: got name=%q addr=%q", tc.raw, name, addr)
		}
	}
}

func TestMailMessagesAsTableRows(t *testing.T) {
	msgs := []map[string]any{{
		"id": float64(42), "subject": "Hi",
		"from": `"Acme" <a@b.com>`,
		"date": "today", "folder": float64(1), "size": float64(100), "isNew": true,
	}}
	rows := MailMessagesAsTableRows(msgs)
	if rows[0]["id"] != "42" || rows[0]["fromName"] != "Acme" || rows[0]["fromAddress"] != "a@b.com" {
		t.Fatalf("got %+v", rows[0])
	}
}

func TestPlainTextToMailHTML(t *testing.T) {
	got := PlainTextToMailHTML("Hello\n\nWorld\nline2")
	if !strings.Contains(got, "<p>Hello</p>") || !strings.Contains(got, "World<br/>line2") {
		t.Fatalf("got %q", got)
	}
	html := "<p>Already</p>"
	if PlainTextToMailHTML(html) != html {
		t.Fatalf("html passthrough failed")
	}
	spaced := MailHTMLWithBlankParagraphs("<p>A</p>\n<p>B</p>")
	if spaced != "<p>A</p>\n<p>&nbsp;</p>\n<p>B</p>" {
		t.Fatalf("spaced: %q", spaced)
	}
}

func TestInt64FromMap(t *testing.T) {
	if Int64FromMap(map[string]any{"id": float64(99)}, "id") != 99 {
		t.Fatal("float64")
	}
	if Int64FromMap(map[string]any{"id": "42"}, "id") != 42 {
		t.Fatal("string")
	}
}

func TestNewClientSetsCookieJar(t *testing.T) {
	c := NewClient(Credentials{Url: "https://example.test", User: "u", Password: "p"})
	if c.client == nil {
		t.Fatal("client is nil")
	}
	if c.client.Jar == nil {
		t.Fatal("cookie jar is nil")
	}
	if _, ok := c.client.Jar.(*cookiejar.Jar); !ok {
		t.Fatalf("unexpected jar type %T", c.client.Jar)
	}
}

func TestDownloadMailAttachmentUsesAuthCookie(t *testing.T) {
	var gotAuth, gotCookie, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/authentication.json":
			http.SetCookie(w, &http.Cookie{Name: "sessionid", Value: "abc123", Path: "/"})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"response":{"token":"tok","expires":"2099-01-01T00:00:00.0000000+00:00"}}`))
		case "/addons/mail/httphandlers/download.ashx":
			gotAuth = r.Header.Get("Authorization")
			gotCookie = r.Header.Get("Cookie")
			gotPath = r.URL.RequestURI()
			if gotCookie == "" {
				http.Error(w, "missing cookie", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte("payload"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	body, err := c.DownloadMailAttachment(ctx, "42")
	if err != nil {
		t.Fatalf("DownloadMailAttachment: %v", err)
	}
	if string(body) != "payload" {
		t.Fatalf("body = %q", body)
	}
	if gotAuth != "tok" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if !strings.Contains(gotCookie, "sessionid=abc123") {
		t.Fatalf("cookie header = %q", gotCookie)
	}
	if gotPath != "/addons/mail/httphandlers/download.ashx?attachid=42" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestSendMailOmitsEmptyCcBcc(t *testing.T) {
	var gotBody map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/authentication.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"response":{"token":"tok","expires":"2099-01-01T00:00:00.0000000+00:00"}}`))
		case "/api/2.0/mail/messages/send.json":
			gotPath = r.URL.Path
			dec := json.NewDecoder(r.Body)
			_ = dec.Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"response":{"id":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(Credentials{Url: srv.URL, User: "u", Password: "p"})
	ctx := context.Background()
	raw, err := c.SendMail(ctx, SendMailParams{
		ID:      99,
		From:    "me@x.com",
		To:      "a@b.com",
		Subject: "hi",
		Body:    "<p>hello</p>",
	})
	if err != nil {
		t.Fatalf("SendMail: %v", err)
	}
	if gotPath != "/api/2.0/mail/messages/send.json" {
		t.Fatalf("path = %q", gotPath)
	}
	if _, hasCC := gotBody["cc"]; hasCC {
		t.Fatalf("empty cc should be omitted: %v", gotBody)
	}
	if _, hasBcc := gotBody["bcc"]; hasBcc {
		t.Fatalf("empty bcc should be omitted: %v", gotBody)
	}
	if gotBody["to"] != "a@b.com" {
		t.Fatalf("to = %v", gotBody["to"])
	}
	if gotBody["id"] != float64(99) {
		t.Fatalf("id = %v", gotBody["id"])
	}
	if !strings.Contains(string(raw), `"id"`) {
		t.Fatalf("raw = %s", raw)
	}
}
