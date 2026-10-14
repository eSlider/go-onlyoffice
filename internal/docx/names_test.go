package docx_test

import (
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/internal/docx"
)

// A prefix is decoration; the namespace URI is the identity. This fixture binds
// the extension URIs to prefixes nobody expects (z14, z15, z16c, z16x) and the
// package must still find every element.
func TestForeignNamespacePrefixesAreResolved(t *testing.T) {
	path := copyFixture(t)
	parts := zipParts(t, path)

	const (
		nsMain = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
		ns2010 = "http://schemas.microsoft.com/office/word/2010/wordml"
		ns2012 = "http://schemas.microsoft.com/office/word/2012/wordml"
		ns16c  = "http://schemas.microsoft.com/office/word/2016/wordml/cid"
		ns18x  = "http://schemas.microsoft.com/office/word/2018/wordml/cex"
	)

	parts["word/comments.xml"] = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:comments xmlns:w="` + nsMain + `" xmlns:z14="` + ns2010 + `">` +
		`<w:comment w:id="0" w:author="Alice Anderson" w:date="2026-10-12T11:50:00Z" w:initials="AA">` +
		`<w:p z14:paraId="0000000A" z14:textId="0000000A"><w:r><w:t>A question.</w:t></w:r></w:p>` +
		`</w:comment>` +
		`<w:comment w:id="1" w:author="Bob Baker" w:date="2026-10-13T09:00:00Z" w:initials="BB">` +
		`<w:p z14:paraId="0000000B" z14:textId="0000000B"><w:r><w:t>An answer.</w:t></w:r></w:p>` +
		`</w:comment>` +
		`</w:comments>`)
	parts["word/commentsExtendedDocument.xml"] = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<z15:commentsEx xmlns:z15="` + ns2012 + `">` +
		`<z15:commentEx z15:paraId="0000000A" z15:done="0"/>` +
		`<z15:commentEx z15:paraId="0000000B" z15:paraIdParent="0000000A" z15:done="0"/>` +
		`</z15:commentsEx>`)
	parts["word/commentsIdsDocument.xml"] = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<z16c:commentsIds xmlns:z16c="` + ns16c + `">` +
		`<z16c:commentId z16c:paraId="0000000A" z16c:durableId="AAAAAAAA"/>` +
		`<z16c:commentId z16c:paraId="0000000B" z16c:durableId="BBBBBBBB"/>` +
		`</z16c:commentsIds>`)
	parts["word/commentsExtensibleDocument.xml"] = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<z16x:commentsExtensible xmlns:z16x="` + ns18x + `">` +
		`<z16x:commentExtensible z16x:durableId="AAAAAAAA" z16x:dateUtc="2026-10-12T11:50:00Z"/>` +
		`<z16x:commentExtensible z16x:durableId="BBBBBBBB" z16x:dateUtc="2026-10-13T09:00:00Z"/>` +
		`</z16x:commentsExtensible>`)
	writeZip(t, path, parts)

	f, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	cs, err := f.Comments()
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("want 2 comments, got %d", len(cs))
	}
	// the reply link had to be read out of a part that spells the namespace z15
	if cs[1].ParentID != cs[0].ID {
		t.Fatalf("parent not resolved through the foreign prefix: %+v", cs)
	}
	if cs[1].Author != "Bob Baker" || cs[1].Date != "2026-10-13T09:00:00Z" {
		t.Fatalf("comment read wrong: %+v", cs[1])
	}

	// a reply added here must land in canonical prefixes and still chain
	if err := f.ReplyToComment(cs[0].ID, "Alice Anderson", "A follow-up."); err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertAllXMLParses(t, path)

	after := zipParts(t, path)
	cex := string(after["word/commentsExtendedDocument.xml"])
	if !strings.Contains(cex, `<w15:commentEx`) {
		t.Fatalf("extended part was not normalised to canonical prefixes: %s", cex)
	}
	if strings.Contains(cex, "z15:") {
		t.Fatalf("foreign prefix survived: %s", cex)
	}
	if n := strings.Count(cex, `w15:paraIdParent="0000000A"`); n != 2 {
		t.Fatalf("both replies must point at the root 0000000A, got %d", n)
	}
	g, _ := docx.Open(path)
	threads, _ := g.Comments()
	if len(threads) != 3 {
		t.Fatalf("want 3 comments, got %d", len(threads))
	}
	replies := 0
	for _, c := range threads {
		if c.ParentID == "0" {
			replies++
		}
	}
	if replies != 2 {
		t.Fatalf("want 2 replies under the root, got %d", replies)
	}
}

// A part that already uses canonical prefixes must not be rewritten, so files we
// do not otherwise touch stay byte for byte identical.
func TestCanonicalPartsAreNotRewritten(t *testing.T) {
	path := copyFixture(t)
	before := zipParts(t, path)
	f, _ := docx.Open(path)
	if f.Changes() != 0 {
		t.Fatalf("opening a clean file must not count as a change")
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := zipParts(t, path)
	for name, want := range before {
		if name == "word/document.xml" {
			continue // stripEmptyCells may normalise an empty table cell
		}
		if got := after[name]; string(got) != string(want) {
			t.Fatalf("part %s was rewritten on load", name)
		}
	}
}
