package docpipe

// Embedded PDF attachments (F6 #42). Digitised invoices often carry the
// original scan as a PDF attachment; the searchable body may hold only a
// summary. pdfdetach (poppler) lists/saves them; each saved attachment is run
// through the normal docpipe extraction (pdftotext/OCR).

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PDFAttachment is one embedded file in a PDF.
type PDFAttachment struct {
	Index int    // 1-based number as `pdfdetach -list` reports it
	Name  string // embedded file name
}

// AttachmentText is the extracted text of one embedded attachment.
type AttachmentText struct {
	Name string
	Text string
}

// parseAttachmentList parses `pdfdetach -list` output. The first line is a
// count ("N embedded files"); every following line is "<index>: <name>".
// Pure, so it is unit-tested.
func parseAttachmentList(out string) []PDFAttachment {
	var atts []PDFAttachment
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colon := strings.Index(line, ":")
		if colon <= 0 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(line[:colon]))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(line[colon+1:])
		if name == "" {
			continue
		}
		atts = append(atts, PDFAttachment{Index: n, Name: name})
	}
	return atts
}

// safeAttachmentName strips directories and leading dots so a hostile
// attachment name cannot escape the extraction directory.
func safeAttachmentName(name string) string {
	name = strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	name = filepath.Base(name)
	name = strings.TrimLeft(name, ".")
	if name == "" || name == "." || name == "/" {
		return ""
	}
	return name
}

// JoinWithAttachments appends attachment text to the document body, each
// section preceded by an "[attachment: <name>]" marker so a search hit shows
// its source. Empty attachments are skipped. Pure, so it is unit-tested.
func JoinWithAttachments(body string, atts []AttachmentText) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(body, "\n"))
	for _, a := range atts {
		text := strings.TrimSpace(a.Text)
		if text == "" {
			continue
		}
		b.WriteString("\n\n[attachment: ")
		b.WriteString(a.Name)
		b.WriteString("]\n\n")
		b.WriteString(text)
	}
	return b.String()
}

// ListAttachments returns the embedded files of a PDF. A PDF without
// attachments yields an empty slice and no error.
func (t Tools) ListAttachments(pdfPath string) ([]PDFAttachment, error) {
	if t.PDFDetach == "" {
		return nil, fmt.Errorf("pdfdetach not found on PATH")
	}
	cmd := exec.Command(t.PDFDetach, "-list", pdfPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pdfdetach -list %s: %w (%s)", filepath.Base(pdfPath), err, strings.TrimSpace(stderr.String()))
	}
	return parseAttachmentList(string(out)), nil
}

// SaveAttachment writes the n-th embedded file (1-based) to outPath.
func (t Tools) SaveAttachment(pdfPath string, index int, outPath string) error {
	if t.PDFDetach == "" {
		return fmt.Errorf("pdfdetach not found on PATH")
	}
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("output path required")
	}
	if err := EnsureDir(outPath); err != nil {
		return err
	}
	cmd := exec.Command(t.PDFDetach, "-save", strconv.Itoa(index), "-o", outPath, pdfPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pdfdetach -save %d: %w (%s)", index, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// ToMarkdownWithAttachments extracts the file as ToMarkdown does, then — for
// PDFs — appends the text of every embedded attachment under an
// "[attachment: <name>]" marker. Attachment failures are non-fatal: the body
// is returned unchanged.
func (t Tools) ToMarkdownWithAttachments(path, workDir, lang string, minChars int) (string, error) {
	res, err := t.ToMarkdown(path, workDir, lang, minChars)
	if err != nil {
		return "", err
	}
	if Ext(path) != ".pdf" {
		return res.Markdown, nil
	}
	atts, err := t.attachmentTexts(path, workDir, lang, minChars)
	if err != nil {
		return res.Markdown, nil
	}
	return JoinWithAttachments(res.Markdown, atts), nil
}

// attachmentTexts saves and extracts every embedded attachment, skipping the
// ones that cannot be read. It returns an error only when the attachment list
// itself cannot be obtained.
func (t Tools) attachmentTexts(pdfPath, workDir, lang string, minChars int) ([]AttachmentText, error) {
	list, err := t.ListAttachments(pdfPath)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	if workDir == "" {
		workDir = os.TempDir()
	}
	dir := filepath.Join(workDir, "att-"+trimExt(filepath.Base(pdfPath)))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	out := make([]AttachmentText, 0, len(list))
	for _, a := range list {
		name := safeAttachmentName(a.Name)
		if name == "" {
			continue
		}
		saved := filepath.Join(dir, fmt.Sprintf("%d-%s", a.Index, name))
		if err := t.SaveAttachment(pdfPath, a.Index, saved); err != nil {
			continue
		}
		text, err := t.attachmentMarkdown(saved, dir, lang, minChars)
		if err != nil {
			continue
		}
		out = append(out, AttachmentText{Name: a.Name, Text: text})
	}
	return out, nil
}

// attachmentMarkdown extracts a saved attachment with the regular pipeline.
// Structured attachments that docpipe does not convert (e-invoice XML,
// CuraSoft JSON, CSV/HTML) fall back to their text content, so the embedded
// original is still searchable. Other unreadable formats return an error and
// the caller skips them.
func (t Tools) attachmentMarkdown(path, workDir, lang string, minChars int) (string, error) {
	if res, err := t.ToMarkdown(path, workDir, lang, minChars); err == nil {
		return res.Markdown, nil
	}
	switch Ext(path) {
	case ".xml", ".html", ".htm":
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return xmlToText(raw), nil
	case ".json", ".csv", ".yaml", ".yml", ".toml", ".txt", ".md", ".markdown":
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	default:
		return textFallback(path)
	}
}

// textFallback reads an attachment of an unknown or missing extension as plain
// text when it looks textual (valid UTF-8, mostly printable runes). Binary
// payloads (images, archives, NUL-padded blobs) are rejected with an error so
// the caller skips them instead of poisoning the index. Classified digitised
// PDFs (Scanner-*.ocr.pdf) carry .yaml/.md attachments; some exporters omit the
// extension, which this covers.
func textFallback(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 1<<20))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("unsupported attachment type %q", Ext(path))
	}
	if !mostlyPrintable(raw) {
		return "", fmt.Errorf("unsupported attachment type %q", Ext(path))
	}
	return string(raw), nil
}

// mostlyPrintable reports whether at least 90% of the runes are printable text
// (newlines, carriage returns and tabs count as text). Pure, so it is tested.
func mostlyPrintable(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	printable, total := 0, 0
	for _, r := range string(b) {
		if r == utf8.RuneError {
			continue
		}
		total++
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			printable++
		}
	}
	return total > 0 && printable*10 >= total*9
}

// xmlToText returns the character data of an XML/HTML document: element text
// values with decoded entities, one per line. Used for invoice XML (EN 16931
// CII / ZUGFeRD) and HTML attachments. Pure, so it is unit-tested.
func xmlToText(raw []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = false
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		cd, ok := tok.(xml.CharData)
		if !ok {
			continue
		}
		s := strings.TrimSpace(string(cd))
		if s == "" {
			continue
		}
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return b.String()
}
