package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

// fakeFolderStore is an in-memory onlyoffice.FileStore for the ensure-path
// logic. It records created folders so idempotency can be asserted.
type fakeFolderStore struct {
	nextID  int
	entries map[string][]onlyoffice.Entry
	created []string
}

func newFakeFolderStore() *fakeFolderStore {
	return &fakeFolderStore{entries: map[string][]onlyoffice.Entry{}}
}

func (f *fakeFolderStore) Name() string { return "fake" }

func (f *fakeFolderStore) List(_ context.Context, parent string) ([]onlyoffice.Entry, error) {
	return append([]onlyoffice.Entry(nil), f.entries[parent]...), nil
}

func (f *fakeFolderStore) CreateFolder(_ context.Context, parent, title string) (onlyoffice.Entry, error) {
	f.nextID++
	e := onlyoffice.Entry{
		ID:       fmt.Sprintf("id-%d", f.nextID),
		ParentID: parent,
		Title:    title,
		Kind:     onlyoffice.Folder,
	}
	f.entries[parent] = append(f.entries[parent], e)
	f.created = append(f.created, parent+"/"+title)
	return e, nil
}

func (f *fakeFolderStore) Stat(context.Context, string) (onlyoffice.Entry, error) {
	return onlyoffice.Entry{}, errors.New("not implemented")
}

func (f *fakeFolderStore) Upload(context.Context, string, string, io.Reader) (onlyoffice.Entry, error) {
	return onlyoffice.Entry{}, errors.New("not implemented")
}

func (f *fakeFolderStore) Download(context.Context, string, io.Writer) (int64, error) {
	return 0, errors.New("not implemented")
}

func (f *fakeFolderStore) Move(context.Context, []string, string) error {
	return errors.New("not implemented")
}

func (f *fakeFolderStore) Copy(context.Context, []string, string) error {
	return errors.New("not implemented")
}

