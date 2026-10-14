package docx_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/internal/docx"
)

// asOnlyOfficeLayout rewrites a normal Word package into the layout the
// OnlyOffice portal writes: the comments live in word/commentsDocument.xml and
// the companion parts commentsExtended / commentsIds / commentsExtensible exist.
// Editing such a file must keep that layout and must keep the companion parts in
// step, otherwise the portal drops the comment on its next save.
func asOnlyOfficeLayout(t *testing.T, path string) {
	t.Helper()
	parts := zipParts(t, path)

	c := parts["word/comments.xml"]
	delete(parts, "word/comments.xml")
	parts["word/commentsDocument.xml"] = c

	ct := string(parts["[Content_Types].xml"])
	ct = strings.Replace(ct, `PartName="/word/comments.xml"`, `PartName="/word/commentsDocument.xml"`, 1)
	ct = strings.Replace(ct, "</Types>",
		`<Override PartName="/word/commentsExtendedDocument.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.commentsExtended+xml"/>`+
			`<Override PartName="/word/commentsIdsDocument.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.commentsIds+xml"/>`+
			`<Override PartName="/word/commentsExtensibleDocument.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.commentsExtensible+xml"/>`+
			"</Types>", 1)
	parts["[Content_Types].xml"] = []byte(ct)

	rels := string(parts["word/_rels/document.xml.rels"])
	rels = strings.Replace(rels, `Target="comments.xml"`, `Target="commentsDocument.xml"`, 1)
	rels = strings.Replace(rels, "</Relationships>",
		`<Relationship Id="rIdComExt" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/commentsExtended" Target="commentsExtendedDocument.xml"/>`+
			`<Relationship Id="rIdComIds" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/commentsIds" Target="commentsIdsDocument.xml"/>`+
			"</Relationships>", 1)
	parts["word/_rels/document.xml.rels"] = []byte(rels)

	parts["word/commentsExtendedDocument.xml"] = []byte(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w15:commentsEx xmlns:w="` + nsW + `" xmlns:w15="http://schemas.microsoft.com/office/word/2012/wordml"></w15:commentsEx>`)
	parts["word/commentsIdsDocument.xml"] = []byte(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w16cid:commentsIds xmlns:w16cid="http://schemas.microsoft.com/office/word/2016/wordml/cid"></w16cid:commentsIds>`)
	parts["word/commentsExtensibleDocument.xml"] = []byte(
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w16cex:commentsExtensible xmlns:w16cex="http://schemas.microsoft.com/office/word/2018/wordml/cex"></w16cex:commentsExtensible>`)

	writeZip(t, path, parts)
}

const nsW = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

func TestOnlyOfficeLayoutIsPreserved(t *testing.T) {
	path := copyFixture(t)
	asOnlyOfficeLayout(t, path)

	f, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	i, err := f.FindParagraph("parish of Encarnação")
	if err != nil {
		t.Fatalf("FindParagraph: %v", err)
	}
	id, err := f.AddComment(i, docx.CommentOptions{
		Author:    "Alice Anderson",
		Text:      "Could you confirm the permit number?",
		Highlight: "use permit no. 77",
	})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := f.ReplyToComment(id, "Bob", "Will check with the lawyer."); err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertAllXMLParses(t, path)

	parts := zipParts(t, path)
	if _, ok := parts["word/commentsDocument.xml"]; !ok {
		t.Fatalf("OnlyOffice comments part disappeared")
	}
	if _, ok := parts["word/comments.xml"]; ok {
		t.Fatalf("a second comments part was created alongside the OnlyOffice one")
	}
	for _, name := range []string{"word/commentsExtendedDocument.xml", "word/commentsIdsDocument.xml", "word/commentsExtensibleDocument.xml"} {
		if _, ok := parts[name]; !ok {
			t.Fatalf("companion part %s disappeared", name)
		}
	}
	ext := string(parts["word/commentsExtendedDocument.xml"])
	if !strings.Contains(ext, "<w15:commentEx") {
		t.Fatalf("commentsExtended was not synced: %s", ext)
	}
	ids := string(parts["word/commentsIdsDocument.xml"])
	if !strings.Contains(ids, "<w16cid:commentId") {
		t.Fatalf("commentsIds was not synced: %s", ids)
	}

	g, err := docx.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	cs, err := g.Comments()
	if err != nil {
		t.Fatalf("Comments: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("want 2 comments (thread + reply), got %d", len(cs))
	}
	var reply, parent *docx.Comment
	for k := range cs {
		if cs[k].ParentID != "" {
			reply = &cs[k]
		} else {
			parent = &cs[k]
		}
	}
	if reply == nil || parent == nil {
		t.Fatalf("reply link missing: %+v", cs)
	}
	if reply.ParentID != parent.ID {
		t.Fatalf("reply parent = %q, want %q", reply.ParentID, parent.ID)
	}
	if !strings.Contains(parent.Text, "Could you confirm the permit number?") {
		t.Fatalf("thread lost its message: %q", parent.Text)
	}
	if !strings.Contains(reply.Text, "Will check with the lawyer.") {
		t.Fatalf("reply lost its message: %q", reply.Text)
	}
}

func TestSaveIsIdempotent(t *testing.T) {
	path := copyFixture(t)
	f, _ := docx.Open(path)
	i, _ := f.FindParagraph("The deposit is EUR 480.00")
	if _, err := f.AddComment(i, docx.CommentOptions{Author: "A", Text: "note", Highlight: "The deposit"}); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	first, _ := os.ReadFile(path)

	// adding nothing and saving again must not churn the file
	g, _ := docx.Open(path)
	if err := g.Save(path); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	second, _ := os.ReadFile(path)
	if !bytes.Equal(first, second) {
		p1 := filepath.Join(t.TempDir(), "x1.docx")
		p2 := filepath.Join(t.TempDir(), "x2.docx")
		os.WriteFile(p1, first, 0o644)
		os.WriteFile(p2, second, 0o644)
		a, b := zipParts(t, p1), zipParts(t, p2)
		for name, want := range a {
			if !bytes.Equal(want, b[name]) {
				t.Fatalf("part %s differs between two consecutive saves", name)
			}
		}
	}
}

// The portal models a threaded reply as a SEPARATE <w:comment> whose commentEx
// entry carries w15:paraIdParent pointing at the parent thread's anchor
// paragraph. Adding a paragraph inside the parent element instead produces one
// comment with several paragraphs, which the editor renders as garbage.
func TestReplyCreatesSeparateCommentWithParentLink(t *testing.T) {
	path := copyFixture(t)
	asOnlyOfficeLayout(t, path)

	f, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	i, _ := f.FindParagraph("parish of Encarnação")
	parent, err := f.AddComment(i, docx.CommentOptions{Author: "Alice", Text: "Alice: original question."})
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	j, _ := f.FindParagraph("The deposit is EUR 480.00")
	if _, err := f.AddComment(j, docx.CommentOptions{Author: "Alice", Text: "Alice: second thread."}); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := f.ReplyToComment(parent, "Bob", "Bob: answered."); err != nil {
		t.Fatalf("ReplyToComment: %v", err)
	}
	if err := f.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	assertAllXMLParses(t, path)

	parts := zipParts(t, path)
	body := string(parts["word/commentsDocument.xml"])
	cex := string(parts["word/commentsExtendedDocument.xml"])

	// the reply is its own element, and the parent element keeps exactly one paragraph
	if got := strings.Count(body, "<w:comment "); got != 3 {
		t.Fatalf("want 3 comment elements (2 threads + 1 reply), got %d", got)
	}
	reParent := regexp.MustCompile(`(?s)<w:comment\b[^>]*w:id="` + parent + `"[^>]*>(.*?)</w:comment>`)
	pm := reParent.FindStringSubmatch(body)
	if pm == nil {
		t.Fatalf("parent thread %s not found", parent)
	}
	if n := strings.Count(pm[1], "<w:p "); n != 1 {
		t.Fatalf("parent thread must stay a single paragraph, got %d", n)
	}

	// anchor paraId of the parent, taken from its commentEx entry
	parentAnchor := ""
	for _, e := range regexp.MustCompile(`<w15:commentEx[^>]*/>`).FindAllString(cex, -1) {
		pid := regexp.MustCompile(`w15:paraId="([^"]+)"`).FindStringSubmatch(e)
		if pid == nil {
			continue
		}
		if strings.Contains(pm[1], `w14:paraId="`+pid[1]+`"`) {
			parentAnchor = pid[1]
		}
	}
	if parentAnchor == "" {
		t.Fatalf("parent anchor paragraph not found in commentEx")
	}

	// exactly one commentEx carries paraIdParent, and it points at the parent anchor
	var withParent []string
	for _, e := range regexp.MustCompile(`<w15:commentEx[^>]*/>`).FindAllString(cex, -1) {
		if strings.Contains(e, "w15:paraIdParent") {
			withParent = append(withParent, e)
		}
	}
	if len(withParent) != 1 {
		t.Fatalf("want exactly 1 threaded entry, got %d: %v", len(withParent), withParent)
	}
	if !strings.Contains(withParent[0], `w15:paraIdParent="`+parentAnchor+`"`) {
		t.Fatalf("reply does not point at parent anchor %s: %s", parentAnchor, withParent[0])
	}

	// one entry per comment element in every companion part
	counts := map[string]int{
		"word/commentsExtendedDocument.xml":  strings.Count(cex, "<w15:commentEx"),
		"word/commentsIdsDocument.xml":       strings.Count(string(parts["word/commentsIdsDocument.xml"]), "<w16cid:commentId"),
		"word/commentsExtensibleDocument.xml": strings.Count(string(parts["word/commentsExtensibleDocument.xml"]), "<w16cex:commentExtensible"),
	}
	for part, n := range counts {
		if n != 3 {
			t.Fatalf("%s: want 3 entries, got %d", part, n)
		}
	}

	// the API reports the thread relationship
	g, _ := docx.Open(path)
	threads, _ := g.Comments()
	if len(threads) != 3 {
		t.Fatalf("want 3 comments, got %d", len(threads))
	}
	replied := 0
	for _, c := range threads {
		if c.ParentID == parent {
			replied++
			if !strings.Contains(c.Text, "Bob: answered.") {
				t.Fatalf("wrong reply body: %q", c.Text)
			}
		}
	}
	if replied != 1 {
		t.Fatalf("want exactly one comment answering %s, got %d", parent, replied)
	}
}
