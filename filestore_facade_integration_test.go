//go:build integration

package onlyoffice

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"
)

// TestIntegrationFacadeCRUD drives the whole operation set through the composed
// facade c.Files(): folder create, upload, stat, list, rename, move, copy,
// delete. Writes must go to REST (the default writeOrder), reads follow
// readOrder (REST when no SQL backend is registered) and every returned Entry
// must report its provider. Destructive — throwaway project, cleaned up.
func TestIntegrationFacadeCRUD(t *testing.T) {
	c := liveClient(t)
	t.Cleanup(func() { cleanupTestProjects(t, c) })
	ctx := context.Background()

	suffix := time.Now().UTC().Format("20060102-150405")
	project, err := c.CreateProject(NewProjectRequest{
		Title:       testProjectPrefix + "facade-" + suffix,
		Description: "go-onlyoffice facade CRUD integration",
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

	f := c.Files()
	if got := f.Write().Name(); got != ProviderREST {
		t.Fatalf("Write().Name() = %q, want %q", got, ProviderREST)
	}
	if got := f.Read().Name(); got != ProviderREST {
		t.Fatalf("Read().Name() = %q, want %q (no SQL backend registered)", got, ProviderREST)
	}

	src, err := f.CreateFolder(ctx, root, "facade-src-"+suffix)
	if err != nil {
		t.Fatalf("CreateFolder src: %v", err)
	}
	if src.Kind != Folder || src.ID == "" {
		t.Fatalf("created src folder: %+v", src)
	}
	if src.Provider != ProviderREST {
		t.Fatalf("CreateFolder provider = %q, want %q", src.Provider, ProviderREST)
	}
	dst, err := f.CreateFolder(ctx, root, "facade-dst-"+suffix)
	if err != nil {
		t.Fatalf("CreateFolder dst: %v", err)
	}
	if dst.Provider != ProviderREST {
		t.Fatalf("CreateFolder dst provider = %q, want %q", dst.Provider, ProviderREST)
	}
	t.Cleanup(func() {
		if err := c.DeleteDavItems(ctx, []string{src.ID, dst.ID}, nil); err != nil {
			t.Logf("cleanup folders: %v", err)
		}
	})

	content := []byte("facade crud " + suffix + "\n")
	up, err := f.Upload(ctx, src.ID, "facade-doc-"+suffix+".txt", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if up.Kind != File || up.ID == "" {
		t.Fatalf("uploaded entry: %+v", up)
	}
	if up.Provider != ProviderREST {
		t.Fatalf("Upload provider = %q, want %q (write order REST first)", up.Provider, ProviderREST)
	}
	if !waitEntry(ctx, f, src.ID, up.ID, 15*time.Second) {
		t.Fatalf("uploaded %s not listed in src", up.ID)
	}

	st, err := f.Stat(ctx, up.ID)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if st.ID != up.ID || st.Kind != File {
		t.Fatalf("Stat = %+v", st)
	}
	if st.Provider != ProviderREST {
		t.Fatalf("Stat provider = %q, want %q (read order REST)", st.Provider, ProviderREST)
	}

	list, err := f.List(ctx, src.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if e := entryByID(list, up.ID); e == nil {
		t.Fatalf("uploaded %s not in List(src)", up.ID)
	} else if e.Provider != ProviderREST {
		t.Fatalf("List provider = %q, want %q", e.Provider, ProviderREST)
	}

	renamed := "facade-renamed-" + suffix + ".txt"
	renameEventually(t, ctx, f, up.ID, renamed)

	moveEventually(t, ctx, f, up.ID, dst.ID)
	if !waitEntry(ctx, f, dst.ID, up.ID, 20*time.Second) {
		t.Fatalf("moved file %s not in dst", up.ID)
	}

	copied := copyEventually(t, ctx, f, up.ID, src.ID, 20*time.Second)
	if copied == nil {
		t.Fatalf("no copy found in src after Copy")
	}
	if copied.Provider != ProviderREST {
		t.Fatalf("Copy provider = %q, want %q", copied.Provider, ProviderREST)
	}

	if err := f.Delete(ctx, []string{up.ID, copied.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !waitNoEntry(ctx, f, dst.ID, up.ID, 20*time.Second) {
		t.Fatalf("file %s still present in dst after delete", up.ID)
	}
	if !waitNoEntry(ctx, f, src.ID, copied.ID, 20*time.Second) {
		t.Fatalf("copy %s still present in src after delete", copied.ID)
	}
}
