package onlyoffice

import (
	"testing"
	"time"
)

func TestEquivalentUploadExt(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{".xls", ".xlsx", true},
		{".XLS", ".xlsx", true},
		{"xls", "xlsx", true},
		{".doc", ".docx", true},
		{".ppt", ".pptx", true},
		{".pdf", ".pdf", true},
		{"", "", true},
		{".xls", ".docx", false},
		{".xlsx", "", false},
		{".csv", ".xlsx", false},
	}
	for _, tc := range tests {
		if got := EquivalentUploadExt(tc.a, tc.b); got != tc.want {
			t.Errorf("EquivalentUploadExt(%q,%q)=%v want %v", tc.a, tc.b, got, tc.want)
		}
		if got := EquivalentUploadExt(tc.b, tc.a); got != tc.want {
			t.Errorf("EquivalentUploadExt(%q,%q)=%v want %v (symmetric)", tc.b, tc.a, got, tc.want)
		}
	}
}

func TestFindFilesByStemExtMatchesConvertedXLS(t *testing.T) {
	xlsx := &FileEntry{ID: jsonNum("3799"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx")}
	pdf := &FileEntry{ID: jsonNum("5"), Title: strPtr("ES29-extracto.pdf"), FileExst: strPtr(".pdf")}
	other := &FileEntry{ID: jsonNum("3887"), Title: strPtr("ES87-extracto.xlsx"), FileExst: strPtr(".xlsx")}
	files := []*FileEntry{xlsx, pdf, other}

	got := FindFilesByStemExt(files, "ES29-extracto", ".xls")
	if len(got) != 1 || got[0] != xlsx {
		t.Fatalf("converted .xls match = %+v, want the saved .xlsx only", got)
	}
}

func TestFindFilesByStemExtKeepsExactMatch(t *testing.T) {
	xlsx := &FileEntry{ID: jsonNum("3799"), Title: strPtr("foo.xlsx"), FileExst: strPtr(".xlsx")}
	pdf := &FileEntry{ID: jsonNum("5"), Title: strPtr("foo.pdf"), FileExst: strPtr(".pdf")}
	files := []*FileEntry{xlsx, pdf}

	if got := FindFilesByStemExt(files, "foo", ".xlsx"); len(got) != 1 || got[0] != xlsx {
		t.Fatalf("exact .xlsx match = %+v, want only xlsx", got)
	}
	if got := FindFilesByStemExt(files, "foo", ".pdf"); len(got) != 1 || got[0] != pdf {
		t.Fatalf("exact .pdf match = %+v, want only pdf", got)
	}
	if got := FindFilesByStemExt(files, "foo", ".docx"); len(got) != 0 {
		t.Fatalf("unrelated ext matched %+v, want none", got)
	}
}

func TestFindFilesByStemExtEmptyStem(t *testing.T) {
	f := &FileEntry{ID: jsonNum("1"), Title: strPtr("foo.xlsx"), FileExst: strPtr(".xlsx")}
	if got := FindFilesByStemExt([]*FileEntry{f}, "", ".xlsx"); len(got) != 0 {
		t.Fatalf("empty stem matched %+v, want none", got)
	}
}

func TestPlanUploadReplacementFreshUpload(t *testing.T) {
	plan := planUploadReplacement(nil, "ES29-extracto", ".xls")
	if plan.UpdateID != "" || len(plan.DeleteIDs) != 0 {
		t.Fatalf("empty folder plan = %+v, want a fresh upload", plan)
	}
}

// TestPlanUploadReplacementUpdatesExactExt covers the in-place path: only a
// stored file with the same extension is updated via UpdateFile. PDF over PDF
// and OOXML over OOXML keep the id and rewrite the body.
func TestPlanUploadReplacementUpdatesExactExt(t *testing.T) {
	tests := []struct {
		name string
		file *FileEntry
		stem string
		ext  string
		id   string
	}{
		{
			name: "xlsx over xlsx",
			file: &FileEntry{ID: jsonNum("3799"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx")},
			stem: "ES29-extracto", ext: ".xlsx", id: "3799",
		},
		{
			name: "pdf over pdf",
			file: &FileEntry{ID: jsonNum("42"), Title: strPtr("extracto.pdf"), FileExst: strPtr(".pdf")},
			stem: "extracto", ext: ".pdf", id: "42",
		},
		{
			name: "legacy xls over xls",
			file: &FileEntry{ID: jsonNum("11"), Title: strPtr("legacy.xls"), FileExst: strPtr(".xls")},
			stem: "legacy", ext: ".xls", id: "11",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan := planUploadReplacement([]*FileEntry{tc.file}, tc.stem, tc.ext)
			if plan.UpdateID != tc.id {
				t.Fatalf("UpdateID = %q, want %q (same ext updates in place)", plan.UpdateID, tc.id)
			}
			if len(plan.DeleteIDs) != 0 {
				t.Fatalf("DeleteIDs = %v, want none", plan.DeleteIDs)
			}
		})
	}
}

// TestPlanUploadReplacementConvertedExtDeletesThenUploads is the regression for
// #84: UpdateFile does not re-run the server-side legacy→OOXML conversion, so a
// .xls upload must not overwrite a stored .xlsx in place (raw OLE2 under an
// .xlsx name). The stale counterpart is deleted and the file uploaded afresh.
func TestPlanUploadReplacementConvertedExtDeletesThenUploads(t *testing.T) {
	xlsx := &FileEntry{ID: jsonNum("3799"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx")}
	plan := planUploadReplacement([]*FileEntry{xlsx}, "ES29-extracto", ".xls")
	if plan.UpdateID != "" {
		t.Fatalf("UpdateID = %q, want empty (do not update across conversion)", plan.UpdateID)
	}
	if len(plan.DeleteIDs) != 1 || plan.DeleteIDs[0] != 3799 {
		t.Fatalf("DeleteIDs = %v, want [3799] (delete the stale .xlsx before upload)", plan.DeleteIDs)
	}
}

func TestPlanUploadReplacementCollapsesDuplicates(t *testing.T) {
	older := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	first := &FileEntry{ID: jsonNum("3799"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx"), Updated: &older}
	second := &FileEntry{ID: jsonNum("3887"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx"), Updated: &newer}
	plan := planUploadReplacement([]*FileEntry{first, second}, "ES29-extracto", ".xlsx")
	if plan.UpdateID != "3887" {
		t.Fatalf("UpdateID = %q, want the newest duplicate 3887", plan.UpdateID)
	}
	if len(plan.DeleteIDs) != 1 || plan.DeleteIDs[0] != 3799 {
		t.Fatalf("DeleteIDs = %v, want [3799]", plan.DeleteIDs)
	}
}

// TestPlanUploadReplacementDeletesConvertedDuplicates: with a converted match
// every stale copy is deleted (there is no keeper — the fresh upload replaces
// them all).
func TestPlanUploadReplacementDeletesConvertedDuplicates(t *testing.T) {
	first := &FileEntry{ID: jsonNum("3799"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx")}
	second := &FileEntry{ID: jsonNum("3887"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx")}
	plan := planUploadReplacement([]*FileEntry{first, second}, "ES29-extracto", ".xls")
	if plan.UpdateID != "" {
		t.Fatalf("UpdateID = %q, want empty", plan.UpdateID)
	}
	if len(plan.DeleteIDs) != 2 {
		t.Fatalf("DeleteIDs = %v, want both stale .xlsx ids", plan.DeleteIDs)
	}
}

// TestPlanUploadReplacementRepeatedXLS is the regression for #84: with the
// server-converted .xlsx already present, the second .xls upload deletes it and
// uploads anew so OnlyOffice converts again — UpdateFile would corrupt it.
func TestPlanUploadReplacementRepeatedXLS(t *testing.T) {
	const stem = "ES29-extracto"

	// First upload: nothing in the folder.
	if plan := planUploadReplacement(nil, stem, ".xls"); plan.UpdateID != "" || len(plan.DeleteIDs) != 0 {
		t.Fatalf("first upload plan = %+v, want create", plan)
	}
	// OnlyOffice converts .xls -> .xlsx on upload; the second upload must
	// delete it and upload fresh, never UpdateFile it.
	saved := &FileEntry{ID: jsonNum("3799"), Title: strPtr(stem + ".xlsx"), FileExst: strPtr(".xlsx")}
	plan := planUploadReplacement([]*FileEntry{saved}, stem, ".xls")
	if plan.UpdateID != "" {
		t.Fatalf("second upload plan = %+v, want delete+upload (not UpdateFile)", plan)
	}
	if len(plan.DeleteIDs) != 1 || plan.DeleteIDs[0] != 3799 {
		t.Fatalf("second upload DeleteIDs = %v, want [3799]", plan.DeleteIDs)
	}
}
