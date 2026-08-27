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
