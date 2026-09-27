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

func TestPlanUploadReplacementUpdatesConvertedFile(t *testing.T) {
	xlsx := &FileEntry{ID: jsonNum("3799"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx")}
	plan := planUploadReplacement([]*FileEntry{xlsx}, "ES29-extracto", ".xls")
	if plan.UpdateID != "3799" {
		t.Fatalf("UpdateID = %q, want 3799 (update the saved .xlsx in place)", plan.UpdateID)
	}
	if len(plan.DeleteIDs) != 0 {
		t.Fatalf("DeleteIDs = %v, want none", plan.DeleteIDs)
	}
}

func TestPlanUploadReplacementCollapsesDuplicates(t *testing.T) {
	older := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	first := &FileEntry{ID: jsonNum("3799"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx"), Updated: &older}
	second := &FileEntry{ID: jsonNum("3887"), Title: strPtr("ES29-extracto.xlsx"), FileExst: strPtr(".xlsx"), Updated: &newer}
	plan := planUploadReplacement([]*FileEntry{first, second}, "ES29-extracto", ".xls")
	if plan.UpdateID != "3887" {
		t.Fatalf("UpdateID = %q, want the newest duplicate 3887", plan.UpdateID)
	}
	if len(plan.DeleteIDs) != 1 || plan.DeleteIDs[0] != 3799 {
		t.Fatalf("DeleteIDs = %v, want [3799]", plan.DeleteIDs)
	}
}

// TestPlanUploadReplacementRepeatedXLS is the regression for #82: the second
// upload of the same .xls must find the server-converted .xlsx and update it,
// not append a second file.
func TestPlanUploadReplacementRepeatedXLS(t *testing.T) {
	const stem = "ES29-extracto"

	// First upload: nothing in the folder.
	if plan := planUploadReplacement(nil, stem, ".xls"); plan.UpdateID != "" {
		t.Fatalf("first upload plan = %+v, want create", plan)
	}
	// OnlyOffice converts .xls -> .xlsx on upload; second upload must match it.
	saved := &FileEntry{ID: jsonNum("3799"), Title: strPtr(stem + ".xlsx"), FileExst: strPtr(".xlsx")}
	plan := planUploadReplacement([]*FileEntry{saved}, stem, ".xls")
	if plan.UpdateID != "3799" {
		t.Fatalf("second upload plan = %+v, want in-place update of 3799 (no duplicate)", plan)
	}
	if len(plan.DeleteIDs) != 0 {
		t.Fatalf("second upload would delete %v, want none", plan.DeleteIDs)
	}
}
