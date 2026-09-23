package onlyoffice

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFileEntryToEntry(t *testing.T) {
	id := json.Number("42")
	title := "invoice.pdf"
	exst := ".pdf"
	size := "12345"
	parent := json.Number("7")
	updated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	f := &FileEntry{
		ID:            &id,
		Title:         &title,
		FileExst:      &exst,
		ContentLength: &size,
		FolderID:      &parent,
		Updated:       &updated,
	}

	e := FileEntryToEntry(f, ProviderREST)
	if e.ID != "42" {
		t.Errorf("ID = %q, want 42", e.ID)
	}
	if e.ParentID != "7" {
		t.Errorf("ParentID = %q, want 7", e.ParentID)
	}
	if e.Title != title {
		t.Errorf("Title = %q, want %q", e.Title, title)
	}
	if e.Kind != File {
		t.Errorf("Kind = %v, want file", e.Kind)
	}
	if e.Size != 12345 {
		t.Errorf("Size = %d, want 12345", e.Size)
	}
	if e.MIME != "application/pdf" {
		t.Errorf("MIME = %q, want application/pdf", e.MIME)
	}
	if !e.Modified.Equal(updated) {
		t.Errorf("Modified = %v, want %v", e.Modified, updated)
	}
	if e.Provider != ProviderREST {
		t.Errorf("Provider = %q, want %q", e.Provider, ProviderREST)
	}
}

func TestFileEntryToEntryNil(t *testing.T) {
	e := FileEntryToEntry(nil, ProviderDAV)
	if e.Kind != File {
		t.Errorf("Kind = %v, want file", e.Kind)
	}
	if e.ID != "" || e.Title != "" {
		t.Errorf("nil entry should be empty: %+v", e)
	}
	if e.Provider != ProviderDAV {
		t.Errorf("Provider = %q, want %q", e.Provider, ProviderDAV)
	}
}

func TestFileEntryToEntrySizeFormats(t *testing.T) {
	cases := map[string]int64{
		"12345":   12345,
		"12345 b": 12345,
		"0":       0,
		"":        0,
		"notanum": 0,
	}
	for in, want := range cases {
		got := parseContentLength(in)
		if got != want {
			t.Errorf("parseContentLength(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestDavFileToEntry(t *testing.T) {
	f := DavFile{
		ID:      "9",
		Title:   "note.txt",
		Size:    10,
		Updated: "2026-01-02T03:04:05.0000000+01:00",
	}
	e := DavFileToEntry(f, ProviderDAV)
	if e.ID != "9" || e.Title != "note.txt" {
		t.Errorf("identity mismatch: %+v", e)
	}
	if e.Kind != File {
		t.Errorf("Kind = %v, want file", e.Kind)
	}
	if e.Size != 10 {
		t.Errorf("Size = %d, want 10", e.Size)
	}
	if !strings.HasPrefix(e.MIME, "text/plain") {
		t.Errorf("MIME = %q, want text/plain*", e.MIME)
	}
	if e.Modified.IsZero() {
		t.Error("Modified not parsed")
	}
	if e.Provider != ProviderDAV {
		t.Errorf("Provider = %q, want %q", e.Provider, ProviderDAV)
	}
}

func TestDavFolderToEntry(t *testing.T) {
	f := DavFolder{
		ID:       "5",
		Title:    "inbox",
		ParentID: "1",
		Updated:  "2026-01-02T03:04:05.0000000+01:00",
	}
	e := DavFolderToEntry(f, ProviderDAV)
	if e.ID != "5" || e.Title != "inbox" || e.ParentID != "1" {
		t.Errorf("identity mismatch: %+v", e)
	}
	if e.Kind != Folder {
		t.Errorf("Kind = %v, want folder", e.Kind)
	}
	if e.MIME != "" {
		t.Errorf("folder MIME = %q, want empty", e.MIME)
	}
	if e.Modified.IsZero() {
		t.Error("Modified not parsed")
	}
}

func TestEntriesFromFolderMap(t *testing.T) {
	m := map[string]any{
		"files": []any{
			map[string]any{"id": float64(42), "title": "a.pdf", "pureContentLength": float64(7)},
		},
		"folders": []any{
			map[string]any{"id": float64(7), "title": "sub", "parentId": float64(1)},
		},
	}
	entries, err := entriesFromFolderMap(m, ProviderREST)
	if err != nil {
		t.Fatalf("entriesFromFolderMap: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}
	byID := map[string]Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	if got := byID["42"]; got.Kind != File || got.Size != 7 || got.Title != "a.pdf" {
		t.Errorf("file entry = %+v", got)
	}
	if got := byID["7"]; got.Kind != Folder || got.ParentID != "1" || got.Title != "sub" {
		t.Errorf("folder entry = %+v", got)
	}
}

func TestEntriesFromFolderMapNil(t *testing.T) {
	entries, err := entriesFromFolderMap(nil, ProviderREST)
	if err != nil || entries != nil {
		t.Fatalf("got %v, %v; want nil, nil", entries, err)
	}
}

func TestKindString(t *testing.T) {
	if File.String() != "file" || Folder.String() != "folder" {
		t.Errorf("kind strings: %q %q", File.String(), Folder.String())
	}
	if Kind(9).String() != "unknown" {
		t.Errorf("unknown kind = %q", Kind(9).String())
	}
}

func TestClientFileStoreSelection(t *testing.T) {
	c := NewClient(Credentials{})
	if got := c.FileStore(ProviderDAV).Name(); got != ProviderDAV {
		t.Errorf("FileStore(dav).Name() = %q", got)
	}
	if got := c.FileStore("webdav").Name(); got != ProviderDAV {
		t.Errorf("FileStore(webdav).Name() = %q", got)
	}
	if got := c.FileStore(ProviderREST).Name(); got != ProviderREST {
		t.Errorf("FileStore(rest).Name() = %q", got)
	}
	if got := c.FileStore("").Name(); got != ProviderREST {
		t.Errorf("FileStore(\"\").Name() = %q", got)
	}
	if got := c.Files().Name(); got != ProviderREST {
		t.Errorf("Files().Name() = %q", got)
	}
}
