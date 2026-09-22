package onlyoffice

import "testing"

func TestFileEditorURL(t *testing.T) {
	cases := []struct{ base, id, want string }{
		{"https://office.example.com", "3651", "https://office.example.com/Products/Files/DocEditor.aspx?fileid=3651"},
		{"https://office.example.com/", " 1234 ", "https://office.example.com/Products/Files/DocEditor.aspx?fileid=1234"},
		{"http://localhost:8087", "a/b", "http://localhost:8087/Products/Files/DocEditor.aspx?fileid=a%2Fb"},
	}
	for _, c := range cases {
		if got := FileEditorURL(c.base, c.id); got != c.want {
			t.Fatalf("FileEditorURL(%q,%q) = %q, want %q", c.base, c.id, got, c.want)
		}
	}
}

func TestFolderURL(t *testing.T) {
	got := FolderURL("https://office.example.com/", "495")
	want := "https://office.example.com/Products/Files/Default.aspx#folder=495"
	if got != want {
		t.Fatalf("FolderURL = %q, want %q", got, want)
	}
}
