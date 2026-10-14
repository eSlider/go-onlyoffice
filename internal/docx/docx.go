// Package docx edits WordprocessingML (.docx) packages without disturbing
// anything that is not explicitly changed.
//
// It works on the raw package parts: every part that is not touched is written
// back byte for byte. That matters here, because these files come from Word and
// from OnlyOffice, and both are sensitive to details that a generic decode and
// re-encode cycle silently rewrites.
//
// Operations:
//
//	paragraph text: read, set, substring replace, insert after, delete
//	comments: add (anchored to a paragraph or to a substring, optionally
//	          highlighted) and reply to an existing thread
//
// Comments are written to word/comments.xml (the Word name). If the package
// already carries the OnlyOffice name word/commentsDocument.xml, that part is
// reused, and the companion parts commentsExtended, commentsIds and
// commentsExtensible are kept in step so the portal does not drop the comment on
// its next save.
package docx

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Sentinel errors returned by the lookup helpers.
var (
	ErrNotFound  = errors.New("docx: no matching paragraph")
	ErrNotUnique = errors.New("docx: more than one paragraph matches")
)

const (
	partDocument    = "word/document.xml"
	partRels        = "word/_rels/document.xml.rels"
	partContentType = "[Content_Types].xml"

	partCommentsStd = "word/comments.xml"
	partCommentsOO  = "word/commentsDocument.xml"

	partCommentsExt  = "word/commentsExtendedDocument.xml"
	partCommentsIDs  = "word/commentsIdsDocument.xml"
	partCommentsExt2 = "word/commentsExtensibleDocument.xml"

	nsW   = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsR   = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsW14 = "http://schemas.microsoft.com/office/word/2010/wordml"
	nsW15 = "http://schemas.microsoft.com/office/word/2012/wordml"
	nsW16 = "http://schemas.microsoft.com/office/word/2016/wordml/cid"
	nsCEX = "http://schemas.microsoft.com/office/word/2018/wordml/cex"

	relTypeComments = nsR + "/comments"
)

var (
	reParagraph = regexp.MustCompile(`(?s)<w:p\b.*?</w:p>`)
	reRun       = regexp.MustCompile(`(?s)<w:r\b.*?</w:r>`)
	reRunHead   = regexp.MustCompile(`(?s)^(<w:r\b[^>]*>)(.*)$`)
	reRPr       = regexp.MustCompile(`(?s)^(<w:rPr>.*?</w:rPr>)(.*)$`)
	reAnyRPr    = regexp.MustCompile(`(?s)<w:rPr>.*?</w:rPr>`)
	reRunText   = regexp.MustCompile(`(?s)^<w:t\b[^>]*>(.*)</w:t>$`)
	reWtText    = regexp.MustCompile(`(?s)<w:t\b[^>]*>(.*?)</w:t>`)
	reParaHead  = regexp.MustCompile(`(?s)^(<w:p\b[^>]*>)(.*)$`)
	reParaPPr   = regexp.MustCompile(`(?s)^(<w:pPr>.*?</w:pPr>)(.*)$`)
	reParaID    = regexp.MustCompile(`\s+w14:paraId="[^"]*"`)
	reTextID    = regexp.MustCompile(`\s+w14:textId="[^"]*"`)
	reCommentEl = regexp.MustCompile(`(?s)<w:comment\b[^>]*>.*?</w:comment>`)
	reHighlight = regexp.MustCompile(`<w:highlight[^>]*/>`)
)

// File is an opened .docx package.
type File struct {
	parts map[string][]byte
	order []string

	doc      string
	docDirty bool

	comments      string
	commentsName  string
	commentsDirty bool

	changes int
}

// Open reads a .docx from disk.
func Open(path string) (*File, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("docx: open %s: %w", path, err)
	}
	defer zr.Close()

	f := &File{parts: map[string][]byte{}}
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			return nil, fmt.Errorf("docx: read %s: %w", zf.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("docx: read %s: %w", zf.Name, err)
		}
		f.parts[zf.Name] = b
		f.order = append(f.order, zf.Name)
	}
	doc, ok := f.parts[partDocument]
	if !ok {
		return nil, fmt.Errorf("docx: %s has no %s", path, partDocument)
	}
	f.doc = f.canonicalPart(partDocument, string(doc))

	switch {
	case f.parts[partCommentsStd] != nil:
		f.commentsName = partCommentsStd
	case f.parts[partCommentsOO] != nil:
		f.commentsName = partCommentsOO
	}
	if f.commentsName != "" {
		f.comments = f.canonicalPart(f.commentsName, string(f.parts[f.commentsName]))
	}
	for _, n := range []string{partCommentsExt, partCommentsIDs, partCommentsExt2} {
		if b, ok := f.parts[n]; ok {
			if norm := canonicalise(string(b)); norm != string(b) {
				f.parts[n] = []byte(norm)
			}
		}
	}
	return f, nil
}

