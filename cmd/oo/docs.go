package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/internal/docpipe"
	"github.com/eslider/go-onlyoffice/internal/xlspipe"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(docsCmd())
}

func docsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Local document pipeline: md↔docx, OCR→PDF, extract Markdown",
		Long: `Agent-friendly conversions (requires pandoc / ocrmypdf / pdftotext on PATH).

OnlyOffice Documents UI is poor for .md/.txt — keep sources in git, store .docx in OO.
Upload Markdown as DOCX:  oo docs put-md PROJECT_ID file.md
Upload plain text:        oo docs put-txt PROJECT_ID file.txt  (preserves line breaks)
Upload/generate XLSX:     oo docs put-xlsx PROJECT_ID [--template cutover-portugal | FILE.xlsx]
Read an OO file as MD:    oo docs as-md FILE_ID
OCR a scan locally:       oo docs ocr scan.pdf --md out.md
Structured OCR (hOCR→MD): oo docs hocr scan.jpg --md out.md --yaml out.yml`,
	}
	cmd.AddCommand(docsConvertCmd())
	cmd.AddCommand(docsOptimizeCmd())
	cmd.AddCommand(docsOCRCmd())
	cmd.AddCommand(docsHOCRCmd())
	cmd.AddCommand(docsAsMDCmd())
	cmd.AddCommand(docsPutMDCmd())
	cmd.AddCommand(docsPutTxtCmd())
	cmd.AddCommand(docsPutXlsxCmd())
	cmd.AddCommand(docsToolsCmd())
	return cmd
}

func docsToolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "Show which converter binaries are on PATH",
		RunE: func(cmd *cobra.Command, args []string) error {
			t := docpipe.LookPath()
			printObject(map[string]any{
				"pandoc":      strOrNil(t.Pandoc),
				"ocrmypdf":    strOrNil(t.OCRMyPDF),
				"pdftotext":   strOrNil(t.PDFToText),
				"tesseract":   strOrNil(t.Tesseract),
				"ghostscript": strOrNil(t.Ghostscript),
			})
			return nil
		},
	}
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func docsConvertCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "convert PATH",
		Short: "Convert a local file with pandoc (md↔docx by default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := args[0]
			t := docpipe.LookPath()
			out := to
			if out == "" {
				switch docpipe.Ext(in) {
				case ".md", ".markdown":
					out = docpipe.SiblingDOCX(in)
				case ".docx":
					out = strings.TrimSuffix(in, docpipe.Ext(in)) + ".md"
				default:
					return fmt.Errorf("--to required for input type %s", docpipe.Ext(in))
				}
			}
			if err := docpipe.EnsureDir(out); err != nil {
				return err
			}
			if err := t.ConvertFile(in, out); err != nil {
				return err
			}
			printObject(map[string]any{"in": in, "out": out})
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "output path (default: sibling .docx or .md)")
	return cmd
}

func docsOptimizeCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "optimize PDF_PATH",
		Short: "Rewrite PDF via Ghostscript (PostScript pdfwrite) for OO-friendly size/text",
		Long: `Use for InDesign/iText PDFs with a good text layer — avoids ocrmypdf invisible
text overlays that break OnlyOffice DocEditor. Skips OCR; rewrites via gs pdfwrite.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := args[0]
			t := docpipe.LookPath()
			if out == "" {
				base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
				out = filepath.Join(filepath.Dir(in), base+".optimized.pdf")
			}
			if err := docpipe.EnsureDir(out); err != nil {
				return err
			}
			chars, _ := t.PDFTextLayerChars(in)
			if err := t.OptimizePDF(in, out); err != nil {
				return err
			}
			outChars, _ := t.PDFTextLayerChars(out)
			printObject(map[string]any{
				"in":              in,
				"pdf":             out,
				"text_chars_in":   chars,
				"text_chars_out":  outChars,
				"note":            "native text layer preserved; no OCR overlay",
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output PDF (default: <name>.optimized.pdf)")
	return cmd
}

func docsOCRCmd() *cobra.Command {
	var out, mdOut, lang string
	var force bool
	var writeMD bool
	cmd := &cobra.Command{
		Use:   "ocr PATH",
		Short: "OCR image/PDF → searchable PDF (and optional Markdown)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := args[0]
			t := docpipe.LookPath()
			if out == "" {
				base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
				out = filepath.Join(filepath.Dir(in), base+".ocr.pdf")
			}
			if err := docpipe.EnsureDir(out); err != nil {
				return err
			}
			if err := t.OCRToPDF(in, out, force, lang); err != nil {
				return err
			}
			res := map[string]any{"in": in, "pdf": out}
			if writeMD || mdOut != "" {
				if mdOut == "" {
					mdOut = strings.TrimSuffix(out, filepath.Ext(out)) + ".md"
				}
				text, err := t.ExtractPDFText(out)
				if err != nil {
					return err
				}
				body := "# " + filepath.Base(in) + "\n\n" + strings.TrimSpace(text) + "\n"
				if err := os.WriteFile(mdOut, []byte(body), 0o644); err != nil {
					return err
				}
				res["md"] = mdOut
			}
			printObject(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output searchable PDF (default: <name>.ocr.pdf)")
	cmd.Flags().StringVar(&mdOut, "md", "", "write Markdown extraction to this path")
	cmd.Flags().BoolVar(&writeMD, "markdown", false, "also write sibling .md next to OCR PDF")
	cmd.Flags().StringVar(&lang, "lang", "eng", "OCR language(s) for tesseract/ocrmypdf")
	cmd.Flags().BoolVar(&force, "force", false, "force OCR even if a text layer exists (default: skip when pdftotext finds enough text)")
	return cmd
}

func docsHOCRCmd() *cobra.Command {
	var mdOut, yamlOut, hocrOut, lang string
	var dpi int
	var minConf float32
	cmd := &cobra.Command{
		Use:   "hocr PATH",
		Short: "Tesseract hOCR → structured Markdown/YAML (via go-hocr)",
		Long: `Runs tesseract with hOCR output, parses with go-hocr, writes Markdown
(and optional YAML). Better reading order / confidence than plain pdftotext.

For Spanish scans: --lang spa or spa+eng (needs tesseract-ocr-spa / TESSDATA_PREFIX).
Phone photos: --dpi 200..300.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := args[0]
			t := docpipe.LookPath()
			dir, err := os.MkdirTemp("", "oo-docs-hocr-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)

			res, err := t.ToHOCRMarkdown(in, dir, lang, dpi, minConf)
			if err != nil {
				return err
			}
			if mdOut == "" {
				base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
				mdOut = filepath.Join(filepath.Dir(in), base+".hocr.md")
			}
			if err := docpipe.EnsureDir(mdOut); err != nil {
				return err
			}
			if err := os.WriteFile(mdOut, []byte(res.Markdown), 0o644); err != nil {
				return err
			}
			obj := map[string]any{
				"in":      in,
				"md":      mdOut,
				"did_ocr": res.DidOCR,
			}
			if hocrOut != "" {
				if err := docpipe.EnsureDir(hocrOut); err != nil {
					return err
				}
				b, err := os.ReadFile(res.HOCRPath)
				if err != nil {
					return err
				}
				if err := os.WriteFile(hocrOut, b, 0o644); err != nil {
					return err
				}
				obj["hocr"] = hocrOut
			} else {
				obj["hocr_tmp"] = res.HOCRPath
			}
			if yamlOut != "" {
				if err := docpipe.EnsureDir(yamlOut); err != nil {
					return err
				}
				if err := os.WriteFile(yamlOut, []byte(res.YAML), 0o644); err != nil {
					return err
				}
				obj["yaml"] = yamlOut
			}
			printObject(obj)
			return nil
		},
	}
	cmd.Flags().StringVar(&mdOut, "md", "", "Markdown output (default: <name>.hocr.md)")
	cmd.Flags().StringVar(&yamlOut, "yaml", "", "also write structured YAML from go-hocr")
	cmd.Flags().StringVar(&hocrOut, "hocr", "", "also keep raw .hocr file at this path")
	cmd.Flags().StringVar(&lang, "lang", "eng", "tesseract language(s), e.g. spa+eng")
	cmd.Flags().IntVar(&dpi, "dpi", 220, "hint DPI for phone photos / scans (0 = tesseract default)")
	cmd.Flags().Float32Var(&minConf, "min-conf", 0, "drop words with OCR confidence below this (0 = keep all)")
	return cmd
}

