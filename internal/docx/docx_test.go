package docx_test

import (
	"archive/zip"
	"regexp"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/internal/docx"
)

const fixture = "../../testdata/docx/sample.docx"

func copyFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "work.docx")
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatalf("write copy: %v", err)
	}
	return dst
}

func zipParts(t *testing.T, path string) map[string][]byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip %s: %v", path, err)
	}
	defer zr.Close()
	out := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open part %s: %v", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read part %s: %v", f.Name, err)
		}
		out[f.Name] = b
	}
	return out
}

// assertAllXMLParses guards the whole package: nothing we write may produce
// malformed XML. Go's decoder is lenient about namespace prefixes, so prefix
// bindings are checked separately: Word and the portal both reject an unbound
// prefix even though Go's parser accepts it.
func assertAllXMLParses(t *testing.T, path string) {
	t.Helper()
	for name, b := range zipParts(t, path) {
		if !strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".rels") {
			continue
		}
		dec := xml.NewDecoder(bytes.NewReader(b))
		for {
			_, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("part %s is not valid XML: %v", name, err)
			}
		}
		assertPrefixesBound(t, name, string(b))
	}
}

// assertPrefixesBound fails when a document uses a namespace prefix that is
// never declared anywhere in it, e.g. w14:paraId without xmlns:w14. Declarations
// may sit on any element, so the whole part is scanned rather than just the root.
func assertPrefixesBound(t *testing.T, part, body string) {
	t.Helper()
	declared := map[string]bool{"xml": true, "xmlns": true}
	for _, m := range regexp.MustCompile(`xmlns:([A-Za-z0-9]+)=`).FindAllStringSubmatch(body, -1) {
		declared[m[1]] = true
	}
	used := map[string]bool{}
	for _, m := range regexp.MustCompile(`</?([A-Za-z0-9]+):`).FindAllStringSubmatch(body, -1) {
		used[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`\s([A-Za-z0-9]+):[A-Za-z0-9]+=`).FindAllStringSubmatch(body, -1) {
		used[m[1]] = true
	}
	for p := range used {
		if !declared[p] {
			t.Fatalf("part %s uses undeclared namespace prefix %q", part, p)
		}
	}
}

func TestOpenAndParagraphTexts(t *testing.T) {
	f, err := docx.Open(fixture)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	texts := f.ParagraphTexts()
	want := "The contract starts on 3 November 2026 and terminates on 3 November 2028."
	found := false
	for _, s := range texts {
		if s == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("paragraph %q not found in %d paragraphs", want, len(texts))
	}
}

func TestFindParagraph(t *testing.T) {
	f, _ := docx.Open(fixture)

	if _, err := f.FindParagraph("monthly rent is EUR 480.00"); err != nil {
		t.Fatalf("unique substring should be found: %v", err)
	}
	if _, err := f.FindParagraph("no such text anywhere"); !errors.Is(err, docx.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	// "the" appears in many paragraphs on purpose: ambiguity must be reported, not guessed.
	if _, err := f.FindParagraph("the"); !errors.Is(err, docx.ErrNotUnique) {
		t.Fatalf("want ErrNotUnique, got %v", err)
	}
}

func TestReplaceInParagraphTouchesOnlyTarget(t *testing.T) {
	path := copyFixture(t)
	f, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before := f.ParagraphTexts()

	i, err := f.FindParagraph("payable up to the 5th day of the month")
	if err != nil {
		t.Fatalf("FindParagraph: %v", err)
	}
	if err := f.ReplaceInParagraph(i, "up to the 5th day of the month", "up to the 8th day of the month"); err != nil {
		t.Fatalf("ReplaceInParagraph: %v", err)
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertAllXMLParses(t, path)

	g, err := docx.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	after := g.ParagraphTexts()
	if len(before) != len(after) {
		t.Fatalf("paragraph count changed: %d -> %d", len(before), len(after))
	}
	for k := range before {
		if k == i {
			if !strings.Contains(after[k], "up to the 8th day of the month") {
				t.Fatalf("target paragraph not updated: %q", after[k])
			}
			continue
		}
		if before[k] != after[k] {
			t.Fatalf("paragraph %d changed unexpectedly:\n before %q\n after  %q", k, before[k], after[k])
		}
	}
}

func TestSetParagraphText(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)
	i, err := f.FindParagraph("Either party may terminate")
	if err != nil {
		t.Fatalf("FindParagraph: %v", err)
	}
	const replacement = "Either party may terminate this contract by giving two months of written notice."
	if err := f.SetParagraphText(i, replacement); err != nil {
		t.Fatalf("SetParagraphText: %v", err)
	}
	f.Save(path)
	assertAllXMLParses(t, path)

	g, _ := docx.Open(path)
	if got := g.ParagraphTexts()[i]; got != replacement {
		t.Fatalf("got %q want %q", got, replacement)
	}
}

func TestInsertAndDeleteParagraph(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)
	before := len(f.ParagraphTexts())

	i, err := f.FindParagraph("SIXTH CLAUSE")
	if err != nil {
		t.Fatalf("FindParagraph: %v", err)
	}
	tmpl, err := f.FindParagraph("The parties agree that the air conditioning")
	if err != nil {
		t.Fatalf("template lookup: %v", err)
	}
	if err := f.InsertParagraphAfter(i, tmpl, "SEVENTH CLAUSE"); err != nil {
		t.Fatalf("InsertParagraphAfter: %v", err)
	}
	if err := f.DeleteParagraph(i); err != nil {
		t.Fatalf("DeleteParagraph: %v", err)
	}
	f.Save(path)
	assertAllXMLParses(t, path)

	g, _ := docx.Open(path)
	texts := g.ParagraphTexts()
	if len(texts) != before {
		t.Fatalf("count: got %d want %d", len(texts), before)
	}
	joined := strings.Join(texts, "\n")
	if strings.Contains(joined, "SIXTH CLAUSE") {
		t.Fatalf("deleted clause still present")
	}
	if !strings.Contains(joined, "SEVENTH CLAUSE") {
		t.Fatalf("inserted clause missing")
	}
}

func TestAddCommentCreatesAnchorHighlightAndRoundTrips(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)

	i, err := f.FindParagraph("parish of Encarnação")
	if err != nil {
		t.Fatalf("FindParagraph: %v", err)
	}
	id, err := f.AddComment(i, docx.CommentOptions{
		Author:    "Alice Anderson",
		Initials:  "AO",
		Date:      "2026-10-14T12:00:00Z",
		Text:      "Could you confirm the permit number?",
		Highlight: "use permit no. 77",
	})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if id == "" {
		t.Fatalf("empty comment id")
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertAllXMLParses(t, path)

	parts := zipParts(t, path)
	doc := string(parts["word/document.xml"])
	if !strings.Contains(doc, `w:commentRangeStart w:id="`+id+`"`) {
		t.Fatalf("commentRangeStart missing in document.xml")
	}
	if !strings.Contains(doc, `w:commentRangeEnd w:id="`+id+`"`) {
		t.Fatalf("commentRangeEnd missing in document.xml")
	}
	if !strings.Contains(doc, `w:commentReference w:id="`+id+`"`) {
		t.Fatalf("commentReference missing in document.xml")
	}
	if !strings.Contains(doc, `<w:highlight w:val="yellow"/>`) {
		t.Fatalf("highlight not applied")
	}

	g, _ := docx.Open(path)
	cs, err := g.Comments()
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(cs) != 1 {
		t.Fatalf("got %d comments want 1", len(cs))
	}
	if cs[0].Author != "Alice Anderson" || cs[0].Text != "Could you confirm the permit number?" {
		t.Fatalf("comment round-trip mismatch: %+v", cs[0])
	}
}

func TestAddCommentWhenNoCommentsPartExists(t *testing.T) {
	path := copyFixture(t)
	stripCommentsPart(t, path)

	f, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	i, err := f.FindParagraph("The deposit is EUR 480.00")
	if err != nil {
		t.Fatalf("FindParagraph: %v", err)
	}
	if _, err := f.AddComment(i, docx.CommentOptions{Author: "A", Text: "note"}); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertAllXMLParses(t, path)

	parts := zipParts(t, path)
	if _, ok := parts["word/comments.xml"]; !ok {
		t.Fatalf("comments part was not created")
	}
	if !strings.Contains(string(parts["[Content_Types].xml"]), "/word/comments.xml") {
		t.Fatalf("content type override not added")
	}
	if !strings.Contains(string(parts["word/_rels/document.xml.rels"]), "comments.xml") {
		t.Fatalf("relationship not added")
	}
	g, _ := docx.Open(path)
	cs, _ := g.Comments()
	if len(cs) != 1 {
		t.Fatalf("got %d comments want 1", len(cs))
	}
}

func TestAddCommentWithoutHighlightAnchorsWholeParagraph(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)
	i, _ := f.FindParagraph("The parties agree that the air conditioning")
	if _, err := f.AddComment(i, docx.CommentOptions{Author: "A", Text: "whole paragraph"}); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	f.Save(path)
	assertAllXMLParses(t, path)
	if strings.Contains(string(zipParts(t, path)["word/document.xml"]), "<w:highlight") {
		t.Fatalf("no highlight expected when Highlight is empty")
	}
}

