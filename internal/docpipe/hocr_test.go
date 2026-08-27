package docpipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHOCRToMarkdown_Fixture(t *testing.T) {
	// Minimal hOCR 1.2 snippet
	hocrXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
 <head>
  <title>tesseract</title>
  <meta http-equiv="Content-Type" content="text/html;charset=utf-8"/>
  <meta name="ocr-system" content="tesseract 5"/>
  <meta name="ocr-capabilities" content="ocr_page ocr_carea ocr_par ocr_line ocrx_word"/>
 </head>
 <body>
  <div class="ocr_page" id="page_1" title="image &quot;x.png&quot;; bbox 0 0 100 50; ppageno 0">
   <div class="ocr_carea" id="block_1_1" title="bbox 0 0 100 50">
    <p class="ocr_par" id="par_1_1" lang="eng" title="bbox 0 0 100 50">
     <span class="ocr_line" id="line_1_1" title="bbox 0 0 100 20; baseline 0 0; x_size 20">
      <span class="ocrx_word" id="word_1_1" title="bbox 0 0 40 20; x_wconf 96">Hello</span>
      <span class="ocrx_word" id="word_1_2" title="bbox 45 0 100 20; x_wconf 92">world</span>
     </span>
    </p>
   </div>
  </div>
 </body>
</html>`
	dir := t.TempDir()
	p := filepath.Join(dir, "sample.hocr")
	if err := os.WriteFile(p, []byte(hocrXML), 0o644); err != nil {
		t.Fatal(err)
	}
	md, yml, err := HOCRToMarkdown(p, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Hello") || !strings.Contains(md, "world") {
		t.Fatalf("md=%q", md)
	}
	if !strings.Contains(yml, "Hello") {
		t.Fatalf("yaml missing word: %s", yml)
	}
}
