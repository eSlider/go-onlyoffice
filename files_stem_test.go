package onlyoffice

import "testing"

func TestNormalizeUploadStem(t *testing.T) {
	tests := []struct {
		title, exst, want string
	}{
		{"OO-HONDA-7-INDEX.docx", ".docx", "OO-HONDA-7-INDEX"},
		{"README.txt", ".txt", "README"},
		{"car-docs-print.docx", ".docx", "car-docs-print"},
		{"plain.", "", "plain"},
		{"foo", ".docx", "foo"},
	}
	for _, tc := range tests {
		if got := NormalizeUploadStem(tc.title, tc.exst); got != tc.want {
			t.Fatalf("NormalizeUploadStem(%q,%q)=%q want %q", tc.title, tc.exst, got, tc.want)
		}
	}
}

func TestFileEntryStem(t *testing.T) {
	title := "00-INDEX.docx"
	exst := ".docx"
	f := &FileEntry{Title: &title, FileExst: &exst}
	if got := FileEntryStem(f); got != "00-INDEX" {
		t.Fatalf("got %q", got)
	}
}

func TestUploadStemFromLocal(t *testing.T) {
	if got := UploadStemFromLocal("/tmp/OO-HONDA-7-INDEX.docx"); got != "OO-HONDA-7-INDEX" {
		t.Fatalf("got %q", got)
	}
}
