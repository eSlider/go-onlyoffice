package onlyoffice

// Text extraction pipeline for the own full-text index (epic #34, F6 #42).
//
// TextIndexer downloads stored documents, extracts text through docpipe
// (pdftotext; OCR for scans) and writes the result to a TextIndex. For PDFs it
// also indexes the text of embedded attachments (pdfdetach), so a scan filed
// as an attachment is searchable too. It is the write side of ESTextIndex and
// never touches the OnlyOffice server's own ES index.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eslider/go-onlyoffice/internal/docpipe"
)

// defaultTextIndexExts are the formats extracted by default. The OnlyOffice
// index already covers docx/xlsx/pptx; F6 adds PDF.
var defaultTextIndexExts = []string{"pdf"}

const (
	defaultTextIndexWorkers = 3
	defaultTextIndexLang    = "deu+eng"
	maxTextIndexErrors      = 20
)

// TextExtractor turns a local file into indexable plain text. The default uses
// docpipe (pdftotext + OCR); tests inject a fake to stay offline.
type TextExtractor interface {
	Extract(path, workDir, lang string, minChars int) (string, error)
}

// docpipeExtractor is the production TextExtractor.
type docpipeExtractor struct{ tools docpipe.Tools }

// Extract renders the file as Markdown, OCRing PDFs/images with a weak text
// layer first and appending the text of embedded PDF attachments
// (docpipe.ToMarkdownWithAttachments).
func (d docpipeExtractor) Extract(path, workDir, lang string, minChars int) (string, error) {
	text, err := d.tools.ToMarkdownWithAttachments(path, workDir, lang, minChars)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// IndexOptions controls a TextIndexer run.
type IndexOptions struct {
	Recursive  bool     // IndexFolder: descend into subfolders
	Extensions []string // empty = defaultTextIndexExts (pdf)
	Limit      int      // max files to index, 0 = all
	Lang       string   // OCR language(s), default deu+eng
	MinChars   int      // OCR threshold, default docpipe.DefaultMinTextChars
	Workers    int      // parallel downloads/extractions, default 3
}

// IndexResult summarises a run.
type IndexResult struct {
	Scanned int
	Indexed int
	Skipped int
	Failed  int
	Errors  []string
}

// TextIndexer wires a FileStore, a TextIndex and an extractor together.
type TextIndexer struct {
	Store     FileStore
	Index     TextIndex
	Extractor TextExtractor // nil = local docpipe tools
	WorkDir   string        // temp dir for downloads/extraction
}

// NewTextIndexer returns a TextIndexer over the given store and index.
func NewTextIndexer(store FileStore, index TextIndex) *TextIndexer {
	return &TextIndexer{Store: store, Index: index}
}

// textIndexEnsurer is implemented by indexes that can be created up front.
type textIndexEnsurer interface {
	Ensure(ctx context.Context) error
}

// Ensure creates the backing index when the TextIndex supports it.
func (ix *TextIndexer) Ensure(ctx context.Context) error {
	if e, ok := ix.Index.(textIndexEnsurer); ok {
		return e.Ensure(ctx)
	}
	return nil
}

// IndexFiles stats the given file ids and indexes them.
func (ix *TextIndexer) IndexFiles(ctx context.Context, ids []string, opts IndexOptions) (IndexResult, error) {
	entries := make([]Entry, 0, len(ids))
	for _, id := range ids {
		e, err := ix.Store.Stat(ctx, id)
		if err != nil {
			return IndexResult{}, fmt.Errorf("onlyoffice: stat %s: %w", id, err)
		}
		entries = append(entries, e)
	}
	return ix.IndexEntries(ctx, entries, opts)
}

// IndexFolder lists a folder and indexes every matching file.
func (ix *TextIndexer) IndexFolder(ctx context.Context, folderID string, opts IndexOptions) (IndexResult, error) {
	entries, err := ix.collect(ctx, folderID, opts.Recursive)
	if err != nil {
		return IndexResult{}, err
	}
	return ix.IndexEntries(ctx, entries, opts)
}

// PlanFolder lists the files IndexFolder would process, without downloading or
// extracting anything.
func (ix *TextIndexer) PlanFolder(ctx context.Context, folderID string, opts IndexOptions) ([]Entry, error) {
	entries, err := ix.collect(ctx, folderID, opts.Recursive)
	if err != nil {
		return nil, err
	}
	return selectEntries(entries, opts), nil
}

// PlanFiles stats the ids and returns those that would be indexed.
func (ix *TextIndexer) PlanFiles(ctx context.Context, ids []string, opts IndexOptions) ([]Entry, error) {
	entries := make([]Entry, 0, len(ids))
	for _, id := range ids {
		e, err := ix.Store.Stat(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("onlyoffice: stat %s: %w", id, err)
		}
		entries = append(entries, e)
	}
	return selectEntries(entries, opts), nil
}

// IndexEntries extracts and indexes the given files (folders are ignored).
func (ix *TextIndexer) IndexEntries(ctx context.Context, entries []Entry, opts IndexOptions) (IndexResult, error) {
	opts = opts.withDefaults()
	var res IndexResult
	work := selectEntries(entries, opts)
	res.Scanned = len(entries)
	res.Skipped = len(entries) - len(work)
	if len(work) == 0 {
		return res, nil
	}

	workers := opts.Workers
	if workers > len(work) {
		workers = len(work)
	}
	if workers < 1 {
		workers = 1
	}

	type outcome struct {
		doc TextDoc
		err error
	}
	jobs := make(chan Entry)
	results := make(chan outcome, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range jobs {
				if err := ctx.Err(); err != nil {
					results <- outcome{err: err}
					continue
				}
				doc, err := ix.indexOne(ctx, e, opts)
				results <- outcome{doc: doc, err: err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, e := range work {
			select {
			case jobs <- e:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var docs []TextDoc
	for r := range results {
		if r.err != nil {
			res.Failed++
			if len(res.Errors) < maxTextIndexErrors {
				res.Errors = append(res.Errors, r.err.Error())
			}
			continue
		}
		docs = append(docs, r.doc)
	}
	if err := ctx.Err(); err != nil {
		return res, err
	}
	if len(docs) > 0 {
		if err := ix.Index.Put(ctx, docs); err != nil {
			return res, fmt.Errorf("onlyoffice: index %d docs: %w", len(docs), err)
		}
		res.Indexed = len(docs)
	}
	return res, nil
}

// indexOne downloads and extracts a single file.
func (ix *TextIndexer) indexOne(ctx context.Context, e Entry, opts IndexOptions) (TextDoc, error) {
	ext := fileExt(e.Title)
	dir := ix.WorkDir
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return TextDoc{}, err
	}
	tmp, err := os.CreateTemp(dir, "ooidx-*."+ext)
	if err != nil {
		return TextDoc{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := ix.Store.Download(ctx, e.ID, tmp); err != nil {
		tmp.Close()
		return TextDoc{}, fmt.Errorf("download %s (%s): %w", e.ID, e.Title, err)
	}
	if err := tmp.Close(); err != nil {
		return TextDoc{}, err
	}
	text, err := ix.extractor().Extract(tmpPath, dir, opts.Lang, opts.MinChars)
	if err != nil {
		return TextDoc{}, fmt.Errorf("extract %s: %w", e.Title, err)
	}
	return TextDoc{ID: e.ID, Title: e.Title, FolderID: e.ParentID, Ext: ext, Content: text}, nil
}

// collect lists files under folderID, breadth-first when recursive.
func (ix *TextIndexer) collect(ctx context.Context, folderID string, recursive bool) ([]Entry, error) {
	var files []Entry
	queue := []string{folderID}
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		id := queue[0]
		queue = queue[1:]
		entries, err := ix.Store.List(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("onlyoffice: list folder %s: %w", id, err)
		}
		for _, e := range entries {
			if e.Kind == Folder {
				if recursive {
					queue = append(queue, e.ID)
				}
				continue
			}
			if e.ParentID == "" {
				e.ParentID = id
			}
			files = append(files, e)
		}
	}
	return files, nil
}

// selectEntries filters files by extension and applies the limit.
func selectEntries(entries []Entry, opts IndexOptions) []Entry {
	allowed := extensionSet(opts.Extensions)
	work := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Kind != File {
			continue
		}
		if opts.Limit > 0 && len(work) >= opts.Limit {
			break
		}
		if !allowed[fileExt(e.Title)] {
			continue
		}
		work = append(work, e)
	}
	return work
}

// extensionSet normalises the extension allow-list (default: pdf).
func extensionSet(exts []string) map[string]bool {
	if len(exts) == 0 {
		exts = defaultTextIndexExts
	}
	set := make(map[string]bool, len(exts))
	for _, e := range normalizeExtensions(exts) {
		set[e] = true
	}
	return set
}

// fileExt returns the lower-case extension without the dot.
func fileExt(title string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(title)), "."))
}

// withDefaults fills zero-valued options.
func (o IndexOptions) withDefaults() IndexOptions {
	if o.Workers <= 0 {
		o.Workers = defaultTextIndexWorkers
	}
	if o.MinChars <= 0 {
		o.MinChars = docpipe.DefaultMinTextChars
	}
	if strings.TrimSpace(o.Lang) == "" {
		o.Lang = defaultTextIndexLang
	}
	return o
}

// extractor returns the configured extractor or the local docpipe default.
func (ix *TextIndexer) extractor() TextExtractor {
	if ix.Extractor != nil {
		return ix.Extractor
	}
	return docpipeExtractor{tools: docpipe.LookPath()}
}
