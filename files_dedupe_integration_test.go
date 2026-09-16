//go:build integration

package onlyoffice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestIntegrationFileDedup proves on a live OnlyOffice portal that the file
// dedup helpers find real duplicates and delete only the redundant copies.
// It creates a throwaway "go-onlyoffice-test-" project (removed by cleanup),
// places same stem|ext files in two subfolders and in the project root, then
// exercises FindProjectDuplicates, mergeProjectRootForDedupe,
// ApplyDedupGroups and DeleteFilesByDedupKey. Destructive — run only against
// an instance you own.
func TestIntegrationFileDedup(t *testing.T) {
	c := liveClient(t)
	t.Cleanup(func() { cleanupTestProjects(t, c) })
	ctx := context.Background()

	suffix := time.Now().UTC().Format("20060102-150405")
	project, err := c.CreateProject(NewProjectRequest{
		Title:       testProjectPrefix + "dedup-" + suffix,
		Description: "go-onlyoffice file dedup integration",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.ID == nil {
		t.Fatal("created project without id")
	}
	pid := strconv.Itoa(*project.ID)

	root := projectFolderEventually(t, ctx, c, pid)

	aID := createFolderLive(t, ctx, c, root, "A-"+suffix)
	bID := createFolderLive(t, ctx, c, root, "B-"+suffix)
	createFolderLive(t, ctx, c, root, "_trash-"+suffix)

	// Check IsTrashFolderTitle against a real live folder title.
	if !IsTrashFolderTitle("_trash-" + suffix) {
		t.Fatalf("IsTrashFolderTitle(%q) = false for a live _trash folder", "_trash-"+suffix)
	}

	stem := "dedup-" + suffix
	local := writeLocalFile(t, stem+".txt", []byte("dedup integration "+suffix+"\n"))

	a1 := uploadFolderLive(t, ctx, c, aID, local)
	a2 := uploadFolderLive(t, ctx, c, aID, local)
	b1 := uploadFolderLive(t, ctx, c, bID, local)
	r1 := uploadFolderLive(t, ctx, c, root, local)
	r2 := uploadFolderLive(t, ctx, c, root, local)
	t.Logf("uploaded a1=%d a2=%d b1=%d r1=%d r2=%d",
		FileEntryNumericID(a1), FileEntryNumericID(a2), FileEntryNumericID(b1),
		FileEntryNumericID(r1), FileEntryNumericID(r2))

	if n := dedupWaitCount(t, ctx, c, aID, stem, ".txt", 2, 30*time.Second); n != 2 {
		t.Fatalf("folder A has %d copies after upload, want 2", n)
	}
	if n := dedupWaitCount(t, ctx, c, bID, stem, ".txt", 1, 30*time.Second); n != 1 {
		t.Fatalf("folder B has %d copies after upload, want 1", n)
	}
	if n := dedupWaitCount(t, ctx, c, root, stem, ".txt", 2, 30*time.Second); n != 2 {
		t.Fatalf("project root has %d copies after upload, want 2", n)
	}

	// #2 mergeProjectRootForDedupe on the live tree: the project root that
	// carries documents must be part of the scan exactly once.
	folders, byFolder, rootFiles := liveProjectIndex(t, ctx, c, pid, root)
	merged, mergedBy := mergeProjectRootForDedupe(root, folders, byFolder, rootFiles)
	rootCount := 0
	for _, folder := range merged {
		if folder != nil && folder.ID != nil && folder.ID.String() == root {
			rootCount++
		}
	}
	if rootCount != 1 {
		t.Fatalf("mergeProjectRootForDedupe: project root appears %d times in live tree, want 1", rootCount)
	}
	if got := len(FindFilesByDedupKey(mergedBy[root], stem, ".txt")); got != 2 {
		t.Fatalf("mergeProjectRootForDedupe: root carries %d matching files, want 2", got)
	}

	// Within-folder scan: a duplicate pair in A and in the project root.
	within := FindProjectDuplicates(merged, mergedBy, DedupOptions{})
	removesByFolder := map[string]int{}
	for _, g := range within {
		removesByFolder[g.FolderID] = len(g.Remove)
	}
	if len(within) != 2 || removesByFolder[aID] != 1 || removesByFolder[root] != 1 {
		t.Fatalf("within-folder groups = %d (%v), want exactly A:1 root:1", len(within), removesByFolder)
	}

	// DeleteFilesByDedupKey removes every stem|ext copy in one folder.
	removed, err := c.DeleteFilesByDedupKey(ctx, aID, stem, ".txt")
	if err != nil {
		t.Fatalf("DeleteFilesByDedupKey(A): %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("DeleteFilesByDedupKey(A) removed %v, want 2 ids", removed)
	}
	if n := dedupWaitCount(t, ctx, c, aID, stem, ".txt", 0, 30*time.Second); n != 0 {
		t.Fatalf("folder A still has %d copies after DeleteFilesByDedupKey", n)
	}

	// Re-create the A duplicates so the cross-folder project scan can be
	// applied and a single survivor proven by polling.
	uploadFolderLive(t, ctx, c, aID, local)
	uploadFolderLive(t, ctx, c, aID, local)
	if n := dedupWaitCount(t, ctx, c, aID, stem, ".txt", 2, 30*time.Second); n != 2 {
		t.Fatalf("folder A has %d re-uploaded copies, want 2", n)
	}

	// Cross-folder dry-run over the live project: one key, five copies, and
	// the project-root copies must be part of the group (root merge live).
	groups, deleted, err := dedupeProjectEventually(t, ctx, c, pid, DedupOptions{CrossFolder: true}, false)
	if err != nil {
		t.Fatalf("DedupeProject: %v", err)
	}
	if len(deleted) != 0 {
		t.Fatalf("dry-run DedupeProject deleted %v", deleted)
	}
	if len(groups) != 1 {
		t.Fatalf("DedupeProject cross-folder groups = %d, want 1 (%+v)", len(groups), groups)
	}
	if len(groups[0].Remove) != 4 {
		t.Fatalf("cross-folder group removes %d files, want 4", len(groups[0].Remove))
	}
	if !dedupGroupHasID(groups[0], FileEntryNumericID(r1)) && !dedupGroupHasID(groups[0], FileEntryNumericID(r2)) {
		t.Fatalf("cross-folder group does not include a project-root copy (merge not applied)")
	}

	keepID := FileEntryNumericID(groups[0].Keep)
	deleted, err = c.ApplyDedupGroups(ctx, groups)
	if err != nil {
		t.Fatalf("ApplyDedupGroups: %v", err)
	}
	if len(deleted) != 4 {
		t.Fatalf("ApplyDedupGroups deleted %v, want 4 ids", deleted)
	}

	survivorID, total := dedupWaitTotal(t, ctx, c, []string{aID, bID, root}, stem, ".txt", 1, 40*time.Second)
	if total != 1 {
		t.Fatalf("after ApplyDedupGroups %d copies survive, want 1", total)
	}
	if survivorID != keepID {
		t.Fatalf("remaining copy id = %d, want kept id %d", survivorID, keepID)
	}

	// The removed ids must really be gone from every folder.
	gone := map[int64]bool{
		FileEntryNumericID(a2): true,
		FileEntryNumericID(b1): true,
		FileEntryNumericID(r1): true,
		FileEntryNumericID(r2): true,
	}
	for _, fid := range []string{aID, bID, root} {
		files, err := c.FolderFiles(ctx, fid)
		if err != nil {
			t.Fatalf("FolderFiles %s: %v", fid, err)
		}
		for _, f := range FindFilesByDedupKey(files, stem, ".txt") {
			id := FileEntryNumericID(f)
			if gone[id] {
				t.Fatalf("removed id %d still present in folder %s", id, fid)
			}
		}
	}
	t.Logf("survivor id=%d keep id=%d, deleted=%v", survivorID, keepID, deleted)
}

// createFolderLive creates a subfolder (retrying transient 5xx) and returns
// its Documents folder id.
func createFolderLive(t *testing.T, ctx context.Context, c *Client, parentID, title string) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var (
		m   map[string]any
		err error
	)
	for {
		m, err = c.CreateFolder(ctx, parentID, title)
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("CreateFolder %q: %v", title, err)
	}
	if m == nil {
		t.Fatalf("CreateFolder %q: empty response", title)
	}
	switch v := m["id"].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case json.Number:
		return v.String()
	case string:
		if v != "" {
			return v
		}
	}
	t.Fatalf("CreateFolder %q: no id in response %#v", title, m)
	return ""
}

// writeLocalFile writes content to a temp file and returns its path.
func writeLocalFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// uploadFolderLive uploads localPath into folderID (retrying the portal's
// transient post-create 500) and returns the file entry.
func uploadFolderLive(t *testing.T, ctx context.Context, c *Client, folderID, localPath string) *FileEntry {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var (
		e   *FileEntry
		err error
	)
	for {
		e, err = c.UploadToFolder(ctx, folderID, localPath)
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("UploadToFolder %s: %v", folderID, err)
	}
	if e == nil || e.ID == nil {
		t.Fatalf("UploadToFolder %s: no file entry (%+v)", folderID, e)
	}
	return e
}

// liveProjectIndex rebuilds the project folder/file index the same way
// DedupeProject does, for direct mergeProjectRootForDedupe assertions.
func liveProjectIndex(t *testing.T, ctx context.Context, c *Client, projectID, rootID string) ([]*FolderEntry, map[string][]*FileEntry, []*FileEntry) {
	t.Helper()
	pf := getProjectFilesEventually(t, ctx, c, projectID)
	rootFiles := folderFilesEventually(t, ctx, c, rootID)
	folders := make([]*FolderEntry, 0, len(pf.Folders)+1)
	byFolder := make(map[string][]*FileEntry, len(pf.Folders)+1)
	for _, folder := range pf.Folders {
		if folder == nil || folder.ID == nil {
			continue
		}
		fid := folder.ID.String()
		if fid == rootID {
			byFolder[fid] = rootFiles
		} else {
			byFolder[fid] = folderFilesEventually(t, ctx, c, fid)
		}
		folders = append(folders, folder)
	}
	return folders, byFolder, rootFiles
}

// projectFolderEventually resolves the project Documents root id, retrying on
// a transient portal answer.
func projectFolderEventually(t *testing.T, ctx context.Context, c *Client, projectID string) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var (
		root string
		err  error
	)
	for {
		root, err = c.projectFolderID(ctx, projectID)
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("projectFolderID: %v", err)
	}
	if root == "" {
		t.Fatal("projectFolderID returned empty id")
	}
	return root
}

