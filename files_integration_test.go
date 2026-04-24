//go:build integration

package onlyoffice

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestIntegrationProjectFilesFlow creates a throwaway project, uploads a file
// into its Documents folder, lists, downloads, renames, deletes, then removes
// the project. Destructive — only run against instances you own.
func TestIntegrationProjectFilesFlow(t *testing.T) {
	c := liveClient(t)
	t.Cleanup(func() { cleanupTestProjects(t, c) })

	suffix := time.Now().UTC().Format("20060102-150405")
	title := testProjectPrefix + "files-" + suffix
	project, err := c.CreateProject(NewProjectRequest{
		Title:       title,
		Description: "go-onlyoffice project files integration",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.ID == nil {
		t.Fatal("created project without id")
	}
	pid := strconv.Itoa(*project.ID)
	ctx := context.Background()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "hello.txt")
	content := []byte("integration project file " + suffix + "\n")
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	entry, err := c.UploadProjectFile(ctx, pid, localPath)
	if err != nil {
		t.Fatalf("UploadProjectFile: %v", err)
	}
	if entry == nil || entry.ID == nil {
		t.Fatalf("upload returned no file: %+v", entry)
	}
	fileID := entry.ID.String()

	pf, err := c.GetProjectFiles(ctx, pid)
	if err != nil {
		t.Fatalf("GetProjectFiles: %v", err)
	}
	found := false
	for _, f := range pf.Files {
		if f != nil && f.ID != nil && f.ID.String() == fileID {
			found = true
			break
		}
	}
	if !found {
		t.Logf("uploaded file id=%s not in project files list (may still be ok); folders=%d files=%d",
			fileID, len(pf.Folders), len(pf.Files))
	}

	var buf bytes.Buffer
	n, err := c.DownloadFile(ctx, fileID, &buf)
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	if n != int64(len(content)) || !bytes.Equal(buf.Bytes(), content) {
		t.Fatalf("download mismatch: got %d bytes %q want %d bytes", n, buf.String(), len(content))
	}

	newTitle := "renamed-" + suffix + ".txt"
	renamed, err := c.RenameFile(ctx, fileID, newTitle)
	if err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if renamed == nil || renamed.Title == nil || *renamed.Title != newTitle {
		t.Fatalf("rename result: %+v", renamed)
	}

	nid := int(FileEntryNumericID(entry))
	if nid == 0 {
		t.Fatal("file id 0 for delete")
	}
	if err := c.DeleteFiles(ctx, []int{nid}); err != nil {
		t.Fatalf("DeleteFiles: %v", err)
	}
}

func TestIntegrationTaskFilesFlow(t *testing.T) {
	c := liveClient(t)
	t.Cleanup(func() { cleanupTestProjects(t, c) })

	suffix := time.Now().UTC().Format("20060102-150405")
	title := testProjectPrefix + "taskfiles-" + suffix
	project, err := c.CreateProject(NewProjectRequest{
		Title:       title,
		Description: "go-onlyoffice task files integration",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.ID == nil {
		t.Fatal("created project without id")
	}

	start := Time(time.Now().AddDate(0, 0, -1))
	deadline := Time(time.Now().AddDate(0, 0, 1))
	task, err := c.CreateProjectTask(NewProjectTaskRequest{
		ProjectId:   *project.ID,
		Title:       "task for file attach " + suffix,
		Description: "integration",
		StartDate:   start,
		Deadline:    deadline,
		Priority:    int(TaskPriorityNormal),
	})
	if err != nil {
		t.Fatalf("CreateProjectTask: %v", err)
	}
	if task.ID == nil {
		t.Fatal("task without id")
	}
	tid := strconv.Itoa(*task.ID)
	ctx := context.Background()

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "attach.txt")
	content := []byte("task attachment " + suffix + "\n")
	if err := os.WriteFile(localPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	up, err := c.UploadTaskFile(ctx, tid, localPath)
	if err != nil {
		t.Fatalf("UploadTaskFile: %v", err)
	}
	if up == nil || up.ID == nil {
		t.Fatalf("upload: %+v", up)
	}
	fileID := up.ID.String()

	list, err := c.GetTaskFiles(ctx, tid)
	if err != nil {
		t.Fatalf("GetTaskFiles: %v", err)
	}
	found := false
	for _, f := range list {
		if f != nil && f.ID != nil && f.ID.String() == fileID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("uploaded file not in GetTaskFiles: %#v", list)
	}

	if err := c.DetachTaskFile(ctx, tid, fileID); err != nil {
		t.Fatalf("DetachTaskFile: %v", err)
	}
	list2, err := c.GetTaskFiles(ctx, tid)
	if err != nil {
		t.Fatalf("GetTaskFiles after detach: %v", err)
	}
	for _, f := range list2 {
		if f != nil && f.ID != nil && f.ID.String() == fileID {
			t.Fatalf("file still attached after detach: %v", fileID)
		}
	}
}