func TestReplyToComment(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)
	i, _ := f.FindParagraph("parish of Encarnação")
	id, err := f.AddComment(i, docx.CommentOptions{Author: "Bob", Text: "Will check."})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := f.ReplyToComment(id, "Alice Anderson", "Thank you, appreciated."); err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	f.Save(path)
	assertAllXMLParses(t, path)

	g, _ := docx.Open(path)
	cs, _ := g.Comments()
	if len(cs) != 2 {
		t.Fatalf("a reply is its own comment: got %d comments, want 2", len(cs))
	}
	var reply *docx.Comment
	for k := range cs {
		if cs[k].ParentID == id {
			reply = &cs[k]
		}
	}
	if reply == nil {
		t.Fatalf("no comment reports %s as its parent: %+v", id, cs)
	}
	if !strings.Contains(reply.Text, "Thank you, appreciated.") {
		t.Fatalf("reply body lost: %q", reply.Text)
	}
	if !strings.Contains(cs[0].Text, "Will check.") {
		t.Fatalf("parent body lost: %q", cs[0].Text)
	}
}

func TestReplyToUnknownCommentFails(t *testing.T) {
	f, _ := docx.Open(fixture)
	if err := f.ReplyToComment("999", "A", "text"); !errors.Is(err, docx.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSecondCommentGetsDistinctID(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)
	i, _ := f.FindParagraph("FIFTH CLAUSE")
	j, _ := f.FindParagraph("SIXTH CLAUSE")
	id1, err := f.AddComment(i, docx.CommentOptions{Author: "A", Text: "one"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	id2, err := f.AddComment(j, docx.CommentOptions{Author: "A", Text: "two"})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("duplicate comment id %q", id1)
	}
	f.Save(path)
	assertAllXMLParses(t, path)
	g, _ := docx.Open(path)
	cs, _ := g.Comments()
	if len(cs) != 2 {
		t.Fatalf("got %d comments want 2", len(cs))
	}
}

func TestSavePreservesUnrelatedParts(t *testing.T) {
	path := copyFixture(t)
	orig := zipParts(t, fixture)

	f, _ := docx.Open(path)
	i, _ := f.FindParagraph("The deposit is EUR 480.00")
	if _, err := f.AddComment(i, docx.CommentOptions{Author: "A", Text: "note"}); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	f.Save(path)

	got := zipParts(t, path)
	touched := map[string]bool{
		"word/document.xml":              true,
		"word/comments.xml":              true,
		"[Content_Types].xml":            true,
		"word/_rels/document.xml.rels":   true,
	}
	for name, want := range orig {
		if touched[name] {
			continue
		}
		have, ok := got[name]
		if !ok {
			t.Fatalf("part %s disappeared", name)
		}
		if !bytes.Equal(want, have) {
			t.Fatalf("part %s changed but should not have", name)
		}
	}
}

// stripCommentsPart emulates a document that has never had comments: no part, no override, no rel.
func stripCommentsPart(t *testing.T, path string) {
	t.Helper()
	parts := zipParts(t, path)
	delete(parts, "word/comments.xml")

	ct := string(parts["[Content_Types].xml"])
	ct = removeTagContaining(ct, "/word/comments.xml")
	parts["[Content_Types].xml"] = []byte(ct)

	rels := string(parts["word/_rels/document.xml.rels"])
	rels = removeTagContaining(rels, "comments.xml")
	parts["word/_rels/document.xml.rels"] = []byte(rels)

	writeZip(t, path, parts)
}

func removeTagContaining(s, needle string) string {
	for {
		i := strings.Index(s, needle)
		if i < 0 {
			return s
		}
		start := strings.LastIndex(s[:i], "<")
		end := strings.Index(s[i:], ">")
		if start < 0 || end < 0 {
			return s
		}
		s = s[:start] + s[i+end+1:]
	}
}

func writeZip(t *testing.T, path string, parts map[string][]byte) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := make([]string, 0, len(parts))
	for _, n := range []string{"[Content_Types].xml", "_rels/.rels", "word/_rels/document.xml.rels"} {
		if _, ok := parts[n]; ok {
			names = append(names, n)
		}
	}
	for n := range parts {
		dup := false
		for _, x := range names {
			if x == n {
				dup = true
			}
		}
		if !dup {
			names = append(names, n)
		}
	}
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatalf("create %s: %v", n, err)
		}
		if _, err := w.Write(parts[n]); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
