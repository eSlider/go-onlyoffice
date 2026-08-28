package docpipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExt(t *testing.T) {
	if Ext("Foo.PDF") != ".pdf" {
		t.Fatalf("Ext: %q", Ext("Foo.PDF"))
	}
}

func TestSiblingDOCX(t *testing.T) {
	if got := SiblingDOCX("notes.md"); got != "notes.docx" {
		t.Fatalf("got %q", got)
	}
}

func TestWrapMD(t *testing.T) {
	s := wrapMD("a.pdf", " hello ")
	if !strings.HasPrefix(s, "# a.pdf\n") || !strings.Contains(s, "hello") {
		t.Fatalf("wrap: %q", s)
	}
}

func TestNeedsOCR_Image(t *testing.T) {
	tools := LookPath()
	need, err := tools.NeedsOCR("x.jpg", 0)
	if err != nil || !need {
		t.Fatalf("jpg should need OCR: need=%v err=%v", need, err)
	}
}

func TestTxtToMarkdown_FixedWidthUsesCodeBlock(t *testing.T) {
	var lines []string
	for i := 0; i < 12; i++ {
		lines = append(lines, "          column layout line "+strings.Repeat("x", 40))
	}
	in := strings.Join(lines, "\n")
	md := TxtToMarkdown(in)
	if !strings.HasPrefix(md, "```\n") || !strings.Contains(md, "```") {
		t.Fatalf("expected code block: %q", md[:min(80, len(md))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestTxtToMarkdownPreservesLines(t *testing.T) {
	in := "line1\nline2\n\nline4"
	md := TxtToMarkdown(in)
	if !strings.Contains(md, "line1  \n") || !strings.Contains(md, "line2  \n") {
		t.Fatalf("hard breaks missing: %q", md)
	}
	if !strings.Contains(md, "line4  \n") {
		t.Fatalf("last line: %q", md)
	}
}

func TestTXTToDOCXPreservesLines(t *testing.T) {
	tools := LookPath()
	if tools.Pandoc == "" {
		t.Skip("pandoc not installed")
	}
	dir := t.TempDir()
	txt := filepath.Join(dir, "sample.txt")
	docx := filepath.Join(dir, "sample.docx")
	body := "MyBox Auto — resumen\nNº contrato: 123\n\nEstado: Vigente\n"
	if err := os.WriteFile(txt, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tools.TXTToDOCX(txt, docx); err != nil {
		t.Fatal(err)
	}
	mdOut := filepath.Join(dir, "out.md")
	if err := tools.DOCXToMD(docx, mdOut); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(mdOut)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	for _, want := range []string{"MyBox Auto", "Nº contrato", "Estado: Vigente"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %q", want, s)
		}
	}
	if strings.Contains(s, "MyBox Auto — resumen Nº") {
		t.Fatalf("lines collapsed: %q", s)
	}
}

func TestMDDocxRoundTrip(t *testing.T) {
	tools := LookPath()
	if tools.Pandoc == "" {
		t.Skip("pandoc not installed")
	}
	dir := t.TempDir()
	md := filepath.Join(dir, "n.md")
	docx := filepath.Join(dir, "n.docx")
	md2 := filepath.Join(dir, "n2.md")
	if err := os.WriteFile(md, []byte("# Title\n\nHello **world**.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tools.MDToDOCX(md, docx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(docx); err != nil {
		t.Fatal(err)
	}
	if err := tools.DOCXToMD(docx, md2); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(md2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Hello") {
		t.Fatalf("round-trip missing Hello: %s", b)
	}
}

func TestOptimizePDF(t *testing.T) {
	tools := LookPath()
	if tools.Ghostscript == "" || tools.PDFToText == "" {
		t.Skip("ghostscript/pdftotext not installed")
	}
	in := "/tmp/ccgg-original.pdf"
	if _, err := os.Stat(in); err != nil {
		t.Skip("local fixture not present")
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "out.pdf")
	charsIn, err := tools.PDFTextLayerChars(in)
	if err != nil || charsIn < 1000 {
		t.Skip("fixture has no text layer")
	}
	if err := tools.OptimizePDF(in, out); err != nil {
		t.Fatal(err)
	}
	charsOut, err := tools.PDFTextLayerChars(out)
	if err != nil {
		t.Fatal(err)
	}
	if charsOut < charsIn/2 {
		t.Fatalf("text layer lost: in=%d out=%d", charsIn, charsOut)
	}
}

func TestToMarkdown_PlainMD(t *testing.T) {
	tools := LookPath()
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	_ = os.WriteFile(p, []byte("hi"), 0o644)
	res, err := tools.ToMarkdown(p, dir, "eng", 0)
	if err != nil || res.Markdown != "hi" {
		t.Fatalf("got %+v err=%v", res, err)
	}
}