func (f *fakeFolderStore) Rename(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (f *fakeFolderStore) Delete(context.Context, []string) error {
	return errors.New("not implemented")
}

// fakeUploader is the *onlyoffice.Client slice dav upload depends on.
type fakeUploader struct {
	conflict bool
	uploads  []string
	replaced []string
}

func (f *fakeUploader) UploadToFolderReplacing(_ context.Context, folderID, localPath string) (*onlyoffice.FileEntry, []int, error) {
	f.uploads = append(f.uploads, localPath)
	ids := []int(nil)
	if f.conflict {
		ids = []int{7}
		f.replaced = append(f.replaced, folderID+"#7")
	}
	return fakeFileEntry(99, localPath, folderID), ids, nil
}

func (f *fakeUploader) AssertNoFileConflict(_ context.Context, folderID, localPath string) error {
	if f.conflict {
		return fmt.Errorf("%w: conflict in folder %s for %s", onlyoffice.ErrFileExists, folderID, localPath)
	}
	return nil
}

func (f *fakeUploader) UploadToFolder(_ context.Context, folderID, localPath string) (*onlyoffice.FileEntry, error) {
	f.uploads = append(f.uploads, localPath)
	return fakeFileEntry(99, localPath, folderID), nil
}

func fakeFileEntry(id int, localPath, folderID string) *onlyoffice.FileEntry {
	num := json.Number(fmt.Sprintf("%d", id))
	title := localPath[strings.LastIndex(localPath, "/")+1:]
	return &onlyoffice.FileEntry{
		ID:       &num,
		Title:    &title,
		FolderID: &num,
	}
}

func TestEnsurePathCreatesNestedFolders(t *testing.T) {
	store := newFakeFolderStore()
	ctx := context.Background()

	got, err := ensurePath(ctx, store, "root", "Banks/Caixa")
	if err != nil {
		t.Fatalf("ensurePath: %v", err)
	}
	if got.Kind != onlyoffice.Folder || got.ID == "" {
		t.Fatalf("ensurePath returned %+v, want a folder with an id", got)
	}
	if want := []string{"root/Banks", "id-1/Caixa"}; !equalStrings(store.created, want) {
		t.Fatalf("created %v, want %v", store.created, want)
	}
}

func TestEnsurePathIsIdempotent(t *testing.T) {
	store := newFakeFolderStore()
	ctx := context.Background()

	first, err := ensurePath(ctx, store, "root", "Banks/Caixa")
	if err != nil {
		t.Fatalf("ensurePath first: %v", err)
	}
	created := len(store.created)

	second, err := ensurePath(ctx, store, "root", "Banks/Caixa")
	if err != nil {
		t.Fatalf("ensurePath second: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("second run id = %q, want %q (no duplicate)", second.ID, first.ID)
	}
	if len(store.created) != created {
		t.Fatalf("second run created folders: %v", store.created)
	}
}

func TestEnsurePathReusesExistingFolder(t *testing.T) {
	store := newFakeFolderStore()
	store.entries["root"] = []onlyoffice.Entry{
		{ID: "banks-id", Title: "Banks", Kind: onlyoffice.Folder},
	}
	store.entries["banks-id"] = []onlyoffice.Entry{
		{ID: "caixa-id", Title: "Caixa", Kind: onlyoffice.Folder},
	}

	got, err := ensurePath(context.Background(), store, "root", "Banks/Caixa")
	if err != nil {
		t.Fatalf("ensurePath: %v", err)
	}
	if got.ID != "caixa-id" {
		t.Fatalf("id = %q, want caixa-id", got.ID)
	}
	if len(store.created) != 0 {
		t.Fatalf("created %v, want none", store.created)
	}
}

func TestEnsurePathRejectsEmptyAndDotDot(t *testing.T) {
	store := newFakeFolderStore()
	for _, path := range []string{"", "/", "Banks/../Caixa"} {
		if _, err := ensurePath(context.Background(), store, "root", path); err == nil {
			t.Fatalf("ensurePath(%q) = nil error, want failure", path)
		}
	}
}

func TestUploadLocalReplacesByDefault(t *testing.T) {
	up := &fakeUploader{conflict: true}
	results, err := uploadLocal(context.Background(), up, "folder-1", []string{"a/f.xlsx"}, true)
	if err != nil {
		t.Fatalf("uploadLocal: %v", err)
	}
	if len(results) != 1 || results[0].Entry.Title != "f.xlsx" {
		t.Fatalf("results = %+v", results)
	}
	if len(results[0].Replaced) != 1 || results[0].Replaced[0] != 7 {
		t.Fatalf("replaced = %v, want [7]", results[0].Replaced)
	}
	if len(up.replaced) != 1 {
		t.Fatalf("UploadToFolderReplacing not used for replace: %v", up.replaced)
	}
}

func TestUploadLocalNoReplaceFailsOnConflict(t *testing.T) {
	up := &fakeUploader{conflict: true}
	_, err := uploadLocal(context.Background(), up, "folder-1", []string{"a/f.xlsx"}, false)
	if !errors.Is(err, onlyoffice.ErrFileExists) {
		t.Fatalf("err = %v, want ErrFileExists", err)
	}
	if len(up.uploads) != 0 {
		t.Fatalf("uploaded despite conflict: %v", up.uploads)
	}
}

func TestUploadLocalNoReplaceUploadsWhenFree(t *testing.T) {
	up := &fakeUploader{}
	results, err := uploadLocal(context.Background(), up, "folder-1", []string{"a/f.xlsx", "a/g.pdf"}, false)
	if err != nil {
		t.Fatalf("uploadLocal: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	if len(up.uploads) != 2 {
		t.Fatalf("uploads = %v", up.uploads)
	}
}

func TestDavRegistersUploadAndEnsurePath(t *testing.T) {
	for _, name := range []string{"ensure-path", "upload"} {
		cmd, _, err := rootCmd.Find([]string{"dav", name})
		if err != nil {
			t.Fatalf("dav %s not registered: %v", name, err)
		}
		if cmd.Name() != name {
			t.Fatalf("resolved %q, want %q", cmd.Name(), name)
		}
	}
	upload, _, err := rootCmd.Find([]string{"dav", "upload"})
	if err != nil {
		t.Fatal(err)
	}
	flag := upload.Flags().Lookup("replace")
	if flag == nil || flag.DefValue != "true" {
		t.Fatalf("upload --replace flag = %+v, want default true", flag)
	}
	ensure, _, err := rootCmd.Find([]string{"dav", "ensure-path"})
	if err != nil {
		t.Fatal(err)
	}
	if ensure.Flags().Lookup("under") == nil {
		t.Fatal("ensure-path missing --under flag")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
