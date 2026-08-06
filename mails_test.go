package onlyoffice

import (
	"strings"
	"testing"
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
		raw          string
		wantName     string
		wantAddress  string
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