// canonicalPart rewrites the namespace prefixes of a part into the canonical
// ones. The prefix a document chooses carries no meaning, only the namespace URI
// does, so normalising it on load keeps every later lookup honest without
// threading a resolver through the whole package. The part is left untouched
// when it already uses canonical prefixes.
func (f *File) canonicalPart(name, body string) string {
	norm := canonicalise(body)
	if norm == body {
		return body
	}
	if name == partDocument {
		f.docDirty = true
	} else if name == f.commentsName {
		f.commentsDirty = true
	} else {
		f.parts[name] = []byte(norm)
	}
	return norm
}

// Changes reports how many edits were applied.
func (f *File) Changes() int { return f.changes }

// Save writes the package back to path. Untouched parts are copied verbatim.
func (f *File) Save(path string) error {
	if f.docDirty {
		f.doc = stripEmptyCells(f.doc)
		f.parts[partDocument] = []byte(f.doc)
	}
	if f.commentsDirty && f.commentsName != "" {
		f.parts[f.commentsName] = []byte(f.comments)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	written := map[string]bool{}
	write := func(name string) error {
		b, ok := f.parts[name]
		if !ok {
			return nil
		}
		w, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("docx: write %s: %w", name, err)
		}
		if _, err := w.Write(b); err != nil {
			return fmt.Errorf("docx: write %s: %w", name, err)
		}
		written[name] = true
		return nil
	}
	for _, name := range f.order {
		if err := write(name); err != nil {
			return err
		}
	}
	var extra []string
	for name := range f.parts {
		if !written[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	for _, name := range extra {
		if err := write(name); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("docx: close archive: %w", err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// ---------------------------------------------------------------- paragraphs

func (f *File) spans() [][]int { return reParagraph.FindAllStringIndex(f.doc, -1) }

// ParagraphTexts returns the visible text of every paragraph, in document order.
func (f *File) ParagraphTexts() []string {
	spans := f.spans()
	out := make([]string, 0, len(spans))
	for _, s := range spans {
		out = append(out, ParagraphText(f.doc[s[0]:s[1]]))
	}
	return out
}

// ParagraphText returns the concatenated <w:t> content of one paragraph element.
func ParagraphText(p string) string {
	var b strings.Builder
	for _, m := range reWtText.FindAllStringSubmatch(p, -1) {
		b.WriteString(Unescape(m[1]))
	}
	return b.String()
}

// FindParagraph returns the index of the single paragraph containing sub.
func (f *File) FindParagraph(sub string) (int, error) {
	if sub == "" {
		return 0, errors.New("docx: empty search text")
	}
	var hits []int
	for i, s := range f.ParagraphTexts() {
		if strings.Contains(s, sub) {
			hits = append(hits, i)
		}
	}
	switch len(hits) {
	case 0:
		return 0, fmt.Errorf("%w: %q", ErrNotFound, clip(sub))
	case 1:
		return hits[0], nil
	default:
		return 0, fmt.Errorf("%w: %q appears in %d paragraphs", ErrNotUnique, clip(sub), len(hits))
	}
}

// FindParagraphExact matches the whole paragraph text, ignoring surrounding space.
func (f *File) FindParagraphExact(text string) (int, error) {
	var hits []int
	want := strings.TrimSpace(text)
	for i, s := range f.ParagraphTexts() {
		if strings.TrimSpace(s) == want {
			hits = append(hits, i)
		}
	}
	switch len(hits) {
	case 0:
		return 0, fmt.Errorf("%w: %q", ErrNotFound, clip(text))
	case 1:
		return hits[0], nil
	default:
		return 0, fmt.Errorf("%w: %q appears in %d paragraphs", ErrNotUnique, clip(text), len(hits))
	}
}

func (f *File) paragraph(i int) (string, error) {
	spans := f.spans()
	if i < 0 || i >= len(spans) {
		return "", fmt.Errorf("docx: paragraph %d out of range (have %d)", i, len(spans))
	}
	return f.doc[spans[i][0]:spans[i][1]], nil
}

func (f *File) replaceParagraph(i int, newXML string) error {
	spans := f.spans()
	if i < 0 || i >= len(spans) {
		return fmt.Errorf("docx: paragraph %d out of range (have %d)", i, len(spans))
	}
	s := spans[i]
	f.doc = f.doc[:s[0]] + newXML + f.doc[s[1]:]
	f.docDirty = true
	f.changes++
	return nil
}

// SetParagraphText replaces the whole text of paragraph i, keeping its paragraph
// properties and the run properties of its first run.
func (f *File) SetParagraphText(i int, text string) error {
	p, err := f.paragraph(i)
	if err != nil {
		return err
	}
	out, err := setParagraphText(p, text)
	if err != nil {
		return err
	}
	return f.replaceParagraph(i, out)
}

// ReplaceInParagraph replaces the first occurrence of old in paragraph i.
func (f *File) ReplaceInParagraph(i int, old, new string) error {
	p, err := f.paragraph(i)
	if err != nil {
		return err
	}
	out, err := replaceInParagraph(p, old, new)
	if err != nil {
		return err
	}
	return f.replaceParagraph(i, out)
}

// InsertParagraphAfter inserts a new paragraph after i, cloning paragraph
// properties and run properties from paragraph template.
func (f *File) InsertParagraphAfter(i, template int, text string) error {
	tp, err := f.paragraph(template)
	if err != nil {
		return err
	}
	clone := reParaID.ReplaceAllString(tp, "")
	clone = reTextID.ReplaceAllString(clone, "")
	np, err := setParagraphText(clone, text)
	if err != nil {
		return err
	}
	spans := f.spans()
	if i < 0 || i >= len(spans) {
		return fmt.Errorf("docx: paragraph %d out of range (have %d)", i, len(spans))
	}
	at := spans[i][1]
	f.doc = f.doc[:at] + np + f.doc[at:]
	f.docDirty = true
	f.changes++
	return nil
}

// DeleteParagraph removes paragraph i.
func (f *File) DeleteParagraph(i int) error {
	spans := f.spans()
	if i < 0 || i >= len(spans) {
		return fmt.Errorf("docx: paragraph %d out of range (have %d)", i, len(spans))
	}
	s := spans[i]
	f.doc = f.doc[:s[0]] + f.doc[s[1]:]
	f.docDirty = true
	f.changes++
	return nil
}

// ---------------------------------------------------------------- run surgery

type runParts struct {
	open string
	rPr  string
	text string
}

func splitRun(raw string) (runParts, bool) {
	m := reRunHead.FindStringSubmatch(raw)
	if m == nil || !strings.HasSuffix(m[2], "</w:r>") {
		return runParts{}, false
	}
	rest := strings.TrimSuffix(m[2], "</w:r>")
	rPr := ""
	if mp := reRPr.FindStringSubmatch(rest); mp != nil {
		rPr, rest = mp[1], mp[2]
	}
	mt := reRunText.FindStringSubmatch(rest)
	if mt == nil {
		return runParts{}, false
	}
	return runParts{open: m[1], rPr: rPr, text: mt[1]}, true
}

func buildRun(p runParts, text string) string {
	return p.open + p.rPr + `<w:t xml:space="preserve">` + Escape(text) + `</w:t></w:r>`
}

// textItems breaks a paragraph into runs and the raw XML between them.
type runItem struct {
	isRun bool
	raw   string
	parts runParts
	txt   string
}

func splitItems(p string) []runItem {
	var items []runItem
	pos := 0
	for _, loc := range reRun.FindAllStringIndex(p, -1) {
		if loc[0] > pos {
			items = append(items, runItem{raw: p[pos:loc[0]]})
		}
		raw := p[loc[0]:loc[1]]
		if rp, ok := splitRun(raw); ok {
			items = append(items, runItem{isRun: true, parts: rp, txt: Unescape(rp.text)})
		} else {
			items = append(items, runItem{raw: raw})
		}
		pos = loc[1]
	}
	if pos < len(p) {
		items = append(items, runItem{raw: p[pos:]})
	}
	return items
}

func itemsText(items []runItem) string {
	var b strings.Builder
	for _, it := range items {
		if it.isRun {
			b.WriteString(it.txt)
		}
	}
	return b.String()
}

// splitAt guarantees a run boundary at a character offset. It returns the new
// item slice (the caller must keep it: inserting can reallocate) together with
// the index of the item that starts at the offset.
func splitAt(items []runItem, offset int) ([]runItem, int) {
	seen := 0
	for i := range items {
		it := items[i]
		if !it.isRun {
			continue
		}
		n := len(it.txt)
		switch {
		case seen == offset:
			return items, i
		case seen < offset && offset < seen+n:
			k := offset - seen
			head, tail := it.parts, it.parts
			head.text = it.txt[:k]
			tail.text = it.txt[k:]
			items = append(items, runItem{})
			copy(items[i+2:], items[i+1:])
			items[i] = runItem{isRun: true, parts: head, txt: head.text}
			items[i+1] = runItem{isRun: true, parts: tail, txt: tail.text}
			return items, i + 1
		}
		seen += n
	}
	return items, len(items)
}

func renderItems(items []runItem) string {
	var out strings.Builder
	for _, it := range items {
		if it.isRun {
			out.WriteString(buildRun(it.parts, it.txt))
		} else {
			out.WriteString(it.raw)
		}
	}
	return out.String()
}

func setParagraphText(p, text string) (string, error) {
	m := reParaHead.FindStringSubmatch(p)
	if m == nil || !strings.HasSuffix(m[2], "</w:p>") {
		return "", errors.New("docx: malformed paragraph")
	}
	open, rest := m[1], strings.TrimSuffix(m[2], "</w:p>")
	pPr := ""
	if mp := reParaPPr.FindStringSubmatch(rest); mp != nil {
		pPr, rest = mp[1], mp[2]
	}
	rpr := reAnyRPr.FindString(rest)
	run := `<w:r>` + rpr + `<w:t xml:space="preserve">` + Escape(text) + `</w:t></w:r>`
	return open + pPr + run + `</w:p>`, nil
}

func replaceInParagraph(p, old, new string) (string, error) {
	items := splitItems(p)
	text := itemsText(items)
	at := strings.Index(text, old)
	if at < 0 {
		return "", fmt.Errorf("%w: %q is not in this paragraph", ErrNotFound, clip(old))
	}
	items, start := splitAt(items, at)
	items, end := splitAt(items, at+len(old))
	found := false
	for i := start; i < end && i < len(items); i++ {
		if !items[i].isRun {
			// Raw fragments between runs (bookmarks, or the closing </w:p> when
			// the match ends at the end of the paragraph) must survive: blanking
			// them would drop paragraph boundaries.
			continue
		}
		if !found {
			parts := items[i].parts
			items[i] = runItem{isRun: true, parts: parts, txt: new}
			found = true
			continue
		}
		parts := items[i].parts
		items[i] = runItem{isRun: true, parts: parts, txt: ""}
	}
	if !found {
		return "", fmt.Errorf("%w: %q does not cover any text", ErrNotFound, clip(old))
	}
	return renderItems(items), nil
}

// stripEmptyCells keeps a table cell valid when a paragraph was removed from it.
func stripEmptyCells(doc string) string {
	var out strings.Builder
	pos := 0
	for {
		i := strings.Index(doc[pos:], "<w:tc>")
		if i < 0 {
			break
		}
		i += pos
		end := strings.Index(doc[i:], "</w:tc>")
		if end < 0 {
			break
		}
		end += i
		cell := doc[i:end]
		out.WriteString(doc[pos:i])
		if strings.Contains(cell, "<w:p") {
			out.WriteString(cell)
		} else {
			out.WriteString(cell + "<w:p/>")
		}
		pos = end
	}
	out.WriteString(doc[pos:])
	return out.String()
}

// ---------------------------------------------------------------- comments

// Comment is one comment thread.
type Comment struct {
	ID     string
	Author string
	Date   string
	Text   string // every message in the thread, joined by newline
	// ParentID is set for a threaded reply: the id of the comment it answers.
	ParentID string
}

// CommentOptions describes a comment to create.
type CommentOptions struct {
	Author   string
	Initials string
	Date     string // RFC3339; a fixed default is used when empty
	Text     string
	// Highlight, when set, is the substring of the target paragraph the comment
	// is attached to. That substring is highlighted in yellow. When empty the
	// whole paragraph is the anchor and nothing is highlighted.
	Highlight string
}

// Comments returns every comment in the package, in document order.
// ParentID is set for threaded replies.
func (f *File) Comments() ([]Comment, error) {
	if f.commentsName == "" {
		return nil, nil
	}
	// commentEx lives in its own part, not in the comments part.
	exParent := f.parentMap()
	isAnchor := map[string]bool{}
	for _, e := range regexp.MustCompile(`<w15:commentEx[^>]*/>`).FindAllString(f.commentExXML(), -1) {
		if pid := regexp.MustCompile(`w15:paraId="([^"]+)"`).FindStringSubmatch(e); pid != nil {
			isAnchor[pid[1]] = true
		}
	}

	reParaID := regexp.MustCompile(`w14:paraId="([0-9A-Fa-f]{8})"`)
	paraToComment := map[string]string{}
	anchorOf := map[string]string{}
	var out []Comment

	for _, m := range reCommentEl.FindAllString(f.comments, -1) {
		head := m[:strings.Index(m, ">")+1]
		c := Comment{ID: attr(head, "w:id"), Author: attr(head, "w:author"), Date: attr(head, "w:date")}
		var msgs []string
		for _, pm := range reParagraph.FindAllString(m, -1) {
			msgs = append(msgs, ParagraphText(pm))
		}
		c.Text = strings.TrimSpace(strings.Join(msgs, "\n"))

		var pids []string
		for _, p := range reParaID.FindAllStringSubmatch(m, -1) {
			pids = append(pids, p[1])
			paraToComment[p[1]] = c.ID
		}
		for _, p := range pids {
			if isAnchor[p] {
				anchorOf[c.ID] = p
			}
		}
		if anchorOf[c.ID] == "" && len(pids) > 0 {
			anchorOf[c.ID] = pids[len(pids)-1]
		}
		out = append(out, c)
	}
	for i := range out {
		if par, ok := exParent[anchorOf[out[i].ID]]; ok {
			out[i].ParentID = paraToComment[par]
		}
	}
	return out, nil
}

// AddComment attaches a comment to paragraph i and returns its id.
func (f *File) AddComment(i int, opt CommentOptions) (string, error) {
	p, err := f.paragraph(i)
	if err != nil {
		return "", err
	}
	if err := f.ensureCommentsPart(); err != nil {
		return "", err
	}
	id := f.nextCommentID()
	updated, err := anchorComment(p, id, opt.Highlight)
	if err != nil {
		return "", err
	}
	if err := f.replaceParagraph(i, updated); err != nil {
		return "", err
	}
	paraID := f.allocParaIDs(1)[0]
	f.appendComment(id, opt, paraID, "")
	return id, nil
}

// parentMap returns paraId -> paraIdParent for every threaded entry.
func (f *File) parentMap() map[string]string {
	out := map[string]string{}
	for _, e := range regexp.MustCompile(`<w15:commentEx[^>]*/>`).FindAllString(f.commentExXML(), -1) {
		pid := regexp.MustCompile(`w15:paraId="([^"]+)"`).FindStringSubmatch(e)
		if pid == nil {
			continue
		}
		if par := regexp.MustCompile(`w15:paraIdParent="([^"]+)"`).FindStringSubmatch(e); par != nil {
			out[pid[1]] = par[1]
		}
	}
	return out
}

// rootAnchorParaID walks up the reply chain and returns the paragraph of the
// thread's root message. The portal nests every reply under the root, so a reply
// to a reply must still point at the root.
func (f *File) rootAnchorParaID(commentID string) (string, error) {
	anchor, err := f.anchorParaID(commentID)
	if err != nil {
		return "", err
	}
	parents := f.parentMap()
	seen := map[string]bool{}
	for {
		if seen[anchor] {
			break
		}
		seen[anchor] = true
		up, ok := parents[anchor]
		if !ok || up == "" {
			break
		}
		anchor = up
	}
	return anchor, nil
}

// ReplyToComment adds a threaded reply.
//
// The portal models a reply as a SEPARATE <w:comment> element whose companion
// commentEx entry carries w15:paraIdParent pointing at the paragraph being
// replied to (the parent thread's anchor paragraph). Adding a paragraph to the
// parent element instead produces one comment with several paragraphs, which the
// editor renders as garbage. Observed on this portal:
//
//	<w:comment w:id="11" w:author="..." ...><w:p w14:paraId="0000001C">reply</w:p></w:comment>
//	<w15:commentEx w15:paraId="0000001C" w15:paraIdParent="0000001B" w15:done="0"/>
func (f *File) ReplyToComment(id, author, text string) error {
	if f.commentsName == "" {
		return fmt.Errorf("%w: comment %s", ErrNotFound, id)
	}
	parentAnchor, err := f.rootAnchorParaID(id)
	if err != nil {
		return err
	}
	newID := f.nextCommentID()
	paraID := f.allocParaIDs(1)[0]
	f.appendComment(newID, CommentOptions{Author: author, Text: text}, paraID, parentAnchor)
	return nil
}

// anchorParaID returns the paragraph that identifies a thread: the paraId its
// commentEx entry carries, falling back to the last paragraph of the element.
func (f *File) anchorParaID(commentID string) (string, error) {
	re := regexp.MustCompile(`(?s)<w:comment\b[^>]*w:id="` + regexp.QuoteMeta(commentID) + `"[^>]*>(.*?)</w:comment>`)
	m := re.FindStringSubmatch(f.comments)
	if m == nil {
		return "", fmt.Errorf("%w: comment %s", ErrNotFound, commentID)
	}
	pids := regexp.MustCompile(`w14:paraId="([0-9A-Fa-f]{8})"`).FindAllStringSubmatch(m[1], -1)
	if len(pids) == 0 {
		return "", fmt.Errorf("docx: comment %s has no paragraph id", commentID)
	}
	for _, e := range regexp.MustCompile(`<w15:commentEx[^>]*/>`).FindAllString(f.commentExXML(), -1) {
		pid := regexp.MustCompile(`w15:paraId="([^"]+)"`).FindStringSubmatch(e)
		if pid == nil {
			continue
		}
		for _, p := range pids {
			if pid[1] == p[1] {
				return pid[1], nil
			}
		}
	}
	return pids[len(pids)-1][1], nil
}

func (f *File) appendComment(id string, opt CommentOptions, paraID, parentParaID string) {
	head := fmt.Sprintf(`<w:comment w:id="%s" w:author="%s" w:date="%s" w:initials="%s">`,
		id, EscapeAttr(opt.Author), dateOr(opt.Date), EscapeAttr(opt.Initials))
	el := head + commentBody(paraID, opt.Text) + `</w:comment>`
	if strings.Contains(f.comments, "</w:comments>") {
		f.comments = strings.Replace(f.comments, "</w:comments>", el+"</w:comments>", 1)
	} else {
		f.comments = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<w:comments xmlns:w="` + nsW + `" xmlns:r="` + nsR + `" xmlns:w14="` + nsW14 + `">` +
			el + `</w:comments>`
	}
	f.commentsDirty = true
	f.changes++
	f.addCompanionEntries(paraID, parentParaID)
}

// commentBody renders one message in a thread. The markup follows what
// OnlyOffice itself writes, including the textId mirroring the paraId. The text
// is used verbatim: authors sign their own messages inside the text.
func commentBody(paraID, text string) string {
	return `<w:p w14:paraId="` + paraID + `" w14:textId="` + paraID + `">` +
		`<w:pPr><w:spacing w:line="240" w:after="0" w:lineRule="auto" w:before="0"/>` +
		`<w:ind w:firstLine="0" w:left="0" w:right="0"/><w:jc w:val="left"/></w:pPr>` +
		`<w:r><w:rPr><w:rFonts w:eastAsia="Arial" w:ascii="Arial" w:hAnsi="Arial" w:cs="Arial"/><w:sz w:val="22"/></w:rPr>` +
		`<w:t xml:space="preserve">` + Escape(text) + `</w:t></w:r></w:p>`
}

func (f *File) ensureCommentsPart() error {
	// The companion parts carry the thread structure, so they are needed even
	// when the package already has a comments part.
	f.ensureCompanionParts()
	if f.commentsName != "" {
		return nil
	}
	f.commentsName = partCommentsStd
	f.comments = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:comments xmlns:w="` + nsW + `" xmlns:r="` + nsR + `" xmlns:w14="` + nsW14 + `"></w:comments>`
	f.commentsDirty = true
	f.changes++
	f.parts[f.commentsName] = []byte(f.comments)

	if ct, ok := f.parts[partContentType]; ok && !strings.Contains(string(ct), "/word/comments.xml") {
		override := `<Override PartName="/word/comments.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.comments+xml"/>`
		f.parts[partContentType] = []byte(strings.Replace(string(ct), "</Types>", override+"</Types>", 1))
	}
	if rels, ok := f.parts[partRels]; ok && !strings.Contains(string(rels), "comments.xml") {
		rel := `<Relationship Id="` + nextRelID(string(rels)) + `" Type="` + relTypeComments + `" Target="comments.xml"/>`
		f.parts[partRels] = []byte(strings.Replace(string(rels), "</Relationships>", rel+"</Relationships>", 1))
	}
	return nil
}

// ensureCompanionParts creates the parts that carry the thread structure when a
// package does not have them yet: commentsExtended holds the parent link of a
// reply, commentsIds the durable id, commentsExtensible the timestamp. Word and
// OnlyOffice both read them, and OnlyOffice writes them on every save.
func (f *File) ensureCompanionParts() {
	decl := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`
	add := func(part, xml, contentType, relType, target string) {
		if _, ok := f.parts[part]; ok {
			return
		}
		f.parts[part] = []byte(xml)
		if ct, ok := f.parts[partContentType]; ok && !strings.Contains(string(ct), part) {
			override := `<Override PartName="/` + part + `" ContentType="` + contentType + `"/>`
			f.parts[partContentType] = []byte(strings.Replace(string(ct), "</Types>", override+"</Types>", 1))
		}
		if rels, ok := f.parts[partRels]; ok && !strings.Contains(string(rels), target) {
			rel := `<Relationship Id="` + nextRelID(string(rels)) + `" Type="` + relType + `" Target="` + target + `"/>`
			f.parts[partRels] = []byte(strings.Replace(string(rels), "</Relationships>", rel+"</Relationships>", 1))
		}
	}
	add(partCommentsExt,
		decl+`<w15:commentsEx xmlns:w="`+nsW+`" xmlns:w15="`+nsW15+`"></w15:commentsEx>`,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.commentsExtended+xml",
		nsR+"/commentsExtended", "commentsExtendedDocument.xml")
	add(partCommentsIDs,
		decl+`<w16cid:commentsIds xmlns:w16cid="`+nsW16+`"></w16cid:commentsIds>`,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.commentsIds+xml",
		nsR+"/commentsIds", "commentsIdsDocument.xml")
	add(partCommentsExt2,
		decl+`<w16cex:commentsExtensible xmlns:w16cex="`+nsCEX+`"></w16cex:commentsExtensible>`,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.commentsExtensible+xml",
		nsR+"/commentsExtensible", "commentsExtensibleDocument.xml")
}

func (f *File) nextCommentID() string {
	max := -1
	for _, m := range regexp.MustCompile(`<w:comment\b[^>]*w:id="(\d+)"`).FindAllStringSubmatch(f.comments, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil && n > max {
			max = n
		}
	}
	return strconv.Itoa(max + 1)
}

// allocParaIDs reserves n consecutive unused 8 digit hex ids, the convention
// that links comment paragraphs across the companion parts.
func (f *File) allocParaIDs(n int) []string {
	max := 0
	for _, m := range regexp.MustCompile(`w14:paraId="([0-9A-Fa-f]{8})"`).FindAllStringSubmatch(f.comments, -1) {
		if v, err := strconv.ParseUint(m[1], 16, 32); err == nil && int(v) > max {
			max = int(v)
		}
	}
	out := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, fmt.Sprintf("%08X", max+i))
	}
	return out
}

// commentExXML returns the commentsExtended part, or "" when the package has
// none. The parent link of a threaded reply lives only there.
func (f *File) commentExXML() string {
	if b, ok := f.parts[partCommentsExt]; ok {
		return string(b)
	}
	return ""
}

// addCompanionEntries registers a comment in the OnlyOffice helper parts. One
// entry per <w:comment> element in each part, keyed to that element's paragraph.
// parentParaID, when set, marks the comment as a reply to that paragraph.
func (f *File) addCompanionEntries(paraID, parentParaID string) {
	durable := randHex8()
	if b, ok := f.parts[partCommentsIDs]; ok {
		s := string(b)
		e := `<w16cid:commentId w16cid:paraId="` + paraID + `" w16cid:durableId="` + durable + `"/>`
		if !strings.Contains(s, e) {
			f.parts[partCommentsIDs] = []byte(strings.Replace(s, "</w16cid:commentsIds>", e+"</w16cid:commentsIds>", 1))
		}
	}
	if b, ok := f.parts[partCommentsExt2]; ok {
		s := string(b)
		e := `<w16cex:commentExtensible w16cex:durableId="` + durable + `" w16cex:dateUtc="` + dateOr("") + `"/>`
		if !strings.Contains(s, e) {
			f.parts[partCommentsExt2] = []byte(strings.Replace(s, "</w16cex:commentsExtensible>", e+"</w16cex:commentsExtensible>", 1))
		}
	}
	if b, ok := f.parts[partCommentsExt]; ok {
		s := string(b)
		ex := `<w15:commentEx w15:paraId="` + paraID + `"`
		if parentParaID != "" {
			ex += ` w15:paraIdParent="` + parentParaID + `"`
		}
		e := ex + ` w15:done="0"/>`
		if !strings.Contains(s, e) {
			f.parts[partCommentsExt] = []byte(strings.Replace(s, "</w15:commentsEx>", e+"</w15:commentsEx>", 1))
		}
	}
}

// anchorComment wraps the target of a paragraph in a comment range. With a
// highlight substring the range covers that substring and marks it yellow;
// without one the range covers the whole paragraph.
func anchorComment(p, id, highlight string) (string, error) {
	if highlight == "" {
		m := reParaHead.FindStringSubmatch(p)
		if m == nil {
			return "", errors.New("docx: malformed paragraph")
		}
		open, rest := m[1], m[2]
		at := len(open)
		if mp := reParaPPr.FindStringSubmatch(rest); mp != nil {
			at = len(open) + len(mp[1])
		}
		out := p[:at] + `<w:commentRangeStart w:id="` + id + `"/>` + p[at:]
		return strings.TrimSuffix(out, "</w:p>") +
			`<w:commentRangeEnd w:id="` + id + `"/><w:r><w:commentReference w:id="` + id + `"/></w:r></w:p>`, nil
	}
	hl, err := highlightSpan(p, highlight, id)
	if err != nil {
		return "", err
	}
	return hl, nil
}

// highlightSpan marks a substring yellow and wraps it in the comment range.
func highlightSpan(p, sub, id string) (string, error) {
	items := splitItems(p)
	text := itemsText(items)
	at := strings.Index(text, sub)
	if at < 0 {
		return "", fmt.Errorf("%w: %q is not in this paragraph", ErrNotFound, clip(sub))
	}
	items, start := splitAt(items, at)
	items, end := splitAt(items, at+len(sub))
	lastRun := -1
	for i := start; i < end && i < len(items); i++ {
		if items[i].isRun {
			lastRun = i
		}
	}
	if lastRun < 0 {
		return "", errors.New("docx: highlight does not cover a text run")
	}
	for i := start; i < end && i < len(items); i++ {
		if !items[i].isRun {
			continue
		}
		rp := items[i].parts
		rp.rPr = withHighlight(rp.rPr)
		items[i] = runItem{isRun: true, parts: rp, txt: items[i].txt}
	}
	// The closing markers must land after the last run of the span, never after a
	// trailing raw fragment: putting them past </w:p> would place the reference
	// outside the paragraph, which is invalid WordprocessingML.
	var out strings.Builder
	for i, it := range items {
		if i == start {
			out.WriteString(`<w:commentRangeStart w:id="` + id + `"/>`)
		}
		if it.isRun {
			out.WriteString(buildRun(it.parts, it.txt))
		} else {
			out.WriteString(it.raw)
		}
		if i == lastRun {
			out.WriteString(`<w:commentRangeEnd w:id="` + id + `"/><w:r><w:commentReference w:id="` + id + `"/></w:r>`)
		}
	}
	return out.String(), nil
}

func withHighlight(rPr string) string {
	if rPr == "" {
		return `<w:rPr><w:highlight w:val="yellow"/></w:rPr>`
	}
	rPr = reHighlight.ReplaceAllString(rPr, "")
	return strings.Replace(rPr, "</w:rPr>", `<w:highlight w:val="yellow"/></w:rPr>`, 1)
}

// ---------------------------------------------------------------- helpers

func attr(tag, name string) string {
	if m := regexp.MustCompile(regexp.QuoteMeta(name) + `="([^"]*)"`).FindStringSubmatch(tag); m != nil {
		return Unescape(m[1])
	}
	return ""
}

func dateOr(s string) string {
	if s != "" {
		return s
	}
	return "2026-10-14T12:00:00Z"
}

func nextRelID(rels string) string {
	max := 0
	for _, m := range regexp.MustCompile(`Id="rId(\d+)"`).FindAllStringSubmatch(rels, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil && n > max {
			max = n
		}
	}
	return "rId" + strconv.Itoa(max+1000)
}

func clip(s string) string {
	if len([]rune(s)) <= 60 {
		return s
	}
	return string([]rune(s)[:60]) + "…"
}

// Escape writes text safe for an XML text node.
func Escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// EscapeAttr writes text safe for a double quoted XML attribute.
func EscapeAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

// Unescape resolves the five named entities and numeric character references.
func Unescape(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ';')
		if end < 0 || end > 12 {
			b.WriteByte(s[i])
			i++
			continue
		}
		ent := s[i+1 : i+end]
		switch ent {
		case "amp":
			b.WriteByte('&')
		case "lt":
			b.WriteByte('<')
		case "gt":
			b.WriteByte('>')
		case "quot":
			b.WriteByte('"')
		case "apos":
			b.WriteByte('\'')
		default:
			if !strings.HasPrefix(ent, "#") {
				b.WriteString(s[i : i+end+1])
				break
			}
			num, base := ent[1:], 10
			if strings.HasPrefix(num, "x") || strings.HasPrefix(num, "X") {
				num, base = num[1:], 16
			}
			if n, err := strconv.ParseInt(num, base, 32); err == nil {
				b.WriteRune(rune(n))
			} else {
				b.WriteString(s[i : i+end+1])
			}
		}
		i += end + 1
	}
	return b.String()
}

// randHex8 returns a fresh durable id for the OnlyOffice helper parts.
func randHex8() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000"
	}
	return strings.ToUpper(fmt.Sprintf("%02X%02X%02X%02X", b[0], b[1], b[2], b[3]))
}
