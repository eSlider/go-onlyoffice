package docpipe

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	hocr "github.com/eslider/go-hocr"
)

// HOCRResult is structured OCR output for agents.
type HOCRResult struct {
	HOCRPath string
	Markdown string
	YAML     string
	DidOCR   bool
	Source   string
}

// ImageToHOCR runs tesseract hOCR into outBase+".hocr" (tesseract adds the extension).
// outBase must not include ".hocr". dpi 0 uses tesseract default; phone photos often need 200–300.
func (t Tools) ImageToHOCR(imgPath, outBase, lang string, dpi int) (string, error) {
	if t.Tesseract == "" {
		return "", fmt.Errorf("tesseract not found on PATH")
	}
	if lang == "" {
		lang = "eng"
	}
	if outBase == "" {
		return "", fmt.Errorf("hOCR output base path required")
	}
	args := []string{imgPath, outBase, "-l", lang}
	if dpi > 0 {
		args = append(args, "--dpi", fmt.Sprintf("%d", dpi))
	}
	args = append(args, resolveHOCRConfig())
	cmd := exec.Command(t.Tesseract, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract hocr: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	out := outBase + ".hocr"
	if _, err := os.Stat(out); err != nil {
		// Some builds write .html
		alt := outBase + ".html"
		if _, err2 := os.Stat(alt); err2 == nil {
			return alt, nil
		}
		return "", fmt.Errorf("tesseract hocr: missing output %s (%s)", out, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// resolveHOCRConfig returns a tesseract config path that works when TESSDATA_PREFIX
// points at a custom traineddata dir without relative "hocr" configs.
func resolveHOCRConfig() string {
	var candidates []string
	if p := strings.TrimSpace(os.Getenv("TESSDATA_PREFIX")); p != "" {
		candidates = append(candidates,
			filepath.Join(p, "configs", "hocr"),
			filepath.Join(p, "tessdata", "configs", "hocr"),
		)
	}
	candidates = append(candidates,
		"/usr/share/tesseract-ocr/5/tessdata/configs/hocr",
		"/usr/share/tesseract-ocr/4.00/tessdata/configs/hocr",
		"/usr/share/tessdata/configs/hocr",
	)
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "hocr"
}

// HOCRToMarkdown parses an hOCR file via go-hocr and returns Markdown (+ optional YAML).
// Words with confidence in (0, minConf) are dropped; minConf 0 keeps all.
func HOCRToMarkdown(hocrPath string, minConf float32) (md string, yml string, err error) {
	doc, err := hocr.ReadFile(hocrPath)
	if err != nil {
		return "", "", fmt.Errorf("go-hocr: %w", err)
	}
	md = doc.ToMarkdown(minConf)
	yml, err = doc.ToYaml()
	if err != nil {
		return md, "", err
	}
	return md, yml, nil
}

// ToHOCRMarkdown OCRs an image (or rasterizes first PDF page) to hOCR → Markdown/YAML.
func (t Tools) ToHOCRMarkdown(path, workDir, lang string, dpi int, minConf float32) (HOCRResult, error) {
	res := HOCRResult{Source: path}
	if workDir == "" {
		workDir = os.TempDir()
	}
	ext := Ext(path)
	img := path
	switch ext {
	case ".jpg", ".jpeg", ".png", ".tif", ".tiff", ".webp", ".gif", ".bmp":
		// ok
	case ".pdf":
		raster, err := t.pdfFirstPagePNG(path, workDir)
		if err != nil {
			return res, err
		}
		img = raster
		if dpi == 0 {
			dpi = 300
		}
	default:
		return res, fmt.Errorf("hOCR path expects image or PDF, got %q", ext)
	}
	base := filepath.Join(workDir, trimExt(filepath.Base(path))+".hocr-out")
	hocrPath, err := t.ImageToHOCR(img, base, lang, dpi)
	if err != nil {
		return res, err
	}
	res.DidOCR = true
	res.HOCRPath = hocrPath
	md, yml, err := HOCRToMarkdown(hocrPath, minConf)
	if err != nil {
		return res, err
	}
	res.Markdown = wrapMD(filepath.Base(path), strings.TrimSpace(md))
	res.YAML = yml
	return res, nil
}

// pdfFirstPagePNG uses pdftoppm when available.
func (t Tools) pdfFirstPagePNG(pdfPath, workDir string) (string, error) {
	pdftoppm, err := exec.LookPath("pdftoppm")
	if err != nil {
		return "", fmt.Errorf("pdftoppm not found (needed to rasterize PDF for hOCR)")
	}
	outBase := filepath.Join(workDir, trimExt(filepath.Base(pdfPath))+".page")
	cmd := exec.Command(pdftoppm, "-png", "-f", "1", "-singlefile", "-r", "200", pdfPath, outBase)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftoppm: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	png := outBase + ".png"
	if _, err := os.Stat(png); err != nil {
		return "", fmt.Errorf("pdftoppm: missing %s", png)
	}
	return png, nil
}
