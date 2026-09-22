package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVCFFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.vcf")
	body := `BEGIN:VCARD
VERSION:3.0
FN:Ada Lovelace
N:Lovelace;Ada;;;
EMAIL;TYPE=INTERNET:ada@example.com
TEL;TYPE=CELL:+1-555-0100
ORG:Analytical Engines
END:VCARD
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := ParseVCFFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries", len(entries))
	}
	e := entries[0]
	if e.Kind != "person" || e.First != "Ada" || e.Last != "Lovelace" {
		t.Fatalf("unexpected entry: %+v", e)
	}
	if len(e.Emails) != 1 || e.Emails[0] != "ada@example.com" {
		t.Fatalf("emails: %v", e.Emails)
	}
	if e.Org != "Analytical Engines" {
		t.Fatalf("org: %q", e.Org)
	}
	if e.Approve {
		t.Fatal("approve should default false")
	}
}

func TestMergeDocs(t *testing.T) {
	a := &Document{Entries: []Entry{{
		ID: "person:ada@example.com", Kind: "person", First: "Ada", Emails: []string{"ada@example.com"},
		Sources: []string{"a.vcf"}, Zone: "private", Role: "unknown",
	}}}
	b := &Document{Entries: []Entry{{
		ID: "person:ada@example.com", Kind: "person", Last: "Lovelace", Phones: []string{"+1"},
		Sources: []string{"folder/Ada"}, Zone: "warm", Role: "work", Approve: true,
	}}}
	m := MergeDocs(a, b)
	if len(m.Entries) != 1 {
		t.Fatalf("len=%d", len(m.Entries))
	}
	e := m.Entries[0]
	if e.First != "Ada" || e.Last != "Lovelace" {
		t.Fatalf("%+v", e)
	}
	if len(e.Sources) != 2 || len(e.Phones) != 1 || !e.Approve {
		t.Fatalf("%+v", e)
	}
	if e.Zone != "warm" || e.Role != "work" {
		t.Fatalf("zone/role: %s/%s", e.Zone, e.Role)
	}
}

func TestScanContactsRoot(t *testing.T) {
	root := t.TempDir()
	vcfDir := filepath.Join(root, "Contacts VCF's")
	if err := os.MkdirAll(vcfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vcfDir, "bob.vcf"), []byte(`BEGIN:VCARD
VERSION:3.0
FN:Bob Builder
EMAIL:bob@work.test
END:VCARD
`), 0o644); err != nil {
		t.Fatal(err)
	}
	personDir := filepath.Join(root, "Carol Smith")
	if err := os.MkdirAll(personDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(personDir, "carol@example.com.txt"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := ScanContactsRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Entries) < 2 {
		t.Fatalf("expected >=2 entries, got %d: %+v", len(doc.Entries), doc.Entries)
	}
}

func TestScanProjectsRoot(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "acme-demo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cl := &Classifier{WorkNames: []string{"acme"}}
	doc, err := ScanProjectsRootOpts(root, 3, ScanOptions{Classifier: cl})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range doc.Entries {
		if e.Kind == "company" && e.Name == "acme-demo" {
			found = true
			if e.Role != "work" || e.Zone != "warm" {
				t.Fatalf("role=%q zone=%q", e.Role, e.Zone)
			}
		}
	}
	if !found {
		t.Fatalf("missing company: %+v", doc.Entries)
	}

	// Neutral default leaves it unclassified.
	doc, err = ScanProjectsRoot(root, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range doc.Entries {
		if e.Kind == "company" && e.Name == "acme-demo" && e.Role != "unknown" {
			t.Fatalf("neutral role=%q", e.Role)
		}
	}
}

func TestEntryID(t *testing.T) {
	if got := EntryID("person", "A@B.COM", ""); got != "person:a@b.com" {
		t.Fatal(got)
	}
	if got := EntryID("company", "", "Acme Corp"); got != "company:acme corp" {
		t.Fatal(got)
	}
}