// getProjectFilesEventually lists a project's files/folders, retrying on a
// transient portal answer.
func getProjectFilesEventually(t *testing.T, ctx context.Context, c *Client, projectID string) *ProjectFilesResponse {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var last error
	for {
		pf, err := c.GetProjectFiles(ctx, projectID)
		if err == nil {
			return pf
		}
		last = err
		if time.Now().After(deadline) {
			t.Fatalf("GetProjectFiles %s: %v", projectID, last)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// folderFilesEventually lists a folder, retrying while the portal answers
// transiently (a freshly created folder can 500 until its parent map settles).
func folderFilesEventually(t *testing.T, ctx context.Context, c *Client, folderID string) []*FileEntry {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var last error
	for {
		files, err := c.FolderFiles(ctx, folderID)
		if err == nil {
			return files
		}
		last = err
		if time.Now().After(deadline) {
			t.Fatalf("FolderFiles %s: %v", folderID, last)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// dedupeProjectEventually runs a project dedup scan, retrying the whole scan on
// a transient portal error. It is used for dry-runs only (apply must stay a
// single deliberate call).
func dedupeProjectEventually(t *testing.T, ctx context.Context, c *Client, projectID string, opts DedupOptions, apply bool) ([]DedupGroup, []int, error) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var (
		groups  []DedupGroup
		deleted []int
		err     error
	)
	for {
		groups, deleted, err = c.DedupeProject(ctx, projectID, opts, apply)
		if err == nil || time.Now().After(deadline) {
			return groups, deleted, err
		}
		time.Sleep(time.Second)
	}
}

// dedupWaitCount polls folderID until want stem|ext copies are visible.
func dedupWaitCount(t *testing.T, ctx context.Context, c *Client, folderID, stem, ext string, want int, d time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(d)
	got := -1
	for {
		if files, err := c.FolderFiles(ctx, folderID); err == nil {
			got = len(FindFilesByDedupKey(files, stem, ext))
			if got == want {
				return got
			}
		}
		if time.Now().After(deadline) {
			return got
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// dedupWaitTotal polls the given folders until the total number of stem|ext
// copies reaches want, returning the last seen file id and count.
func dedupWaitTotal(t *testing.T, ctx context.Context, c *Client, folderIDs []string, stem, ext string, want int, d time.Duration) (int64, int) {
	t.Helper()
	deadline := time.Now().Add(d)
	var survivor int64
	total := -1
	for {
		survivor, total = 0, 0
		for _, fid := range folderIDs {
			files, err := c.FolderFiles(ctx, fid)
			if err != nil {
				continue
			}
			for _, f := range FindFilesByDedupKey(files, stem, ext) {
				total++
				survivor = FileEntryNumericID(f)
			}
		}
		if total == want {
			return survivor, total
		}
		if time.Now().After(deadline) {
			return survivor, total
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func dedupGroupHasID(g DedupGroup, id int64) bool {
	if id == 0 {
		return false
	}
	if FileEntryNumericID(g.Keep) == id {
		return true
	}
	for _, f := range g.Remove {
		if FileEntryNumericID(f) == id {
			return true
		}
	}
	return false
}
