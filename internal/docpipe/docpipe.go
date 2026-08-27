// Package docpipe converts documents for OnlyOffice agent workflows:
// Markdown ↔ DOCX (pandoc) and image/PDF OCR → searchable PDF + Markdown text.
//
// External tools (optional at runtime; helpers skip/error clearly when missing):
//   - pandoc     — md↔docx
//   - ocrmypdf   — OCR into a searchable PDF
//   - pdftotext  — extract text layer
//   - tesseract  — OCR single images when ocrmypdf is unsuitable
package docpipe

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultMinTextChars: below this, a PDF is treated as needing OCR.
const DefaultMinTextChars = 200

// Tools reports which converters are available on PATH.
type Tools struct {
	Pandoc    string
	OCRMyPDF  string
	PDFToText string
	Tesseract string
}

// LookPath resolves converter binaries (empty string if missing).
func LookPath() Tools {
	find := func(names ...string) string {
		for _, n := range names {
			if p, err := exec.LookPath(n); err == nil {
				return p
			}
		}
		return ""
	}
	return Tools{
		Pandoc:    find("pandoc"),
		OCRMyPDF:  find("ocrmypdf"),
		PDFToText: find("pdftotext"),
		Tesseract: find("tesseract"),
	}
}

func (t Tools) requirePandoc() error {
	if t.Pandoc == "" {
		return fmt.Errorf("pandoc not found on PATH (needed for md↔docx)")
	}
	return nil
}