func docsAsMDCmd() *cobra.Command {
	var to, lang string
	var minChars int
	var uploadOCR bool
	var useHOCR bool
	var dpi int
	var minConf float32
	cmd := &cobra.Command{
		Use:   "as-md FILE_ID",
		Short: "Download an OO Documents file and emit Markdown (OCR PDF/image if needed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			meta, err := c.GetFile(ctx, args[0])
			if err != nil {
				return err
			}
			title := onlyoffice.FileEntryTitle(meta)
			dir, err := os.MkdirTemp("", "oo-docs-as-md-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)

			local := filepath.Join(dir, onlyoffice.SafeLocalFileName(title))
			f, err := os.Create(local)
			if err != nil {
				return err
			}
			if _, err := c.DownloadFile(ctx, args[0], f); err != nil {
				_ = f.Close()
				return err
			}
			_ = f.Close()

			tools := docpipe.LookPath()
			var res docpipe.Result
			if useHOCR {
				hr, err := tools.ToHOCRMarkdown(local, dir, lang, dpi, minConf)
				if err != nil {
					return err
				}
				res = docpipe.Result{Markdown: hr.Markdown, DidOCR: hr.DidOCR, Source: hr.Source}
			} else {
				res, err = tools.ToMarkdown(local, dir, lang, minChars)
				if err != nil {
					return err
				}
			}
			outPath := to
			if outPath == "" {
				base := strings.TrimSuffix(onlyoffice.SafeLocalFileName(title), filepath.Ext(onlyoffice.SafeLocalFileName(title)))
				if base == "" || base == "download" {
					base = "file-" + args[0]
				}
				outPath = base + ".md"
			}
			if err := docpipe.EnsureDir(outPath); err != nil {
				return err
			}
			if err := os.WriteFile(outPath, []byte(res.Markdown), 0o644); err != nil {
				return err
			}

			obj := map[string]any{
				"file_id": args[0],
				"title":   title,
				"md":      outPath,
				"did_ocr": res.DidOCR,
			}
			if res.OCRPDFPath != "" && uploadOCR {
				folderID := onlyoffice.FileFolderID(meta)
				upName := strings.TrimSuffix(onlyoffice.SafeLocalFileName(title), filepath.Ext(onlyoffice.SafeLocalFileName(title))) + ".ocr.pdf"
				tmpUp := filepath.Join(dir, upName)
				data, err := os.ReadFile(res.OCRPDFPath)
				if err != nil {
					return err
				}
				if err := os.WriteFile(tmpUp, data, 0o644); err != nil {
					return err
				}
				if folderID == "" {
					obj["ocr_pdf_local"] = res.OCRPDFPath
					obj["note"] = "file has no folderId; OCR PDF left local — pass after moving into a folder"
				} else {
					ent, err := c.UploadToFolder(ctx, folderID, tmpUp)
					if err != nil {
						return err
					}
					obj["ocr_pdf_file_id"] = fileIDStr(ent)
					obj["ocr_pdf_title"] = onlyoffice.FileEntryTitle(ent)
				}
			} else if res.OCRPDFPath != "" {
				// Keep OCR PDF outside temp by copying beside md if requested via env-less default:
				kept := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".ocr.pdf"
				if b, err := os.ReadFile(res.OCRPDFPath); err == nil {
					_ = os.WriteFile(kept, b, 0o644)
					obj["ocr_pdf_local"] = kept
				} else {
					obj["ocr_pdf_local"] = res.OCRPDFPath
				}
			}
			printObject(obj)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "write Markdown to this path (default: ./<title>.md)")
	cmd.Flags().StringVar(&lang, "lang", "eng", "OCR language")
	cmd.Flags().IntVar(&minChars, "min-chars", docpipe.DefaultMinTextChars, "OCR PDF if text layer shorter than this")
	cmd.Flags().BoolVar(&uploadOCR, "upload-ocr", false, "upload searchable OCR PDF back into the same OO folder")
	cmd.Flags().BoolVar(&useHOCR, "hocr", false, "use tesseract hOCR + go-hocr instead of ocrmypdf/pdftotext")
	cmd.Flags().IntVar(&dpi, "dpi", 220, "DPI hint when --hocr (phone photos)")
	cmd.Flags().Float32Var(&minConf, "min-conf", 0, "drop low-confidence words when --hocr")
	return cmd
}

func docsPutMDCmd() *cobra.Command {
	var folderID string
	var keepLocalDOCX string
	var replace bool
	cmd := &cobra.Command{
		Use:   "put-md PROJECT_ID MARKDOWN_PATH",
		Short: "Convert Markdown→DOCX and upload DOCX into a project (OO-friendly)",
		Long:  `Agents edit .md locally; this uploads .docx so OnlyOffice can open/version it.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, mdPath := args[0], args[1]
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			tools := docpipe.LookPath()
			dir, err := os.MkdirTemp("", "oo-docs-put-md-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			docxName := strings.TrimSuffix(filepath.Base(mdPath), filepath.Ext(mdPath)) + ".docx"
			docxPath := filepath.Join(dir, docxName)
			if err := tools.MDToDOCX(mdPath, docxPath); err != nil {
				return err
			}
			if keepLocalDOCX != "" {
				if err := docpipe.EnsureDir(keepLocalDOCX); err != nil {
					return err
				}
				b, err := os.ReadFile(docxPath)
				if err != nil {
					return err
				}
				if err := os.WriteFile(keepLocalDOCX, b, 0o644); err != nil {
					return err
				}
			}
			ctx := cmd.Context()
			var ent *onlyoffice.FileEntry
			var deleted []int
			if folderID != "" {
				if replace {
					ent, deleted, err = c.UploadToFolderReplacing(ctx, folderID, docxPath)
				} else {
					ent, err = c.UploadToFolder(ctx, folderID, docxPath)
				}
				if err != nil {
					return err
				}
				obj := map[string]any{
					"project_id": pid,
					"folder_id":  folderID,
					"md":         mdPath,
					"uploaded":   fileEntryToMap(ent),
				}
				if len(deleted) > 0 {
					obj["replaced_file_ids"] = deleted
				}
				printObject(obj)
				return nil
			}
			ent, err = c.UploadProjectFile(ctx, pid, docxPath)
			if err != nil {
				return err
			}
			printObject(map[string]any{
				"project_id": pid,
				"md":         mdPath,
				"uploaded":   fileEntryToMap(ent),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&folderID, "folder", "", "Documents folder id (default: project root)")
	cmd.Flags().StringVar(&keepLocalDOCX, "keep-docx", "", "also write the generated DOCX to this local path")
	cmd.Flags().BoolVar(&replace, "replace", true, "delete same-stem files in folder before upload (put-md upsert)")
	return cmd
}

func docsPutTxtCmd() *cobra.Command {
	var folderID string
	var keepLocalDOCX string
	var replace bool
	cmd := &cobra.Command{
		Use:   "put-txt PROJECT_ID TEXT_PATH",
		Short: "Convert plain text→DOCX (preserve line breaks) and upload into a project",
		Long:  `OnlyOffice cannot render .txt well. This keeps each source line on its own DOCX line.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, txtPath := args[0], args[1]
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			tools := docpipe.LookPath()
			dir, err := os.MkdirTemp("", "oo-docs-put-txt-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)
			docxName := strings.TrimSuffix(filepath.Base(txtPath), filepath.Ext(txtPath)) + ".docx"
			docxPath := filepath.Join(dir, docxName)
			if err := tools.TXTToDOCX(txtPath, docxPath); err != nil {
				return err
			}
			if keepLocalDOCX != "" {
				if err := docpipe.EnsureDir(keepLocalDOCX); err != nil {
					return err
				}
				b, err := os.ReadFile(docxPath)
				if err != nil {
					return err
				}
				if err := os.WriteFile(keepLocalDOCX, b, 0o644); err != nil {
					return err
				}
			}
			ctx := cmd.Context()
			var ent *onlyoffice.FileEntry
			var deleted []int
			if folderID != "" {
				if replace {
					ent, deleted, err = c.UploadToFolderReplacing(ctx, folderID, docxPath)
				} else {
					ent, err = c.UploadToFolder(ctx, folderID, docxPath)
				}
				if err != nil {
					return err
				}
				obj := map[string]any{
					"project_id": pid,
					"txt":        txtPath,
					"folder_id":  folderID,
					"uploaded":   fileEntryToMap(ent),
				}
				if len(deleted) > 0 {
					obj["replaced_file_ids"] = deleted
				}
				printObject(obj)
				return nil
			}
			ent, err = c.UploadProjectFile(ctx, pid, docxPath)
			if err != nil {
				return err
			}
			printObject(map[string]any{
				"project_id": pid,
				"txt":        txtPath,
				"uploaded":   fileEntryToMap(ent),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&folderID, "folder", "", "Documents folder id (default: project root)")
	cmd.Flags().StringVar(&keepLocalDOCX, "keep-docx", "", "also write the generated DOCX to this local path")
	cmd.Flags().BoolVar(&replace, "replace", true, "delete same-stem files in folder before upload (put-txt upsert)")
	return cmd
}

func docsPutXlsxCmd() *cobra.Command {
	var folderID, template, title, keepLocal string
	var replace bool
	cmd := &cobra.Command{
		Use:   "put-xlsx PROJECT_ID [LOCAL_XLSX]",
		Short: "Upload or generate an XLSX workbook into a project (excelize templates with formulas)",
		Long: `Spreadsheets live in OnlyOffice — not in git. Generate multi-sheet workbooks with
formulas (SUM/AVG, cross-sheet refs, named inputs) via --template, or upload an existing .xlsx.

  oo docs put-xlsx 218 --template cutover-portugal
  oo docs put-xlsx 218 --template cutover-portugal --title 2026-08-28-cutover-budget.xlsx
  oo docs put-xlsx 218 ./my.xlsx`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := args[0]
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			dir, err := os.MkdirTemp("", "oo-docs-put-xlsx-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(dir)

			var xlsxPath string
			var srcLabel string
			switch {
			case template != "":
				wb, err := xlspipe.BuildTemplate(template)
				if err != nil {
					return err
				}
				name := title
				if name == "" {
					name = "cutover-budget.xlsx"
					if template == xlspipe.TemplateCutoverPortugal {
						name = "2026-08-28-cutover-budget.xlsx"
					}
				}
				if !strings.HasSuffix(strings.ToLower(name), ".xlsx") {
					name += ".xlsx"
				}
				xlsxPath = filepath.Join(dir, name)
				if err := xlspipe.Save(wb, xlsxPath); err != nil {
					wb.Close()
					return err
				}
				wb.Close()
				srcLabel = "template:" + template
			case len(args) == 2:
				xlsxPath = args[1]
				if docpipe.Ext(xlsxPath) != ".xlsx" {
					return fmt.Errorf("expected .xlsx, got %s", docpipe.Ext(xlsxPath))
				}
				srcLabel = xlsxPath
			default:
				return fmt.Errorf("pass LOCAL_XLSX or --template")
			}

			if keepLocal != "" {
				b, err := os.ReadFile(xlsxPath)
				if err != nil {
					return err
				}
				if err := docpipe.EnsureDir(keepLocal); err != nil {
					return err
				}
				if err := os.WriteFile(keepLocal, b, 0o644); err != nil {
					return err
				}
			}

			ctx := cmd.Context()
			var ent *onlyoffice.FileEntry
			var deleted []int
			if folderID != "" {
				if replace {
					ent, deleted, err = c.UploadToFolderReplacing(ctx, folderID, xlsxPath)
				} else {
					ent, err = c.UploadToFolder(ctx, folderID, xlsxPath)
				}
			} else {
				if replace {
					ent, deleted, err = c.UploadProjectFileReplacing(ctx, pid, xlsxPath)
				} else {
					ent, err = c.UploadProjectFile(ctx, pid, xlsxPath)
				}
			}
			if err != nil {
				return err
			}
			obj := map[string]any{
				"project_id": pid,
				"source":     srcLabel,
				"uploaded":   fileEntryToMap(ent),
			}
			if folderID != "" {
				obj["folder_id"] = folderID
			}
			if len(deleted) > 0 {
				obj["replaced_file_ids"] = deleted
			}
			printObject(obj)
			return nil
		},
	}
	cmd.Flags().StringVar(&folderID, "folder", "", "Documents folder id (default: project root)")
	cmd.Flags().StringVar(&template, "template", "", "built-in workbook template (cutover-portugal)")
	cmd.Flags().StringVar(&title, "title", "", "upload file name when using --template")
	cmd.Flags().StringVar(&keepLocal, "keep-xlsx", "", "also write generated/uploaded bytes to this local path")
	cmd.Flags().BoolVar(&replace, "replace", true, "delete same-stem files before upload")
	return cmd
}
