package onlyoffice

// Canonical file model and the backend-agnostic store interface. REST
// (files.go), WebDAV (files_webdav.go) and future backends (PostgreSQL,
// Elasticsearch) implement FileStore/Searcher so callers stop depending on a
// concrete transport. This file holds only types and pure conversions — no IO.

import (
	"context"
	"io"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Kind distinguishes files from folders in the canonical model.
type Kind int

const (
	File Kind = iota
	Folder
)

// String renders the kind for logs and table output.
func (k Kind) String() string {
	switch k {
	case File:
		return "file"
	case Folder:
		return "folder"
	default:
		return "unknown"
	}
}

// Provider names for the FileStore adapters.
const (
	ProviderREST = "rest"
	ProviderDAV  = "dav"
)

// Entry is the backend-independent representation of a document or folder.
// Fields that a backend cannot supply stay at their zero value.
type Entry struct {
	ID       string
	ParentID string
	Title    string
	Kind     Kind
	Size     int64
	MIME     string
	Created  time.Time
	Modified time.Time
	// Updated is the backend-native timestamp string, when the backend exposes
	// one. It lets list output round-trip the API value; Modified is the
	// parsed form for logic.
	Updated  string
	Version  int
	Provider string

	// Folder-only counters. Zero for files and for backends that do not
	// report them.
	FilesCount   int
	FoldersCount int
}

// FileStore is the operation surface every file backend implements.
type FileStore interface {
	Name() string
	List(ctx context.Context, parentID string) ([]Entry, error)
	Stat(ctx context.Context, id string) (Entry, error)
	CreateFolder(ctx context.Context, parentID, title string) (Entry, error)
	Upload(ctx context.Context, parentID, title string, r io.Reader) (Entry, error)
	Download(ctx context.Context, id string, w io.Writer) (int64, error)
	Move(ctx context.Context, ids []string, parentID string) error
	Copy(ctx context.Context, ids []string, parentID string) error
	Rename(ctx context.Context, id, title string) error
	Delete(ctx context.Context, ids []string) error
}

// SearchQuery narrows a Searcher request. InContent asks the backend to match
// document bodies, not just titles. Substring switches title matching from the
// analyzer's whole-token match to a case-insensitive "*term*" wildcard and ANDs
// every whitespace-separated term (e.g. "rechnung 2025").
type SearchQuery struct {
	Text       string
	InContent  bool
	FolderID   string
	Extensions []string
	Limit      int
	Substring  bool
}

// SearchHit is one Searcher result: the matching entry plus backend-specific
// ranking metadata.
type SearchHit struct {
	Entry
	Score     float64
	Highlight string
	Path      []string
}

// Searcher is the optional content/name search surface. Only some backends
// (for example Elasticsearch) provide it.
type Searcher interface {
	Search(ctx context.Context, q SearchQuery) ([]SearchHit, error)
	Name() string
}

// FileStore returns the adapter for a backend name: ProviderREST (default) or
// ProviderDAV. Unknown or empty names select the REST backend. The composed
// facade (backend selection/fallback) lives on FileClient in file_facade.go.
func (c *Client) FileStore(backend string) FileStore {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case ProviderDAV, "webdav":
		return &davStore{c: c}
	default:
		return &restStore{c: c}
	}
}

// Files returns the composed file facade. The returned *FileClient implements
// FileStore, so callers that used Files() as the plain REST store keep working.
func (c *Client) Files() *FileClient { return c.newFileClient() }

// retryStoreOp runs one store operation under the shared deterministic
// transient-error policy (429/502/503/504).
func retryStoreOp(ctx context.Context, fn func() error) error {
	return DoRetry(ctx, DefaultRetryPolicy(), fn)
}

// FileEntryToEntry converts a Files-module file row to the canonical model.
func FileEntryToEntry(f *FileEntry, provider string) Entry {
	e := Entry{Kind: File, Provider: provider}
	if f == nil {
		return e
	}
	if f.ID != nil {
		e.ID = f.ID.String()
	}
	e.ParentID = FileFolderID(f)
	if f.Title != nil {
		e.Title = *f.Title
	}
	if f.ContentLength != nil {
		e.Size = parseContentLength(*f.ContentLength)
	}
	exst := ""
	if f.FileExst != nil {
		exst = *f.FileExst
	}
	e.MIME = mimeForTitle(e.Title, exst)
	if f.Updated != nil {
		e.Modified = *f.Updated
		e.Updated = f.Updated.Format(time.RFC3339)
	}
	return e
}

// DavFileToEntry converts a WebDAV file row to the canonical model.
func DavFileToEntry(f DavFile, provider string) Entry {
	return Entry{
		ID:       f.ID,
		Title:    f.Title,
		Kind:     File,
		Size:     f.Size,
		MIME:     mimeForTitle(f.Title, ""),
		Modified: f.ModTime(),
		Updated:  f.Updated,
		Provider: provider,
	}
}

// DavFolderToEntry converts a WebDAV folder row to the canonical model.
func DavFolderToEntry(f DavFolder, provider string) Entry {
	return Entry{
		ID:           f.ID,
		ParentID:     f.ParentID,
		Title:        f.Title,
		Kind:         Folder,
		Modified:     f.ModTime(),
		Updated:      f.Updated,
		Provider:     provider,
		FilesCount:   f.FilesCount,
		FoldersCount: f.FoldersCount,
	}
}

// parseContentLength reads the leading integer of an OnlyOffice contentLength
// string (the API sometimes appends a unit, e.g. "12345 b").
func parseContentLength(s string) int64 {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// mimeForTitle derives a MIME type from an explicit extension or the title.
func mimeForTitle(title, exst string) string {
	ext := strings.TrimSpace(exst)
	if ext == "" {
		ext = filepath.Ext(title)
	}
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return mime.TypeByExtension(strings.ToLower(ext))
}
