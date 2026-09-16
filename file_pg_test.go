package onlyoffice

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRebind(t *testing.T) {
	mysqlQuery := "SELECT id FROM files_file WHERE folder_id = ? AND title = ? LIMIT ?"
	if got := rebind(mysqlQuery, ProviderMySQL); got != mysqlQuery {
		t.Errorf("mysql query changed: %q", got)
	}
	want := "SELECT id FROM files_file WHERE folder_id = $1 AND title = $2 LIMIT $3"
	if got := rebind(mysqlQuery, ProviderPG); got != want {
		t.Errorf("rebind = %q, want %q", got, want)
	}
}

func TestCSPObjectKey(t *testing.T) {
	cases := []struct {
		tenant  int64
		fileID  int64
		version int
		ext     string
		want    string
	}{
		{1, 2, 1, ".docx", "00/00/01/files/folder_1000/file_2/v1/content.docx"},
		{1, 999, 1, ".pdf", "00/00/01/files/folder_1000/file_999/v1/content.pdf"},
		{1, 1000, 1, ".xlsx", "00/00/01/files/folder_2000/file_1000/v1/content.xlsx"},
		{1, 3727, 1, ".pdf", "00/00/01/files/folder_4000/file_3727/v1/content.pdf"},
		{1, 22484, 1, ".PDF", "00/00/01/files/folder_23000/file_22484/v1/content.pdf"},
		{1, 4, 6, "xlsx", "00/00/01/files/folder_1000/file_4/v6/content.xlsx"},
		{0, 7, 0, "", "00/00/01/files/folder_1000/file_7/v1/content.bin"},
		{2, 11, 3, ".doc", "00/00/02/files/folder_1000/file_11/v3/content.doc"},
	}
	for _, tc := range cases {
		if got := csObjectKey(tc.tenant, tc.fileID, tc.version, tc.ext); got != tc.want {
			t.Errorf("csObjectKey(%d,%d,%d,%q) = %q, want %q", tc.tenant, tc.fileID, tc.version, tc.ext, got, tc.want)
		}
	}
}

func TestPGDriverDetection(t *testing.T) {
	cases := []struct {
		dsn, explicit, want string
	}{
		{"postgres://u:p@h:5432/onlyoffice", "", ProviderPG},
		{"postgresql://u:p@h/db", "", ProviderPG},
		{"host=h user=u password=p dbname=onlyoffice sslmode=disable", "", ProviderPG},
		{"root:secret@tcp(127.0.0.1:3306)/onlyoffice?parseTime=true", "", ProviderMySQL},
		{"mysql://root:secret@127.0.0.1:3306/onlyoffice", "", ProviderMySQL},
		{"root:secret@tcp(h:3306)/db", "postgres", ProviderPG},
		{"postgres://u:p@h/db", "mysql", ProviderMySQL},
	}
	for _, tc := range cases {
		if got := pgDriver(tc.dsn, tc.explicit); got != tc.want {
			t.Errorf("pgDriver(%q, %q) = %q, want %q", tc.dsn, tc.explicit, got, tc.want)
		}
	}
}

func TestNormalizeSQLDSNMySQL(t *testing.T) {
	got, err := normalizeSQLDSN(ProviderMySQL, "mysql://root:secret@127.0.0.1:3306/onlyoffice")
	if err != nil {
		t.Fatalf("normalizeSQLDSN: %v", err)
	}
	want := "root:secret@tcp(127.0.0.1:3306)/onlyoffice?parseTime=true"
	if got != want {
		t.Errorf("normalize = %q, want %q", got, want)
	}

	// A driver DSN keeps parseTime and gains it when missing.
	got, err = normalizeSQLDSN(ProviderMySQL, "root:secret@tcp(127.0.0.1:3306)/onlyoffice")
	if err != nil {
		t.Fatalf("normalizeSQLDSN: %v", err)
	}
	if got != want {
		t.Errorf("normalize = %q, want %q", got, want)
	}
}

func TestFileRowToEntry(t *testing.T) {
	created := time.Date(2026, 9, 12, 18, 0, 37, 0, time.UTC)
	modified := time.Date(2026, 9, 13, 13, 50, 36, 0, time.UTC)
	e := fileRowToEntry(pgFileRow{
		id: 22484, folderID: 649, title: "Rechnung.pdf",
		size: 123433, version: 2, created: created, modified: modified,
	}, ProviderMySQL)
	if e.ID != "22484" || e.ParentID != "649" {
		t.Errorf("ids = %q/%q", e.ID, e.ParentID)
	}
	if e.Title != "Rechnung.pdf" || e.Kind != File {
		t.Errorf("title/kind = %q/%v", e.Title, e.Kind)
	}
	if e.Size != 123433 || e.Version != 2 {
		t.Errorf("size/version = %d/%d", e.Size, e.Version)
	}
	if e.MIME != "application/pdf" {
		t.Errorf("mime = %q", e.MIME)
	}
	if !e.Created.Equal(created) || !e.Modified.Equal(modified) {
		t.Errorf("times = %v/%v", e.Created, e.Modified)
	}
	if e.Provider != ProviderMySQL {
		t.Errorf("provider = %q", e.Provider)
	}
}

func TestFolderRowToEntry(t *testing.T) {
	modified := time.Date(2026, 8, 1, 10, 30, 0, 0, time.UTC)
	e := folderRowToEntry(pgFolderRow{id: 649, parentID: 647, title: "2025", modified: modified}, ProviderMySQL)
	if e.ID != "649" || e.ParentID != "647" || e.Title != "2025" {
		t.Errorf("folder = %+v", e)
	}
	if e.Kind != Folder {
		t.Errorf("kind = %v, want folder", e.Kind)
	}
	if e.Size != 0 || e.MIME != "" {
		t.Errorf("folder size/mime = %d/%q", e.Size, e.MIME)
	}
	if !e.Modified.Equal(modified) {
		t.Errorf("modified = %v", e.Modified)
	}
}

func TestPGStoreWriteMethodsReadOnly(t *testing.T) {
	s := &pgStore{driver: ProviderPG}
	ctx := context.Background()
	if _, err := s.CreateFolder(ctx, "1", "x"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("CreateFolder err = %v", err)
	}
	if _, err := s.Upload(ctx, "1", "x", nil); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Upload err = %v", err)
	}
	if err := s.Move(ctx, nil, "1"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Move err = %v", err)
	}
	if err := s.Copy(ctx, nil, "1"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Copy err = %v", err)
	}
	if err := s.Rename(ctx, "1", "x"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Rename err = %v", err)
	}
	if err := s.Delete(ctx, nil); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Delete err = %v", err)
	}
}

func TestPGStoreName(t *testing.T) {
	if got := (&pgStore{driver: ProviderPG}).Name(); got != ProviderPG {
		t.Errorf("Name = %q, want %q", got, ProviderPG)
	}
	if got := (&pgStore{driver: ProviderMySQL}).Name(); got != ProviderMySQL {
		t.Errorf("Name = %q, want %q", got, ProviderMySQL)
	}
}

func TestPGStoreStatRejectsNonNumeric(t *testing.T) {
	s := &pgStore{driver: ProviderPG}
	if _, err := s.Stat(context.Background(), "not-a-number"); err == nil {
		t.Error("Stat accepted a non-numeric id")
	}
}
