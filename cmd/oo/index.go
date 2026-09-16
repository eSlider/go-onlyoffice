package main

import (
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(indexCmd())
}

// indexFlags are shared by the `oo index folder` and `oo index files` verbs.
type indexFlags struct {
	recursive bool
	exts      string
	limit     int
	workers   int
	lang      string
	minChars  int
	workDir   string
	backend   string
	dryRun    bool
	asJSON    bool
}

// indexCmd populates the own full-text index (oo_docs_text) that makes PDF and
// scanned content searchable. The OnlyOffice index is left untouched.
func indexCmd() *cobra.Command {
	f := &indexFlags{}
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Populate the own full-text index for PDF/scan content",
		Long: "Index document text that the OnlyOffice Elasticsearch index does not\n" +
			"cover (PDFs and scans) into a separate index (ONLYOFFICE_ES_TEXT_INDEX,\n" +
			"default oo_docs_text). Text is extracted with docpipe (pdftotext, OCR)\n" +
			"and the OnlyOffice server is never modified.\n\n" +
			"Requires ONLYOFFICE_URL/USER/PASS (to download files) and ONLYOFFICE_ES_URL\n" +
			"(to write the index). See docs/elasticsearch.md.",
	}
	cmd.PersistentFlags().BoolVar(&f.recursive, "recursive", false, "folder: descend into subfolders")
	cmd.PersistentFlags().StringVar(&f.exts, "exts", "pdf", "comma-separated extensions to index")
	cmd.PersistentFlags().IntVar(&f.limit, "limit", 0, "maximum number of files to index (0 = all)")
	cmd.PersistentFlags().IntVar(&f.workers, "workers", 3, "parallel downloads/extractions")
	cmd.PersistentFlags().StringVar(&f.lang, "lang", "deu+eng", "OCR language(s)")
	cmd.PersistentFlags().IntVar(&f.minChars, "min-chars", 0, "text-layer threshold below which OCR runs")
	cmd.PersistentFlags().StringVar(&f.workDir, "work-dir", "", "temp dir for downloads (default: system temp)")
	cmd.PersistentFlags().StringVar(&f.backend, "backend", "rest", "file backend: rest|dav")
	cmd.PersistentFlags().BoolVar(&f.dryRun, "dry-run", false, "list what would be indexed, without changes")
	cmd.PersistentFlags().BoolVar(&f.asJSON, "json", false, "shorthand for --output json")

	cmd.AddCommand(indexFolderCmd(f), indexFilesCmd(f))
	return cmd
}

func indexFolderCmd(f *indexFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "folder FOLDER_ID",
		Short: "Index every matching file in a Documents folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIndex(cmd, f, args[0], nil)
		},
	}
}

func indexFilesCmd(f *indexFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "files FILE_ID...",
		Short: "Index specific Documents files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIndex(cmd, f, "", args)
		},
	}
}

func runIndex(cmd *cobra.Command, f *indexFlags, folderID string, ids []string) error {
	if f.asJSON {
		outputFormat = "json"
	}
	c, err := newOO(cmd)
	if err != nil {
		return err
	}
	idx, err := onlyoffice.NewESTextIndex(onlyoffice.ESTextConfigFromEnv())
	if err != nil {
		return err
	}
	ti := onlyoffice.NewTextIndexer(c.FileStore(f.backend), idx)
	ti.WorkDir = f.workDir
	opts := onlyoffice.IndexOptions{
		Recursive:  f.recursive,
		Extensions: splitList(f.exts),
		Limit:      f.limit,
		Lang:       f.lang,
		MinChars:   f.minChars,
		Workers:    f.workers,
	}
	ctx := cmd.Context()

	if f.dryRun {
		var entries []onlyoffice.Entry
		if folderID != "" {
			entries, err = ti.PlanFolder(ctx, folderID, opts)
		} else {
			entries, err = ti.PlanFiles(ctx, ids, opts)
		}
		if err != nil {
			return err
		}
		rows := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, map[string]any{
				"id":     e.ID,
				"title":  e.Title,
				"folder": e.ParentID,
			})
		}
		printTable([]string{"id", "title", "folder"}, rows)
		return nil
	}

	if err := ti.Ensure(ctx); err != nil {
		return err
	}
	var res onlyoffice.IndexResult
	if folderID != "" {
		res, err = ti.IndexFolder(ctx, folderID, opts)
	} else {
		res, err = ti.IndexFiles(ctx, ids, opts)
	}
	if err != nil {
		return err
	}
	printObject(map[string]any{
		"index":   idx.Index(),
		"scanned": res.Scanned,
		"indexed": res.Indexed,
		"skipped": res.Skipped,
		"failed":  res.Failed,
		"errors":  res.Errors,
	})
	return nil
}

// splitList parses a comma-separated flag value, dropping blanks.
func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
