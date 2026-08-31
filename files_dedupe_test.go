package onlyoffice

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFileDedupKey(t *testing.T) {
	title := "OO-HONDA-7-INDEX.docx"
	exst := ".docx"
	f := &FileEntry{Title: &title, FileExst: &exst}
	if got := FileDedupKey(f); got != "OO-HONDA-7-INDEX|docx" {
		t.Fatalf("got %q", got)
	}
}

func TestFindFilesByDedupKey(t *testing.T) {
	a := &FileEntry{Title: strPtr("foo.docx"), FileExst: strPtr(".docx")}
	b := &FileEntry{Title: strPtr("foo.md"), FileExst: strPtr(".md")}
	files := []*FileEntry{a, b}
	got := FindFilesByDedupKey(files, "foo", ".docx")
	if len(got) != 1 || got[0] != a {
		t.Fatalf("got %+v", got)
	}
}

func TestFindWithinFolderDuplicates(t *testing.T) {
	t1 := time.Date(2026, 8, 27, 16, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	old := &FileEntry{ID: jsonNum("1"), Title: strPtr("idx.docx"), FileExst: strPtr(".docx"), Updated: &t1}
	new := &FileEntry{ID: jsonNum("2"), Title: strPtr("idx.docx"), FileExst: strPtr(".docx"), Updated: &t2}
	indexed := []ProjectFolderFile{
		{FolderID: "490", FolderTitle: "00-Index", File: old},
		{FolderID: "490", FolderTitle: "00-Index", File: new},
	}
	groups := findWithinFolderDuplicates(indexed)
	if len(groups) != 1 {
		t.Fatalf("groups=%d", len(groups))
	}
	if FileEntryNumericID(groups[0].Keep) != 2 {
		t.Fatalf("keep id=%d", FileEntryNumericID(groups[0].Keep))
	}
	if len(groups[0].Remove) != 1 || FileEntryNumericID(groups[0].Remove[0]) != 1 {
		t.Fatalf("remove=%v", groups[0].Remove)
	}
}

func TestCrossFolderPrefersNonTrash(t *testing.T) {
	t1 := time.Date(2026, 8, 27, 18, 0, 0, 0, time.UTC)
	t2 := t1.Add(-time.Hour)
	trash := &FileEntry{ID: jsonNum("10"), Title: strPtr("INDEX.md"), FileExst: strPtr(".md"), Updated: &t1}
	good := &FileEntry{ID: jsonNum("20"), Title: strPtr("INDEX.md"), FileExst: strPtr(".md"), Updated: &t2}
	indexed := []ProjectFolderFile{
		{FolderID: "493", FolderTitle: "_trash-md", File: trash},
		{FolderID: "492", FolderTitle: "OCR", File: good},
	}
	groups := findCrossFolderDuplicates(indexed)
	if len(groups) != 1 {
		t.Fatalf("groups=%d", len(groups))
	}
	if FileEntryNumericID(groups[0].Keep) != 20 {
		t.Fatalf("keep id=%d", FileEntryNumericID(groups[0].Keep))
	}
}

func TestMergeProjectRootForDedupe(t *testing.T) {
	old := &FileEntry{ID: jsonNum("1"), Title: strPtr("a.docx"), FileExst: strPtr(".docx")}
	newer := &FileEntry{ID: jsonNum("2"), Title: strPtr("a.docx"), FileExst: strPtr(".docx")}
	rootFiles := []*FileEntry{old, newer}
	folders, byFolder := mergeProjectRootForDedupe("489", nil, nil, rootFiles)
	if len(folders) != 1 || folders[0].ID.String() != "489" {
		t.Fatalf("folders=%+v", folders)
	}
	if len(byFolder["489"]) != 2 {
		t.Fatalf("root files=%d", len(byFolder["489"]))
	}
	groups := findWithinFolderDuplicates([]ProjectFolderFile{
		{FolderID: "489", FolderTitle: "(project root)", File: old},
		{FolderID: "489", FolderTitle: "(project root)", File: newer},
	})
	if len(groups) != 1 {
		t.Fatalf("groups=%d", len(groups))
	}
}

func TestIsTrashFolderTitle(t *testing.T) {
	if !IsTrashFolderTitle("_trash-md") {
		t.Fatal("expected trash")
	}
	if IsTrashFolderTitle("00-Index") {
		t.Fatal("expected not trash")
	}
}

func strPtr(s string) *string { return &s }

func jsonNum(s string) *json.Number {
	n := json.Number(s)
	return &n
}
