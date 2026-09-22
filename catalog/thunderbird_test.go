package catalog

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNoisyEmail(t *testing.T) {
	if !noisyEmail("noreply@example.com") {
		t.Fatal("expected noisy")
	}
	if !noisyEmail("x@marketplace.amazon.de") {
		t.Fatal("amazon marketplace")
	}
	if noisyEmail("alice.smith@acme.example") {
		t.Fatal("should keep a human work address")
	}
}

func TestParseMABEmails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abook.mab")
	body := `// mork junk
		PrimaryEmail=alice.smith@acme.example
		noreply@github.com
		bob.jones@acme.example
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Neutral default: no deployment rules → unclassified.
	ents, err := parseMABEmails(path, DefaultClassifier())
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 2 {
		t.Fatalf("got %d: %+v", len(ents), ents)
	}
	for _, e := range ents {
		if e.Org != "" || e.Role != "unknown" {
			t.Fatalf("%+v", e)
		}
	}
	// A deployment rule classifies the domain as work.
	cl := &Classifier{MailOrgs: []MailRule{{Domain: "acme.example", Org: "Acme", Zone: "warm", Role: "work"}}}
	ents, err = parseMABEmails(path, cl)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if e.Org != "Acme" || e.Role != "work" || e.Zone != "warm" {
			t.Fatalf("%+v", e)
		}
	}
}

func TestParseGlodaContacts(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "global-messages-db.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE contacts (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE identities (id INTEGER PRIMARY KEY, contactID INTEGER, kind TEXT, value TEXT);
		INSERT INTO contacts VALUES (1, 'Alice Smith');
		INSERT INTO identities VALUES (1, 1, 'email', 'alice.smith@acme.example');
		INSERT INTO contacts VALUES (2, 'Noise Bot');
		INSERT INTO identities VALUES (2, 2, 'email', 'noreply@example.com');
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	ents, err := parseGlodaContacts(dbPath, DefaultClassifier())
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 {
		t.Fatalf("got %d %+v", len(ents), ents)
	}
	if ents[0].First != "Alice" || ents[0].Emails[0] != "alice.smith@acme.example" {
		t.Fatalf("%+v", ents[0])
	}
}

func TestScanThunderbirdRoot(t *testing.T) {
	root := t.TempDir()
	prof := filepath.Join(root, "profile")
	if err := os.MkdirAll(prof, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prof, "abook.mab"), []byte("mail=paul.schmidt@acme.example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := ScanThunderbirdRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Entries) < 1 {
		t.Fatal(doc.Entries)
	}
}
