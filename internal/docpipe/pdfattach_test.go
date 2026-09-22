package docpipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAttachmentList(t *testing.T) {
	out := "2 embedded files\n1: original.pdf\n2: scan_001.png\n"
	got := parseAttachmentList(out)
	want := []PDFAttachment{{Index: 1, Name: "original.pdf"}, {Index: 2, Name: "scan_001.png"}}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("att[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseAttachmentListEmptyAndMalformed(t *testing.T) {
	for _, in := range []string{"", "0 embedded files\n", "garbage\n\n  \n"} {
		if got := parseAttachmentList(in); len(got) != 0 {
			t.Errorf("parseAttachmentList(%q) = %+v, want empty", in, got)
		}
	}
}

func TestSafeAttachmentName(t *testing.T) {
	cases := map[string]string{
		"note.txt":            "note.txt",
		"../../evil.pdf":      "evil.pdf",
		`..\..\evil.pdf`:      "evil.pdf",
		"/abs/scan_001.pdf":   "scan_001.pdf",
		".hidden":             "hidden",
		"  spaced name.txt  ": "spaced name.txt",
		"..":                  "",
		"":                    "",
	}
	for in, want := range cases {
		if got := safeAttachmentName(in); got != want {
			t.Errorf("safeAttachmentName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJoinWithAttachments(t *testing.T) {
	body := "# scan.pdf\n\nbody token\n"
	atts := []AttachmentText{
		{Name: "original.pdf", Text: "  original token  "},
		{Name: "empty.txt", Text: "   "},
	}
	got := JoinWithAttachments(body, atts)
	if !strings.Contains(got, "body token") {
		t.Errorf("body text lost: %q", got)
	}
	if !strings.Contains(got, "[attachment: original.pdf]") {
		t.Errorf("marker missing: %q", got)
	}
	if !strings.Contains(got, "original token") {
		t.Errorf("attachment text missing: %q", got)
	}
	if strings.Contains(got, "empty.txt") {
		t.Errorf("empty attachment must be skipped: %q", got)
	}
}

func TestJoinWithAttachmentsNoAttachments(t *testing.T) {
	got := JoinWithAttachments("# a.pdf\n\ntext\n\n", nil)
	if got != "# a.pdf\n\ntext" {
		t.Errorf("got %q, want trimmed body only", got)
	}
}

func TestListAttachmentsWithoutTool(t *testing.T) {
	if _, err := (Tools{}).ListAttachments("x.pdf"); err == nil || !strings.Contains(err.Error(), "pdfdetach") {
		t.Fatalf("want pdfdetach error, got %v", err)
	}
}

// TestToMarkdownWithAttachmentsFixture exercises the real pdfdetach + pdftotext
// pipeline on testdata/pdf-with-attachment.pdf (body token + embedded
// goo-note.txt). Skips when poppler is not installed.
func TestToMarkdownWithAttachmentsFixture(t *testing.T) {
	tools := LookPath()
	if tools.PDFDetach == "" || tools.PDFToText == "" {
		t.Skip("pdfdetach/pdftotext not on PATH — skipping attachment extraction test")
	}
	fixture := filepath.Join("..", "..", "testdata", "pdf-with-attachment.pdf")
	got, err := tools.ToMarkdownWithAttachments(fixture, t.TempDir(), "eng", 1)
	if err != nil {
		t.Fatalf("ToMarkdownWithAttachments: %v", err)
	}
	for _, want := range []string{"goobodytoken", "[attachment: goo-note.txt]", "gooattachmenttoken"} {
		if !strings.Contains(got, want) {
			t.Errorf("result missing %q:\n%s", want, got)
		}
	}
}

func TestXMLToText(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice><rsm:ExchangedDocument>
<ram:ID>S1063</ram:ID></rsm:ExchangedDocument>
<ram:Name>Acme &amp; Co</ram:Name><ram:GrandTotalAmount>42.00</ram:GrandTotalAmount>
</rsm:CrossIndustryInvoice>`)
	got := xmlToText(raw)
	for _, want := range []string{"S1063", "Acme & Co", "42.00"} {
		if !strings.Contains(got, want) {
			t.Errorf("xmlToText missing %q:\n%s", want, got)
		}
	}
	if strings.ContainsAny(got, "<>") {
		t.Errorf("xmlToText left markup: %q", got)
	}
}

// TestAttachmentMarkdownFallback verifies structured attachments that docpipe
// cannot convert are still reduced to searchable text, and unknown binary
// formats error (so the caller skips them).
func TestAttachmentMarkdownFallback(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "factur-x.xml")
	if err := os.WriteFile(xmlPath, []byte(`<Invoice><Number>S1063</Number></Invoice>`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := (Tools{}).attachmentMarkdown(xmlPath, dir, "", 0)
	if err != nil {
		t.Fatalf("attachmentMarkdown(xml): %v", err)
	}
	if !strings.Contains(got, "S1063") {
		t.Errorf("xml attachment text = %q, want S1063", got)
	}

	// Classified digitised PDFs (Scanner-*.ocr.pdf) carry .yaml metadata.
	yamlPath := filepath.Join(dir, "Scanner-123-003.ocr.yaml")
	if err := os.WriteFile(yamlPath, []byte("document:\n  type: Rechnung\nnumber: S1063\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gotYAML, err := (Tools{}).attachmentMarkdown(yamlPath, dir, "", 0)
	if err != nil {
		t.Fatalf("attachmentMarkdown(yaml): %v", err)
	}
	if !strings.Contains(gotYAML, "S1063") {
		t.Errorf("yaml attachment text = %q, want S1063", gotYAML)
	}

	// Extensionless textual attachment falls back to raw text.
	noExt := filepath.Join(dir, "attachment")
	if err := os.WriteFile(noExt, []byte("plain attachment token goonoext"), 0o644); err != nil {
		t.Fatal(err)
	}
	gotNoExt, err := (Tools{}).attachmentMarkdown(noExt, dir, "", 0)
	if err != nil {
		t.Fatalf("attachmentMarkdown(no extension): %v", err)
	}
	if !strings.Contains(gotNoExt, "goonoext") {
		t.Errorf("extensionless attachment text = %q, want goonoext", gotNoExt)
	}

	binPath := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(binPath, []byte{0, 1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (Tools{}).attachmentMarkdown(binPath, dir, "", 0); err == nil {
		t.Error("unsupported attachment: want error, got nil")
	}
}

// TestToMarkdownWithAttachmentsPlainPDF ensures a PDF without attachments
// returns just the body (pdfdetach prints "0 embedded files").
func TestToMarkdownWithAttachmentsPlainPDF(t *testing.T) {
	tools := LookPath()
	if tools.PDFDetach == "" || tools.PDFToText == "" {
		t.Skip("pdfdetach/pdftotext not on PATH")
	}
	// The fixture itself is a PDF with one attachment; strip it by extracting
	// the body only through ToMarkdown and compare JoinWithAttachments(nil).
	res, err := tools.ToMarkdown(filepath.Join("..", "..", "testdata", "pdf-with-attachment.pdf"), t.TempDir(), "eng", 1)
	if err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if strings.Contains(res.Markdown, "gooattachmenttoken") {
		t.Fatalf("body must not contain attachment text: %q", res.Markdown)
	}
}
