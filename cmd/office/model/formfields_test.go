package model

import "testing"

func TestFormFieldsFromRawMail(t *testing.T) {
	raw := map[string]any{"subject": "Hi", "body": "Hello"}
	f := FormFieldsFromRaw(KindMail, raw)
	if f.Primary != "Hi" || f.Secondary != "Hello" || !f.ReadOnly {
		t.Fatalf("unexpected mail fields: %+v", f)
	}
}

func TestIsDocumentKind(t *testing.T) {
	if !IsDocumentKind(KindFile) {
		t.Fatal("file should be document")
	}
	if !IsDocumentKind(KindMail) {
		t.Fatal("mail should be document preview")
	}
	if IsDocumentKind(KindTask) {
		t.Fatal("task should be form")
	}
}
