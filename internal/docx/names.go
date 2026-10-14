package docx

import (
	"fmt"
	"regexp"
	"strings"
)

// Namespace URIs. These are fixed by the OOXML standard. The PREFIX a document
// binds them to is not: a file may write w15, x15 or anything else for the same
// URI, so every lookup here resolves the URI to the prefix actually in use.
type NS string

const (
	NSWord     NS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	NSW2010    NS = "http://schemas.microsoft.com/office/word/2010/wordml"
	NSW2012    NS = "http://schemas.microsoft.com/office/word/2012/wordml"
	NSW2016CID NS = "http://schemas.microsoft.com/office/word/2016/wordml/cid"
	NSW2018CEX NS = "http://schemas.microsoft.com/office/word/2018/wordml/cex"
)

// Name is a qualified element or attribute name: a namespace URI plus a local
// name. Nothing in this package matches on a prefix alone.
type Name struct {
	NS    NS
	Local string
}

// Elements and attributes the package reads or writes.
var (
	ElComments      = Name{NSWord, "comments"}
	ElComment       = Name{NSWord, "comment"}
	ElParagraph     = Name{NSWord, "p"}
	ElRun           = Name{NSWord, "r"}
	ElText          = Name{NSWord, "t"}
	ElCommentEx     = Name{NSW2012, "commentEx"}
	ElCommentID     = Name{NSW2016CID, "commentId"}
	ElCommentExt    = Name{NSW2018CEX, "commentExtensible"}
	AttrParaID      = Name{NSW2010, "paraId"}
	AttrTextID      = Name{NSW2010, "textId"}
	AttrParaIDParnt = Name{NSW2012, "paraIdParent"}
	AttrDone        = Name{NSW2012, "done"}
	AttrDurableID   = Name{NSW2016CID, "durableId"}
	AttrDateUTC     = Name{NSW2018CEX, "dateUtc"}
	AttrWID         = Name{NSWord, "id"}
	AttrAuthor      = Name{NSWord, "author"}
	AttrDate        = Name{NSWord, "date"}
	AttrInitials    = Name{NSWord, "initials"}
)

// canonicalPrefix is what this package emits when it creates a part or adds an
// element. Word and OnlyOffice both use these prefixes, so generated markup
// stays familiar to every reader.
var canonicalPrefix = map[NS]string{
	NSWord:     "w",
	NSW2010:    "w14",
	NSW2012:    "w15",
	NSW2016CID: "w16cid",
	NSW2018CEX: "w16cex",
}

var reXMLNS = regexp.MustCompile(`xmlns:([A-Za-z0-9_.-]+)="([^"]*)"`)

// resolver maps namespace URIs to the prefixes one specific part uses.
type resolver map[NS]string

// newResolver reads the xmlns declarations of a part. Declarations may sit on
// any element, so the whole part is scanned; the first binding of a URI wins.
func newResolver(part string) resolver {
	r := resolver{}
	for _, m := range reXMLNS.FindAllStringSubmatch(part, -1) {
		prefix, uri := m[1], m[2]
		if prefix == "" {
			continue
		}
		for _, ns := range allNS {
			if uri == string(ns) {
				if _, seen := r[ns]; !seen {
					r[ns] = prefix
				}
			}
		}
	}
	return r
}

var allNS = []NS{NSWord, NSW2010, NSW2012, NSW2016CID, NSW2018CEX}

// canonical is the resolver used when no part context is available.
func canonical() resolver {
	r := resolver{}
	for ns, p := range canonicalPrefix {
		r[ns] = p
	}
	return r
}

// qname returns "prefix:local" as written in the part.
func (r resolver) qname(n Name) string {
	if p, ok := r[n.NS]; ok && p != "" {
		return p + ":" + n.Local
	}
	if p, ok := canonicalPrefix[n.NS]; ok && p != "" {
		return p + ":" + n.Local
	}
	return n.Local
}

// local returns just the local name, for attributes that may appear unprefixed.
func (r resolver) local(n Name) string { return n.Local }

// element matches <prefix:local ...> or </prefix:local>.
func (r resolver) element(n Name) string {
	return `<` + r.qname(n) + `\b`
}

// elementComplete matches an empty element <prefix:local ... />.
func (r resolver) elementComplete(n Name) string {
	return `<` + r.qname(n) + `\b[^>]*/>`
}

// attr matches prefix:local="...".
func (r resolver) attr(n Name) string {
	return r.qname(n) + `="([^"]*)"`
}

// hasName reports whether a start tag carries the given name.
func (r resolver) hasName(tag string, n Name) bool {
	return strings.HasPrefix(strings.TrimLeft(tag, "<"), r.qname(n))
}

// attrValue reads one attribute from a start tag.
func (r resolver) attrValue(tag string, n Name) string {
	re := regexp.MustCompile(regexp.QuoteMeta(r.qname(n)) + `="([^"]*)"`)
	if m := re.FindStringSubmatch(tag); m != nil {
		return Unescape(m[1])
	}
	return ""
}

// decl renders an xmlns declaration for a namespace, used when creating parts.
func (r resolver) decl(prefix string, ns NS) string {
	return fmt.Sprintf(`xmlns:%s="%s"`, prefix, ns)
}

// canonicalise makes every prefix that is bound to a known OOXML namespace URI
// match the canonical one, so that the rest of the package can look names up by
// URI without caring what the producing application chose to write.
//
// A prefix carries no meaning: only the URI does. Rather than thread a resolver
// through every lookup, the prefix is normalised once, on load, and only inside
// tags, never in text content. A part that already uses the canonical prefixes
// is returned unchanged, so untouched parts stay byte for byte identical.
func canonicalise(part string) string {
	seen := newResolver(part)
	rename := map[string]string{}
	for _, ns := range allNS {
		prefix, ok := seen[ns]
		if !ok {
			continue
		}
		want := canonicalPrefix[ns]
		if prefix != want {
			rename[prefix] = want
		}
	}
	if len(rename) == 0 {
		return part
	}

	var out strings.Builder
	out.Grow(len(part))
	for i := 0; i < len(part); {
		lt := strings.IndexByte(part[i:], '<')
		if lt < 0 {
			out.WriteString(part[i:])
			break
		}
		lt += i
		out.WriteString(part[i:lt]) // text node, left alone
		gt := strings.IndexByte(part[lt:], '>')
		if gt < 0 {
			out.WriteString(part[lt:])
			break
		}
		gt += lt
		tag := part[lt : gt+1]
		for from, to := range rename {
			tag = strings.ReplaceAll(tag, "xmlns:"+from+`="`, "xmlns:"+to+`="`)
			tag = strings.ReplaceAll(tag, "<"+from+":", "<"+to+":")
			tag = strings.ReplaceAll(tag, "</"+from+":", "</"+to+":")
			tag = strings.ReplaceAll(tag, " "+from+":", " "+to+":")
		}
		out.WriteString(tag)
		i = gt + 1
	}
	return out.String()
}
