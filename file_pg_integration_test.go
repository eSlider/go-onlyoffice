//go:build integration

package onlyoffice

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// TestIntegrationPGStore exercises the read-only SQL backend against the live
// Community Server database and cross-checks list/stat/download with the REST
// FileStore. It needs ONLYOFFICE_DSN plus the usual ONLYOFFICE_URL/USER/PASS;
// ONLYOFFICE_PG_TEST_FILE_ID / ONLYOFFICE_PG_TEST_FOLDER_ID pick a real file
// (a file reachable over REST too). Download streams from MinIO, so it also
// needs MINIO_ACCESS_KEY/MINIO_SECRET_KEY.
//
// The live Community Server runs on MySQL; PostgreSQL is supported by the same
// code path when the DSN says so.
func TestIntegrationPGStore(t *testing.T) {
	cfg := PGConfigFromEnv()
	if strings.TrimSpace(cfg.DSN) == "" {
		t.Skip("ONLYOFFICE_DSN not set — skipping SQL store integration test")
	}
	store, err := NewPGStore(cfg)
	if err != nil {
		t.Fatalf("NewPGStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	t.Logf("sql store backend: %s", store.Name())
	ctx := context.Background()

	if err := testPGStoreReadOnly(ctx, store); err != nil {
		t.Fatal(err)
	}

	fileID := strings.TrimSpace(os.Getenv("ONLYOFFICE_PG_TEST_FILE_ID"))
	folderID := strings.TrimSpace(os.Getenv("ONLYOFFICE_PG_TEST_FOLDER_ID"))
	if fileID == "" || folderID == "" {
		t.Skip("ONLYOFFICE_PG_TEST_FILE_ID / ONLYOFFICE_PG_TEST_FOLDER_ID not set — skipping live comparison")
	}

	c := liveClient(t)
	rest := c.Files()

	dbFile, err := store.Stat(ctx, fileID)
	if err != nil {
		t.Fatalf("sql Stat(%s): %v", fileID, err)
	}
	restFile, err := rest.Stat(ctx, fileID)
	if err != nil {
		t.Fatalf("rest Stat(%s): %v", fileID, err)
	}
	if dbFile.Kind != File {
		t.Errorf("sql kind = %v, want file", dbFile.Kind)
	}
	if dbFile.ID != restFile.ID || dbFile.Title != restFile.Title || dbFile.ParentID != restFile.ParentID {
		t.Errorf("stat mismatch sql=%+v rest=%+v", dbFile, restFile)
	}
	// GetFile omits contentLength, so size is only comparable when REST has it.
	if restFile.Size > 0 && dbFile.Size != restFile.Size {
		t.Errorf("size sql=%d rest=%d", dbFile.Size, restFile.Size)
	}
	if d := dbFile.Modified.Sub(restFile.Modified); d > 2*time.Minute || d < -2*time.Minute {
		t.Errorf("modified sql=%v rest=%v", dbFile.Modified, restFile.Modified)
	}

	list, err := store.List(ctx, folderID)
	if err != nil {
		t.Fatalf("sql List(%s): %v", folderID, err)
	}
	if entryByID(list, fileID) == nil {
		t.Errorf("file %s not in sql List(%s)", fileID, folderID)
	}

	// Every file the REST layer can see in the folder must be in the SQL list
	// (the SQL store sees more, so only assert this direction).
	restList, err := rest.List(ctx, folderID)
	if err != nil {
		t.Fatalf("rest List(%s): %v", folderID, err)
	}
	dbIDs := make(map[string]bool, len(list))
	for _, e := range list {
		dbIDs[e.ID] = true
	}
	for _, e := range restList {
		if e.Kind == File && !dbIDs[e.ID] {
			t.Errorf("rest file %s (%q) missing from sql list", e.ID, e.Title)
		}
	}

	// Download reads the object store, not the database, so it only runs with
	// the MinIO credentials configured (the portal's S3 layout). Without them
	// the DSN-only assertions above still prove the SQL reads.
	if os.Getenv("MINIO_ACCESS_KEY") == "" || os.Getenv("MINIO_SECRET_KEY") == "" {
		t.Log("MINIO_ACCESS_KEY/MINIO_SECRET_KEY not set — SQL download cross-check skipped")
		return
	}

	var buf bytes.Buffer
	n, err := store.Download(ctx, fileID, &buf)
	if err != nil {
		t.Fatalf("sql Download(%s): %v", fileID, err)
	}
	if n == 0 || n != dbFile.Size {
		t.Errorf("sql Download = %d bytes, stat says %d", n, dbFile.Size)
	}
	var restBuf bytes.Buffer
	rn, err := rest.Download(ctx, fileID, &restBuf)
	if err != nil {
		t.Fatalf("rest Download(%s): %v", fileID, err)
	}
	if rn != n || !bytes.Equal(restBuf.Bytes(), buf.Bytes()) {
		t.Errorf("download mismatch sql=%d rest=%d bytes", n, rn)
	}
}

// TestIntegrationSQLFacade proves that reads are served by the SQL store when
// it is part of the file client, not by REST. It needs ONLYOFFICE_DSN plus
// ONLYOFFICE_PG_TEST_FILE_ID / ONLYOFFICE_PG_TEST_FOLDER_ID and the usual REST
// credentials (for the cross-check). Every Entry served by SQL carries
// Provider "mysql"; REST entries carry "rest", so the provider is the proof of
// which backend answered.
func TestIntegrationSQLFacade(t *testing.T) {
	cfg := PGConfigFromEnv()
	if strings.TrimSpace(cfg.DSN) == "" {
		t.Skip("ONLYOFFICE_DSN not set — skipping SQL facade integration test")
	}
	fileID := strings.TrimSpace(os.Getenv("ONLYOFFICE_PG_TEST_FILE_ID"))
	folderID := strings.TrimSpace(os.Getenv("ONLYOFFICE_PG_TEST_FOLDER_ID"))
	if fileID == "" || folderID == "" {
		t.Skip("ONLYOFFICE_PG_TEST_FILE_ID / ONLYOFFICE_PG_TEST_FOLDER_ID not set — skipping SQL facade integration test")
	}
	c := liveClient(t)
	ctx := context.Background()

	sqlStore, err := c.SQLFileStore()
	if err != nil {
		t.Fatalf("SQLFileStore: %v", err)
	}
	if closer, ok := sqlStore.(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closer.Close() })
	}
	if sqlStore.Name() == ProviderREST {
		t.Fatalf("SQLFileStore returned REST")
	}

	// Direct constructor: Client.FileStore("pg") must not be REST either.
	direct := c.FileStore("pg")
	if direct == nil || direct.Name() == ProviderREST {
		t.Fatalf("FileStore(\"pg\") = %v, want SQL backend", direct)
	}
	if closer, ok := direct.(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closer.Close() })
	}

	f := c.Files()
	f.RegisterStore(ProviderPG, sqlStore)
	if got := f.Read().Name(); got != sqlStore.Name() {
		t.Fatalf("facade read backend = %q, want %q (SQL)", got, sqlStore.Name())
	}

	got, err := f.Stat(ctx, fileID)
	if err != nil {
		t.Fatalf("facade Stat(%s): %v", fileID, err)
	}
	if got.Provider != ProviderMySQL {
		t.Errorf("facade Stat provider = %q, want %q (SQL, not REST)", got.Provider, ProviderMySQL)
	}
	want, err := c.FileStore(ProviderREST).Stat(ctx, fileID)
	if err != nil {
		t.Fatalf("rest Stat(%s): %v", fileID, err)
	}
	if got.ID != want.ID || got.Title != want.Title || got.ParentID != want.ParentID {
		t.Errorf("facade SQL stat %+v != REST %+v", got, want)
	}

	list, err := f.List(ctx, folderID)
	if err != nil {
		t.Fatalf("facade List(%s): %v", folderID, err)
	}
	entry := entryByID(list, fileID)
	if entry == nil {
		t.Fatalf("file %s not in facade List(%s)", fileID, folderID)
	}
	if entry.Provider != ProviderMySQL {
		t.Errorf("facade List provider = %q, want %q", entry.Provider, ProviderMySQL)
	}

	d, err := direct.Stat(ctx, fileID)
	if err != nil {
		t.Fatalf("FileStore(\"pg\").Stat(%s): %v", fileID, err)
	}
	if d.Provider != ProviderMySQL {
		t.Errorf("FileStore(\"pg\") provider = %q, want %q", d.Provider, ProviderMySQL)
	}
}

// testPGStoreReadOnly asserts that every write method returns ErrReadOnly.
func testPGStoreReadOnly(ctx context.Context, s *pgStore) error {
	if _, err := s.CreateFolder(ctx, "1", "x"); !errors.Is(err, ErrReadOnly) {
		return errors.New("CreateFolder did not return ErrReadOnly")
	}
	if _, err := s.Upload(ctx, "1", "x", strings.NewReader("x")); !errors.Is(err, ErrReadOnly) {
		return errors.New("Upload did not return ErrReadOnly")
	}
	if err := s.Move(ctx, []string{"1"}, "2"); !errors.Is(err, ErrReadOnly) {
		return errors.New("Move did not return ErrReadOnly")
	}
	if err := s.Copy(ctx, []string{"1"}, "2"); !errors.Is(err, ErrReadOnly) {
		return errors.New("Copy did not return ErrReadOnly")
	}
	if err := s.Rename(ctx, "1", "x"); !errors.Is(err, ErrReadOnly) {
		return errors.New("Rename did not return ErrReadOnly")
	}
	if err := s.Delete(ctx, []string{"1"}); !errors.Is(err, ErrReadOnly) {
		return errors.New("Delete did not return ErrReadOnly")
	}
	return nil
}
