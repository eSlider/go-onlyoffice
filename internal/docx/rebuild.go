package docx

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ThreadSpec is one conversation. The first message is the root comment, every
// later message is a threaded reply to the message before it.
type ThreadSpec struct {
	Messages []MessageSpec
}

// MessageSpec is one message in a thread. Date is RFC3339 and is written to the
// comment, the commentsExtended entry and the commentsExtensible entry, so the
// panel shows when each message was actually written. Every message after the
// first is a reply to the thread's root message, which is how the portal nests
// a thread.
type MessageSpec struct {
	Author string
	Text   string
	Date   string
}

var reCommentAnchor = regexp.MustCompile(
	`<w:commentRangeStart[^>]*/>|<w:commentRangeEnd[^>]*/>|<w:r><w:commentReference[^>]*/></w:r>|<w:commentReference[^>]*/>`)

// RebuildComments replaces every comment in the package with the given threads.
//
// This is how a document gets a clean set of comments: one <w:comment> element
// per message, the author on each element, its own date, and the reply
// relationship expressed with w15:paraIdParent in commentsExtendedDocument.xml.
// Neither the portal nor Word merges two people's messages into one element, and
// neither relies on a "Name: " prefix inside the text.
//
// Comment anchors in document.xml are dropped: OnlyOffice itself saves comments
// without them, and stale anchors would point at ids that no longer exist. Run
// highlighting is left untouched.
func (f *File) RebuildComments(threads []ThreadSpec) error {
	if len(threads) == 0 {
		return errors.New("docx: no threads given")
	}
	total := 0
	for _, t := range threads {
		if len(t.Messages) == 0 {
			return errors.New("docx: empty thread")
		}
		for _, m := range t.Messages {
			total++
			if strings.TrimSpace(m.Text) == "" {
				return fmt.Errorf("docx: message %d has empty text", total)
			}
		}
	}
	if err := f.ensureCommentsPart(); err != nil {
		return err
	}

	// Ids and paragraph ids are allocated up front so a reply can reference the
	// paragraph of the message it answers.
	paraIDs := make([]string, total)
	for i := range paraIDs {
		paraIDs[i] = fmt.Sprintf("%08X", i+1)
	}

	type msg struct {
		id, paraID, durable string
		spec                MessageSpec
		parent              string
	}
	all := make([]msg, 0, total)
	k := 0
	for _, t := range threads {
		// Every reply points at the thread's ROOT paragraph. The portal nests a
		// thread under its root only, so a reply to a reply has to reference the
		// root as well or the editor will not show it.
		root := paraIDs[k]
		for i, m := range t.Messages {
			parent := ""
			if i > 0 {
				parent = root
			}
			all = append(all, msg{
				id:      fmt.Sprintf("%d", k),
				paraID:  paraIDs[k],
				durable: randHex8(),
				spec:    m,
				parent:  parent,
			})
			k++
		}
	}

	decl := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`
	var comments, ex, ids, ext strings.Builder
	// w14 must be declared: the message paragraphs carry w14:paraId and w14:textId.
	comments.WriteString(decl + `<w:comments xmlns:w="` + nsW + `" xmlns:r="` + nsR +
		`" xmlns:w14="` + nsW14 + `">`)
	ex.WriteString(decl + `<w15:commentsEx xmlns:w="` + nsW + `" xmlns:w15="` + nsW15 + `">`)
	ids.WriteString(decl + `<w16cid:commentsIds xmlns:w16cid="` + nsW16 + `">`)
	ext.WriteString(decl + `<w16cex:commentsExtensible xmlns:w16cex="` + nsCEX + `">`)

	for _, m := range all {
		when := dateOr(m.spec.Date)
		comments.WriteString(fmt.Sprintf(`<w:comment w:id="%s" w:author="%s" w:date="%s" w:initials="%s">`,
			m.id, EscapeAttr(m.spec.Author), EscapeAttr(when), EscapeAttr(initialsOf(m.spec.Author))))
		comments.WriteString(commentBody(m.paraID, m.spec.Text))
		comments.WriteString(`</w:comment>`)

		entry := `<w15:commentEx w15:paraId="` + m.paraID + `"`
		if m.parent != "" {
			entry += ` w15:paraIdParent="` + m.parent + `"`
		}
		ex.WriteString(entry + ` w15:done="0"/>`)
		ids.WriteString(`<w16cid:commentId w16cid:paraId="` + m.paraID + `" w16cid:durableId="` + m.durable + `"/>`)
		ext.WriteString(`<w16cex:commentExtensible w16cex:durableId="` + m.durable + `" w16cex:dateUtc="` + EscapeAttr(when) + `"/>`)
	}
	comments.WriteString(`</w:comments>`)
	ex.WriteString(`</w15:commentsEx>`)
	ids.WriteString(`</w16cid:commentsIds>`)
	ext.WriteString(`</w16cex:commentsExtensible>`)

	f.comments = comments.String()
	f.commentsDirty = true
	f.changes++

	f.ensureCompanionParts()
	f.parts[partCommentsExt] = []byte(ex.String())
	f.parts[partCommentsIDs] = []byte(ids.String())
	f.parts[partCommentsExt2] = []byte(ext.String())

	// Drop anchors, they would reference ids that no longer exist.
	f.doc = reCommentAnchor.ReplaceAllString(f.doc, "")
	f.docDirty = true
	return nil
}

// initialsOf "Alice Anderson" -> "AA".
func initialsOf(name string) string {
	var b strings.Builder
	for _, f := range strings.Fields(name) {
		r := []rune(f)
		if len(r) > 0 {
			b.WriteString(strings.ToUpper(string(r[0])))
		}
	}
	if b.Len() > 2 {
		return b.String()[:2]
	}
	return b.String()
}
