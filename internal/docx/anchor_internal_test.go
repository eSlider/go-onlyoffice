package docx

import (
	"regexp"
	"strings"
	"testing"
)

// assertMarkersInsideParagraph is the regression guard for the anchor bug: the
// comment range and its reference must sit inside the paragraph, before the
// closing </w:p>. Word and OnlyOffice both produce paragraphs that end in runs
// with no <w:t> at all, and an off-by-one there used to push the markers past
// the paragraph boundary while still looking like valid XML.
func assertMarkersInsideParagraph(t *testing.T, paragraph, id string) {
	t.Helper()
	start := strings.Index(paragraph, `<w:commentRangeStart w:id="`+id+`"/>`)
	end := strings.Index(paragraph, `<w:commentReference w:id="`+id+`"/>`)
	if start < 0 || end < 0 {
		t.Fatalf("markers missing in %q", paragraph)
	}
	if end < start {
		t.Fatalf("reference before range start in %q", paragraph)
	}
	for _, marker := range []string{`<w:commentRangeEnd w:id="` + id + `"/>`, `<w:commentReference w:id="` + id + `"/>`} {
		i := strings.Index(paragraph, marker)
		before := paragraph[:i]
		if strings.Contains(before, "</w:p>") {
			t.Fatalf("marker %s sits outside the paragraph: %q", marker, paragraph)
		}
	}
	if !strings.Contains(paragraph[end:], "</w:p>") {
		t.Fatalf("paragraph does not close after the reference: %q", paragraph)
	}
}

const rprBold = `<w:rPr><w:rFonts w:ascii="Calibri"/><w:b/></w:rPr>`

func TestAnchorCommentWithHighlightInsideParagraph(t *testing.T) {
	tests := []struct {
		name      string
		paragraph string
		highlight string
	}{
		{
			name:      "single run",
			paragraph: `<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r>` + rprBold + `<w:t xml:space="preserve">FOURTH CLAUSE</w:t></w:r></w:p>`,
			highlight: "FOURTH CLAUSE",
		},
		{
			// the shape Word writes: a text run followed by empty runs with no <w:t>
			name: "trailing runs without text",
			paragraph: `<w:p><w:pPr><w:jc w:val="center"/></w:pPr>` +
				`<w:r>` + rprBold + `<w:t xml:space="preserve">FOURTH CLAUSE</w:t></w:r>` +
				`<w:r>` + rprBold + `</w:r><w:r>` + rprBold + `</w:r></w:p>`,
			highlight: "FOURTH",
		},
		{
			name: "highlight spanning several runs",
			paragraph: `<w:p><w:r><w:t xml:space="preserve">the </w:t></w:r>` +
				`<w:r><w:t xml:space="preserve">use permit </w:t></w:r>` +
				`<w:r><w:t xml:space="preserve">no. 77</w:t></w:r>` +
				`<w:r><w:t xml:space="preserve"> issued by the council</w:t></w:r></w:p>`,
			highlight: "use permit no. 77",
		},
		{
			name:      "paragraph ending with a bookmark fragment",
			paragraph: `<w:p><w:r><w:t xml:space="preserve">Rent is due on the 8th.</w:t></w:r><w:bookmarkEnd w:id="4"/></w:p>`,
			highlight: "the 8th",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := anchorComment(tc.paragraph, "7", tc.highlight)
			if err != nil {
				t.Fatalf("anchorComment: %v", err)
			}
			assertMarkersInsideParagraph(t, out, "7")
			if !strings.Contains(out, `<w:highlight w:val="yellow"/>`) {
				t.Fatalf("highlight not applied: %q", out)
			}
			if got, want := ParagraphText(out), ParagraphText(tc.paragraph); got != want {
				t.Fatalf("text changed: got %q want %q", got, want)
			}
			if strings.Count(out, "</w:p>") != strings.Count(tc.paragraph, "</w:p>") {
				t.Fatalf("paragraph boundaries changed: %q", out)
			}
		})
	}
}

