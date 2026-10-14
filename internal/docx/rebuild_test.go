package docx_test

import (
	"encoding/xml"
	"regexp"
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/internal/docx"
)

func TestRebuildCommentsShapeAndDates(t *testing.T) {
	path := copyFixture(t)
	f, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// seed an old comment so the rebuild has something to replace
	i, _ := f.FindParagraph("parish of Encarnação")
	if _, err := f.AddComment(i, docx.CommentOptions{Author: "Old", Text: "stale note"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	threads := []docx.ThreadSpec{
		{Messages: []docx.MessageSpec{
			{Author: "Alice Anderson", Text: "first question", Date: "2026-10-12T14:00:00Z"},
			{Author: "Bob Baker", Text: "answer", Date: "2026-10-13T09:00:00Z"},
			{Author: "Alice Anderson", Text: "thank you", Date: "2026-10-14T18:00:00Z"},
		}},
		{Messages: []docx.MessageSpec{
			{Author: "Alice Anderson", Text: "second thread", Date: "2026-10-12T14:00:00Z"},
		}},
	}
	if err := f.RebuildComments(threads); err != nil {
		t.Fatalf("RebuildComments: %v", err)
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertAllXMLParses(t, path)

	parts := zipParts(t, path)
	body := commentPart(t, parts)
	cex := string(parts["word/commentsExtendedDocument.xml"])

	// one element per message, authors and dates preserved
	if got := strings.Count(body, "<w:comment "); got != 4 {
		t.Fatalf("want 4 comment elements, got %d", got)
	}
	if strings.Contains(body, "stale note") {
		t.Fatalf("old comment survived the rebuild")
	}
	for _, want := range []string{
		`w:author="Alice Anderson" w:date="2026-10-12T14:00:00Z"`,
		`w:author="Bob Baker" w:date="2026-10-13T09:00:00Z"`,
		`w:author="Alice Anderson" w:date="2026-10-14T18:00:00Z"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("comment attribute missing: %s", want)
		}
	}
	// one paragraph per element, no merged bodies, no name prefixes invented
	for _, m := range regexp.MustCompile(`(?s)<w:comment\b.*?</w:comment>`).FindAllString(body, -1) {
		if n := strings.Count(m, "<w:p "); n != 1 {
			t.Fatalf("element holds %d paragraphs, want 1: %s", n, m)
		}
	}

	// replies nest under the thread root, not under each other: the portal only
	// shows one level, so message 3 must also point at message 1
	if n := strings.Count(cex, "w15:paraIdParent"); n != 2 {
		t.Fatalf("want 2 threaded entries, got %d", n)
	}
	if !strings.Contains(cex, `w15:paraId="00000002" w15:paraIdParent="00000001"`) {
		t.Fatalf("second message does not point at the root: %s", cex)
	}
	if !strings.Contains(cex, `w15:paraId="00000003" w15:paraIdParent="00000001"`) {
		t.Fatalf("third message must point at the root as well: %s", cex)
	}

	// companion parts stay 1:1 with the comment elements
	if got := strings.Count(string(parts["word/commentsIdsDocument.xml"]), "<w16cid:commentId"); got != 4 {
		t.Fatalf("commentsIds entries = %d, want 4", got)
	}
	if got := strings.Count(string(parts["word/commentsExtensibleDocument.xml"]), "<w16cex:commentExtensible"); got != 4 {
		t.Fatalf("commentsExtensible entries = %d, want 4", got)
	}
	// dates land in the extensible part too
	if !strings.Contains(string(parts["word/commentsExtensibleDocument.xml"]), `dateUtc="2026-10-13T09:00:00Z"`) {
		t.Fatalf("commentExtensible dates not written")
	}

	// anchors are gone, so nothing references a removed id
	if doc := string(parts["word/document.xml"]); strings.Contains(doc, "<w:commentRangeStart") ||
		strings.Contains(doc, "<w:commentReference") {
		t.Fatalf("comment anchors survived the rebuild")
	}

	// the API reports the chain
	g, _ := docx.Open(path)
	cs, err := g.Comments()
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(cs) != 4 {
		t.Fatalf("want 4 comments, got %d", len(cs))
	}
	if cs[0].ParentID != "" {
		t.Fatalf("root must have no parent, got %q", cs[0].ParentID)
	}
	if cs[1].ParentID != cs[0].ID {
		t.Fatalf("second parent = %q, want %q", cs[1].ParentID, cs[0].ID)
	}
	if cs[2].ParentID != cs[0].ID {
		t.Fatalf("third parent = %q, want the root %q", cs[2].ParentID, cs[0].ID)
	}
	if cs[3].ParentID != "" {
		t.Fatalf("second thread root must have no parent, got %q", cs[3].ParentID)
	}
	if cs[1].Date != "2026-10-13T09:00:00Z" {
		t.Fatalf("date not surfaced: %q", cs[1].Date)
	}
}

func TestRebuildCommentsKeepsHighlighting(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)
	i, _ := f.FindParagraph("parish of Encarnação")
	if _, err := f.AddComment(i, docx.CommentOptions{Author: "A", Text: "note", Highlight: "use permit no. 77"}); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := f.RebuildComments([]docx.ThreadSpec{{Messages: []docx.MessageSpec{{Author: "A", Text: "replacement"}}}}); err != nil {
		t.Fatalf("RebuildComments: %v", err)
	}
	f.Save(path)
	assertAllXMLParses(t, path)
	doc := string(zipParts(t, path)["word/document.xml"])
	if !strings.Contains(doc, `<w:highlight w:val="yellow"/>`) {
		t.Fatalf("rebuild dropped the highlighting")
	}
	if strings.Contains(doc, "<w:commentRange") {
		t.Fatalf("anchors should be gone")
	}
}

func TestRebuildCommentsRejectsEmptyMessage(t *testing.T) {
	f, _ := docx.Open(fixture)
	err := f.RebuildComments([]docx.ThreadSpec{{Messages: []docx.MessageSpec{{Author: "A", Text: "  "}}}})
	if err == nil {
		t.Fatalf("empty message must be rejected")
	}
}

func TestRebuildCommentsProducesValidXML(t *testing.T) {
	f, _ := docx.Open(fixture)
	if err := f.RebuildComments([]docx.ThreadSpec{
		{Messages: []docx.MessageSpec{{Author: `O"Brien & Co`, Text: `a < b & c > d`, Date: "2026-10-14T18:00:00Z"}}},
	}); err != nil {
		t.Fatalf("RebuildComments: %v", err)
	}
	path := copyFixture(t)
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	for name, b := range zipParts(t, path) {
		if !strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".rels") {
			continue
		}
		var v any
		if err := xml.Unmarshal(b, &v); err != nil {
			t.Fatalf("part %s invalid after escaping: %v", name, err)
		}
	}
}

// commentPart returns the comments part, which carries the Word name in a
// Writer-produced file and the OnlyOffice name in a portal-produced one.
func commentPart(t *testing.T, parts map[string][]byte) string {
	t.Helper()
	for _, n := range []string{"word/comments.xml", "word/commentsDocument.xml"} {
		if b, ok := parts[n]; ok {
			return string(b)
		}
	}
	t.Fatalf("no comments part in package")
	return ""
}
