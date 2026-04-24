package onlyoffice

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDecodeProjectFilesEnvelope(t *testing.T) {
	raw, err := os.ReadFile("testdata/project_files_response.json")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := responseField(json.RawMessage(raw), "response")
	if err != nil {
		t.Fatal(err)
	}
	var pf ProjectFilesResponse
	if err := json.Unmarshal(resp, &pf); err != nil {
		t.Fatal(err)
	}
	if len(pf.Folders) != 1 || pf.Folders[0].Title == nil || *pf.Folders[0].Title != "Subfolder" {
		t.Fatalf("folders: %+v", pf.Folders)
	}
	if len(pf.Files) != 1 || pf.Files[0].Title == nil || *pf.Files[0].Title != "readme.txt" {
		t.Fatalf("files: %+v", pf.Files)
	}
	if pf.Files[0].ID == nil || pf.Files[0].ID.String() != "100" {
		t.Fatalf("file id: %v", pf.Files[0].ID)
	}
}

func TestDecodeTaskFilesEnvelope(t *testing.T) {
	raw, err := os.ReadFile("testdata/task_files_response.json")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := responseField(json.RawMessage(raw), "response")
	if err != nil {
		t.Fatal(err)
	}
	var list []*FileEntry
	if err := json.Unmarshal(resp, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Title == nil || *list[0].Title != "attach.pdf" {
		t.Fatalf("list: %+v", list)
	}
}

func TestProjectIDFromTaskMap(t *testing.T) {
	m := map[string]any{
		"projectOwner": map[string]any{"id": float64(33)},
	}
	if got := projectIDFromTaskMap(m); got != "33" {
		t.Fatalf("got %q", got)
	}
	if projectIDFromTaskMap(nil) != "" {
		t.Fatal("expected empty")
	}
}

func TestSafeLocalFileName(t *testing.T) {
	if got := SafeLocalFileName("  foo/bar.txt  "); got != "bar.txt" {
		t.Fatalf("got %q", got)
	}
	if got := SafeLocalFileName(""); got != "download" {
		t.Fatalf("got %q", got)
	}
}