func TestAnchorWholeParagraphInsideParagraph(t *testing.T) {
	p := `<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:t xml:space="preserve">NINTH CLAUSE</w:t></w:r><w:r></w:r></w:p>`
	out, err := anchorComment(p, "9", "")
	if err != nil {
		t.Fatalf("anchorComment: %v", err)
	}
	assertMarkersInsideParagraph(t, out, "9")
	if strings.Contains(out, "<w:highlight") {
		t.Fatalf("whole paragraph anchor must not highlight")
	}
	// the range opens before the first run and closes after the last one
	if !regexp.MustCompile(`<w:commentRangeStart w:id="9"/><w:r>`).MatchString(out) {
		t.Fatalf("range start not immediately before the first run: %q", out)
	}
}

func TestSplitAtKeepsSliceAligned(t *testing.T) {
	items := splitItems(`<w:p><w:r><w:t xml:space="preserve">ab</w:t></w:r><w:r><w:t xml:space="preserve">cd</w:t></w:r></w:p>`)
	items, i := splitAt(items, 1)
	items, j := splitAt(items, 3)
	if i >= j {
		t.Fatalf("boundaries out of order: %d %d", i, j)
	}
	if got := itemsText(items); got != "abcd" {
		t.Fatalf("text lost while splitting: %q", got)
	}
	if !strings.HasPrefix(items[i].txt, "b") {
		t.Fatalf("boundary 1 should start a run at 'b', got %q", items[i].txt)
	}
	if !strings.HasPrefix(items[j].txt, "d") {
		t.Fatalf("boundary 3 should start a run at 'd', got %q", items[j].txt)
	}
}

// Regression: a match that ends at the very end of the paragraph text used to
// overwrite the trailing raw fragment, which is where </w:p> lives. The result
// still parsed as XML only because the extra content happened to sit elsewhere.
func TestReplaceInParagraphKeepsTrailingFragments(t *testing.T) {
	tests := []struct {
		name      string
		paragraph string
		old       string
		new       string
		want      string
	}{
		{
			name:      "match ends at end of text",
			paragraph: `<w:p><w:pPr><w:jc w:val="both"/></w:pPr><w:r><w:t xml:space="preserve">belongings are excluded and and are his responsibility to insure.</w:t></w:r><w:r><w:rPr><w:b/></w:rPr></w:r></w:p>`,
			old:       "and and are his responsibility to insure.",
			new:       "and remain their responsibility to insure.",
			want:      "belongings are excluded and remain their responsibility to insure.",
		},
		{
			name:      "match in the middle keeps both neighbours",
			paragraph: `<w:p><w:r><w:t xml:space="preserve">payable </w:t></w:r><w:r><w:t xml:space="preserve">up to the 5th day</w:t></w:r><w:r><w:t xml:space="preserve"> of the month</w:t></w:r><w:r></w:r></w:p>`,
			old:       "up to the 5th day",
			new:       "up to the 8th day",
			want:      "payable up to the 8th day of the month",
		},
		{
			name:      "match ends at end with no trailing runs",
			paragraph: `<w:p><w:r><w:t xml:space="preserve">no garden or pool</w:t></w:r></w:p>`,
			old:       "no garden or pool",
			new:       "no garden or swimming pool",
			want:      "no garden or swimming pool",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := replaceInParagraph(tc.paragraph, tc.old, tc.new)
			if err != nil {
				t.Fatalf("replaceInParagraph: %v", err)
			}
			if got := ParagraphText(out); got != tc.want {
				t.Fatalf("text: got %q want %q", got, tc.want)
			}
			if open, closed := strings.Count(out, "<w:p")-strings.Count(out, "<w:pPr"), strings.Count(out, "</w:p>"); open != closed {
				t.Fatalf("paragraph boundary lost: %d open vs %d closed in %q", open, closed, out)
			}
			if !strings.HasSuffix(out, "</w:p>") {
				t.Fatalf("paragraph no longer closes with </w:p>: %q", out)
			}
		})
	}
}
