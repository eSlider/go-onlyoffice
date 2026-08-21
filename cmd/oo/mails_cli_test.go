package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteMailAttachment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attach.bin")
	body := []byte("payload")
	if err := writeMailAttachment(path, body); err != nil {
		t.Fatalf("writeMailAttachment: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("body = %q", got)
	}
}

func TestWriteMailAttachmentRequiresPath(t *testing.T) {
	if err := writeMailAttachment("", []byte("x")); err == nil {
		t.Fatal("expected error for empty path")
	}
}
