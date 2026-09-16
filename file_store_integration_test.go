//go:build integration

package onlyoffice

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"
)

// TestIntegrationFileStores runs the same operation set (create folder, upload,
// list, stat, download, move, copy, rename, delete) through the REST and DAV
// FileStore adapters against a throwaway project Documents folder. Destructive
// — only run against instances you own.
//
// The Documents fileops API is asynchronous: a move/copy/delete is accepted
// immediately and becomes visible a moment later, so effects are polled.
func TestIntegrationFileStores(t *testing.T) {
	c := liveClient(t)
	t.Cleanup(func() { cleanupTestProjects(t, c) })
	ctx := context.Background()

	suffix := time.Now().UTC().Format("20060102-150405")
	project, err := c.CreateProject(NewProjectRequest{
		Title:       testProjectPrefix + "store-" + suffix,
		Description: "go-onlyoffice file store integration",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.ID == nil {
		t.Fatal("created project without id")
	}
	root, err := c.projectFolderID(ctx, strconv.Itoa(*project.ID))
	if err != nil {
		t.Fatalf("projectFolderID: %v", err)
	}

	for _, backend := range []string{ProviderREST, ProviderDAV} {
		t.Run(backend, func(t *testing.T) {
			testFileStoreOps(t, ctx, c, c.FileStore(backend), root, suffix)
		})
	}
}

func testFileStoreOps(t *testing.T, ctx context.Context, c *Client, store FileStore, root, suffix string) {
	t.Helper()
	content := []byte("file store " + store.Name() + " " + suffix + "\n")

	src, err := store.CreateFolder(ctx, root, "fs-src-"+suffix)
	if err != nil {
		t.Fatalf("CreateFolder src: %v", err)
	}
	if src.Kind != Folder || src.ID == "" {
		t.Fatalf("created src folder: %+v", src)
	}
	dst, err := store.CreateFolder(ctx, root, "fs-dst-"+suffix)
	if err != nil {
		t.Fatalf("CreateFolder dst: %v", err)
	}
	if dst.Kind != Folder || dst.ID == "" {
		t.Fatalf("created dst folder: %+v", dst)
	}
	t.Cleanup(func() {
		if err := c.DeleteDavItems(ctx, []string{src.ID, dst.ID}, nil); err != nil {
			t.Logf("cleanup folders: %v", err)
		}
	})

	up, err := store.Upload(ctx, src.ID, "doc-"+suffix+".txt", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if up.Kind != File || up.ID == "" {
		t.Fatalf("uploaded entry: %+v", up)
	}
	if !waitEntry(ctx, store, src.ID, up.ID, 15*time.Second) {
		t.Fatalf("uploaded %s not listed in src", up.ID)
	}

	st, err := store.Stat(ctx, up.ID)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if st.ID != up.ID || st.Kind != File {
		t.Fatalf("stat = %+v", st)
	}

	var buf bytes.Buffer
	n, err := store.Download(ctx, up.ID, &buf)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if n != int64(len(content)) || !bytes.Equal(buf.Bytes(), content) {
		t.Fatalf("download mismatch: got %d bytes %q want %d", n, buf.String(), len(content))
	}

	moveEventually(t, ctx, store, up.ID, dst.ID)
	if !waitEntry(ctx, store, dst.ID, up.ID, 20*time.Second) {
		t.Fatalf("moved file %s not in dst", up.ID)
	}

	if err := store.Copy(ctx, []string{up.ID}, src.ID); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	copied := waitOtherFile(ctx, store, src.ID, up.ID, 20*time.Second)
	if copied == nil {
		t.Fatalf("no copy found in src after Copy")
	}

	newTitle := "renamed-" + suffix + ".txt"
	renameEventually(t, ctx, store, up.ID, newTitle)

	if err := store.Delete(ctx, []string{up.ID, copied.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !waitNoEntry(ctx, store, dst.ID, up.ID, 20*time.Second) {
		t.Fatalf("file %s still present in dst after delete", up.ID)
	}
	if !waitNoEntry(ctx, store, src.ID, copied.ID, 20*time.Second) {
		t.Fatalf("copy %s still present in src after delete", copied.ID)
	}
}

// moveEventually issues Move and retries while the operation is not visible yet
// (the fileops API accepts asynchronously and occasionally rejects a move that
// raced the just-finished upload).
func moveEventually(t *testing.T, ctx context.Context, store FileStore, id, dstID string) {
	t.Helper()
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if lastErr = store.Move(ctx, []string{id}, dstID); lastErr == nil {
			if waitEntry(ctx, store, dstID, id, 6*time.Second) {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("Move %s -> %s: %v", id, dstID, lastErr)
}

func renameEventually(t *testing.T, ctx context.Context, store FileStore, id, title string) {
	t.Helper()
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if lastErr = store.Rename(ctx, id, title); lastErr == nil {
			if e, err := store.Stat(ctx, id); err == nil && e.Title == title {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("Rename %s -> %q: %v", id, title, lastErr)
}

func waitEntry(ctx context.Context, store FileStore, parentID, id string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if list, err := store.List(ctx, parentID); err == nil && entryByID(list, id) != nil {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func waitNoEntry(ctx context.Context, store FileStore, parentID, id string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if list, err := store.List(ctx, parentID); err == nil && entryByID(list, id) == nil {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func waitOtherFile(ctx context.Context, store FileStore, parentID, id string, d time.Duration) *Entry {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if list, err := store.List(ctx, parentID); err == nil {
			if e := firstFileOtherThan(list, id); e != nil {
				return e
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func entryByID(entries []Entry, id string) *Entry {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}

func firstFileOtherThan(entries []Entry, id string) *Entry {
	for i := range entries {
		if entries[i].Kind == File && entries[i].ID != id {
			return &entries[i]
		}
	}
	return nil
}