// Ext returns lower-case extension including dot (".pdf").
func Ext(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

// ConvertFile converts between md and docx (and other pandoc formats) via pandoc.
// outExt may be ".md", ".docx", or a full output path.
func (t Tools) ConvertFile(inPath, outPath string) error {
	if err := t.requirePandoc(); err != nil {
		return err
	}
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("output path required")
	}
	cmd := exec.Command(t.Pandoc, inPath, "-o", outPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc %s → %s: %w (%s)", inPath, outPath, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// MDToDOCX writes a DOCX next to or at outPath from a Markdown file.
func (t Tools) MDToDOCX(mdPath, docxPath string) error {
	if Ext(mdPath) != ".md" && Ext(mdPath) != ".markdown" {
		return fmt.Errorf("expected markdown input, got %q", mdPath)
	}
	if docxPath == "" {
		docxPath = strings.TrimSuffix(mdPath, Ext(mdPath)) + ".docx"
	}
	return t.ConvertFile(mdPath, docxPath)
}

// DOCXToMD writes Markdown from a DOCX file.
func (t Tools) DOCXToMD(docxPath, mdPath string) error {
	if Ext(docxPath) != ".docx" {
		return fmt.Errorf("expected .docx input, got %q", docxPath)
	}
	if mdPath == "" {
		mdPath = strings.TrimSuffix(docxPath, Ext(docxPath)) + ".md"
	}
	return t.ConvertFile(docxPath, mdPath)
}

// PDFTextLayerChars returns approximate extracted character count (0 if unavailable).
func (t Tools) PDFTextLayerChars(pdfPath string) (int, error) {
	if t.PDFToText == "" {
		return 0, fmt.Errorf("pdftotext not found on PATH")
	}
	cmd := exec.Command(t.PDFToText, "-layout", pdfPath, "-")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return len(bytes.TrimSpace(out)), nil
}

// NeedsOCR reports whether path likely needs OCR before text extraction.
func (t Tools) NeedsOCR(path string, minChars int) (bool, error) {
	if minChars <= 0 {
		minChars = DefaultMinTextChars
	}
	switch Ext(path) {
	case ".jpg", ".jpeg", ".png", ".tif", ".tiff", ".webp", ".gif", ".bmp":
		return true, nil
	case ".pdf":
		n, err := t.PDFTextLayerChars(path)
		if err != nil {
			// If we cannot measure, prefer OCR.
			return true, nil
		}
		return n < minChars, nil
	default:
		return false, nil
	}
}

// OCRToPDF runs ocrmypdf into outPDF (searchable). Forces OCR when force is true.
func (t Tools) OCRToPDF(inPath, outPDF string, force bool, lang string) error {
	if t.OCRMyPDF == "" {
		return fmt.Errorf("ocrmypdf not found on PATH")
	}
	if outPDF == "" {
		return fmt.Errorf("output PDF path required")
	}
	if lang == "" {
		lang = "eng"
	}
	args := []string{"-l", lang, "--skip-big", "100"}
	if force {
		args = append(args, "--force-ocr")
	} else {
		args = append(args, "--skip-text")
	}
	args = append(args, inPath, outPDF)
	cmd := exec.Command(t.OCRMyPDF, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Retry with force if skip-text refused.
		if !force && strings.Contains(stderr.String(), "PriorOcrFoundError") == false {
			args2 := []string{"-l", lang, "--force-ocr", inPath, outPDF}
			cmd2 := exec.Command(t.OCRMyPDF, args2...)
			var stderr2 bytes.Buffer
			cmd2.Stderr = &stderr2
			if err2 := cmd2.Run(); err2 == nil {
				return nil
			}
		}
		return fmt.Errorf("ocrmypdf: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// ImageToText OCRs a raster image with tesseract (stdout text).
func (t Tools) ImageToText(imgPath, lang string) (string, error) {
	if t.Tesseract == "" {
		return "", fmt.Errorf("tesseract not found on PATH")
	}
	if lang == "" {
		lang = "eng"
	}
	cmd := exec.Command(t.Tesseract, imgPath, "stdout", "-l", lang)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// ExtractPDFText returns layout text from a PDF via pdftotext.
func (t Tools) ExtractPDFText(pdfPath string) (string, error) {
	if t.PDFToText == "" {
		return "", fmt.Errorf("pdftotext not found on PATH")
	}
	cmd := exec.Command(t.PDFToText, "-layout", pdfPath, "-")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Result of ToMarkdown.
type Result struct {
	Markdown   string
	OCRPDFPath string // set when a searchable PDF was produced
	DidOCR     bool
	Source     string
}

// ToMarkdown turns a local file into Markdown text.
// PDFs/images with weak/no text layer are OCR'd to a searchable PDF first (when tools exist).
func (t Tools) ToMarkdown(path string, workDir string, lang string, minChars int) (Result, error) {
	res := Result{Source: path}
	ext := Ext(path)
	switch ext {
	case ".md", ".markdown", ".txt":
		b, err := os.ReadFile(path)
		if err != nil {
			return res, err
		}
		res.Markdown = string(b)
		return res, nil
	case ".docx", ".odt", ".rtf", ".html", ".htm":
		if err := t.requirePandoc(); err != nil {
			return res, err
		}
		tmp := filepath.Join(workDir, "out.md")
		if err := t.ConvertFile(path, tmp); err != nil {
			return res, err
		}
		b, err := os.ReadFile(tmp)
		if err != nil {
			return res, err
		}
		res.Markdown = string(b)
		return res, nil
	case ".pdf":
		need, _ := t.NeedsOCR(path, minChars)
		pdf := path
		if need {
			if workDir == "" {
				workDir = os.TempDir()
			}
			outPDF := filepath.Join(workDir, trimExt(filepath.Base(path))+".ocr.pdf")
			if err := t.OCRToPDF(path, outPDF, true, lang); err != nil {
				return res, err
			}
			res.DidOCR = true
			res.OCRPDFPath = outPDF
			pdf = outPDF
		}
		text, err := t.ExtractPDFText(pdf)
		if err != nil {
			return res, err
		}
		res.Markdown = wrapMD(filepath.Base(path), text)
		return res, nil
	case ".jpg", ".jpeg", ".png", ".tif", ".tiff", ".webp", ".gif", ".bmp":
		if workDir == "" {
			workDir = os.TempDir()
		}
		outPDF := filepath.Join(workDir, trimExt(filepath.Base(path))+".ocr.pdf")
		if t.OCRMyPDF != "" {
			if err := t.OCRToPDF(path, outPDF, true, lang); err == nil {
				res.DidOCR = true
				res.OCRPDFPath = outPDF
				text, err := t.ExtractPDFText(outPDF)
				if err != nil {
					return res, err
				}
				res.Markdown = wrapMD(filepath.Base(path), text)
				return res, nil
			}
		}
		text, err := t.ImageToText(path, lang)
		if err != nil {
			return res, err
		}
		res.DidOCR = true
		res.Markdown = wrapMD(filepath.Base(path), text)
		return res, nil
	default:
		return res, fmt.Errorf("unsupported type %q for markdown extraction", ext)
	}
}

func wrapMD(title, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "# " + title + "\n\n_(empty text layer)_\n"
	}
	return "# " + title + "\n\n" + body + "\n"
}

func trimExt(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// SiblingDOCX returns path with .docx extension replacing the original ext.
func SiblingDOCX(mdPath string) string {
	return strings.TrimSuffix(mdPath, Ext(mdPath)) + ".docx"
}

// EnsureDir creates parent directories for path.
func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
