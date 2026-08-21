package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMboxHeaderEmails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "INBOX")
	body := `From - Mon Jul 1 00:00:00 2016
From: Axel jones <alice.jones@acme.example>
To: eSlider <alice.smith@acme.example>
Cc: noreply@example.com, client@stadt-example.de
Subject: test

Body line ignored
From - Mon Jul 2 00:00:00 2016
From: Someone <bob.smith@acme.example>
To: list@acme.example

more body
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ents, err := parseMboxHeaderEmails(path)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range ents {
		if len(e.Emails) > 0 {
			got[e.Emails[0]] = true
		}
	}
	if !got["alice.jones@acme.example"] || !got["alice.smith@acme.example"] {
		t.Fatalf("%v", got)
	}
	if got["noreply@example.com"] {
		t.Fatal("noreply should be filtered")
	}
	if !got["client@stadt-example.de"] {
		t.Fatal("missing client")
	}
}

func TestScanThunderbirdRootOptsMbox(t *testing.T) {
	root := t.TempDir()
	imap := filepath.Join(root, "ImapMail", "mail.example.com")
	if err := os.MkdirAll(imap, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imap, "INBOX"), []byte(
		"From - x\nFrom: a@acme.example\nTo: b@acme.example\n\nbody\n",
	), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := ScanThunderbirdRootOpts(root, ScanOptions{MboxHeaders: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Entries) < 2 {
		t.Fatalf("%+v", doc.Entries)
	}
}
