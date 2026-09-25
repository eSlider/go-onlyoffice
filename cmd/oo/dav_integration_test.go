//go:build integration

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/internal/bootstrap"
)

// TestIntegrationDavEnsurePathUpload exercises the dav ensure-path/upload
// helpers against a live instance: idempotent folder creation, upload, replace
// and no-replace conflict. Destructive (creates and removes a throwaway
// folder); skips cleanly without credentials. Only run against instances you
// own: go test -tags=integration ./cmd/oo -run TestIntegrationDavEnsurePathUpload
func TestIntegrationDavEnsurePathUpload(t *testing.T) {
	ctx := context.Background()
	c, err := bootstrap.NewClient(ctx)
	if err != nil {
		t.Skipf("no live OnlyOffice credentials: %v", err)
	}
	root, err := myDocumentsID(ctx, c)
	if err != nil {
		t.Skipf("My documents root unavailable: %v", err)
	}

	store := c.FileStore(onlyoffice.ProviderDAV)
	suffix := time.Now().UTC().Format("20060102-150405")
	base := "oo-it-" + suffix

	first, err := ensurePath(ctx, store, root, base+"/nested")
	if err != nil {
		t.Fatalf("ensurePath: %v", err)
	}
	baseID := findFolderID(t, ctx, store, root, base)
	t.Cleanup(func() { _ = store.Delete(context.Background(), []string{baseID}) })

	second, err := ensurePath(ctx, store, root, base+"/nested")
	if err != nil {
		t.Fatalf("ensurePath (second): %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("ensure-path not idempotent: %s != %s", second.ID, first.ID)
	}
	if n := countFolders(t, ctx, store, baseID, "nested"); n != 1 {
		t.Fatalf("nested folder duplicated: %d children named nested, want 1", n)
	}

	local := filepath.Join(t.TempDir(), "oo-it-"+suffix+".xlsx")
	if err := os.WriteFile(local, []byte("integration dav upload "+suffix+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(local)

	if _, err := uploadLocal(ctx, c, first.ID, []string{local}, true); err != nil {
		t.Fatalf("uploadLocal: %v", err)
	}
	if n := countFiles(t, ctx, store, first.ID, name); n != 1 {
		t.Fatalf("after upload: %d files named %s, want 1", n, name)
	}

	if _, err := uploadLocal(ctx, c, first.ID, []string{local}, true); err != nil {
		t.Fatalf("uploadLocal (replace): %v", err)
	}
	if n := countFiles(t, ctx, store, first.ID, name); n != 1 {
		t.Fatalf("after replace: %d files named %s, want 1", n, name)
	}

	if _, err := uploadLocal(ctx, c, first.ID, []string{local}, false); !errors.Is(err, onlyoffice.ErrFileExists) {
		t.Fatalf("no-replace conflict err = %v, want ErrFileExists", err)
	}
}

func findFolderID(t *testing.T, ctx context.Context, store onlyoffice.FileStore, parent, title string) string {
	t.Helper()
	for _, e := range mustList(t, ctx, store, parent) {
		if e.Kind == onlyoffice.Folder && strings.EqualFold(e.Title, title) {
			return e.ID
		}
	}
	t.Fatalf("folder %q not found under %s", title, parent)
	return ""
}

func countFolders(t *testing.T, ctx context.Context, store onlyoffice.FileStore, parent, title string) int {
	t.Helper()
	n := 0
	for _, e := range mustList(t, ctx, store, parent) {
		if e.Kind == onlyoffice.Folder && strings.EqualFold(e.Title, title) {
			n++
		}
	}
	return n
}

func countFiles(t *testing.T, ctx context.Context, store onlyoffice.FileStore, parent, title string) int {
	t.Helper()
	n := 0
	for _, e := range mustList(t, ctx, store, parent) {
		if e.Kind == onlyoffice.File && e.Title == title {
			n++
		}
	}
	return n
}

func mustList(t *testing.T, ctx context.Context, store onlyoffice.FileStore, parent string) []onlyoffice.Entry {
	t.Helper()
	entries, err := store.List(ctx, parent)
	if err != nil {
		t.Fatalf("List(%s): %v", parent, err)
	}
	return entries
}
