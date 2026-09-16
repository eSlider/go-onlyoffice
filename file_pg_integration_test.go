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

	var buf bytes.Buffer
	n, err := store.Download(ctx, fileID, &buf)
	if err != nil {
		t.Fatalf("sql Download(%s): %v", fileID, err)
	}
	if n == 0 || n != dbFile.Size {
		t.Errorf("sql Download = %d bytes, stat says %d", n, dbFile.Size)
	}
	if os.Getenv("MINIO_ACCESS_KEY") != "" && os.Getenv("MINIO_SECRET_KEY") != "" {
		var restBuf bytes.Buffer
		rn, err := rest.Download(ctx, fileID, &restBuf)
		if err != nil {
			t.Fatalf("rest Download(%s): %v", fileID, err)
		}
		if rn != n || !bytes.Equal(restBuf.Bytes(), buf.Bytes()) {
			t.Errorf("download mismatch sql=%d rest=%d bytes", n, rn)
		}
	} else {
		t.Log("MINIO_ACCESS_KEY/MINIO_SECRET_KEY not set — REST download cross-check skipped")
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
