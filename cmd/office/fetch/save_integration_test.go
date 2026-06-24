//go:build integration

package fetch_test

import (
	"testing"
	"time"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestIntegrationSaveProject(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.ListSpec{Subject: model.SubjectProjects})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no projects")
	}
	item := items[0]
	fields, err := loader.DetailForm(ctx, item)
	if err != nil {
		t.Fatalf("DetailForm: %v", err)
	}
	if fields.ResponsibleID == "" {
		t.Skip("project has no responsible id on this instance")
	}
	marker := " office-save-test " + time.Now().UTC().Format(time.RFC3339)
	fields.Primary = fields.Primary + marker
	if err := loader.SaveItem(ctx, item, fields); err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
	after, err := loader.DetailForm(ctx, item)
	if err != nil {
		t.Fatalf("DetailForm after save: %v", err)
	}
	if after.Primary != fields.Primary {
		t.Fatalf("title not saved: got %q want %q", after.Primary, fields.Primary)
	}
	fields.Primary = stringsTrimSuffixMarker(after.Primary, marker)
	if err := loader.SaveItem(ctx, item, fields); err != nil {
		t.Logf("cleanup save: %v", err)
	}
}

func stringsTrimSuffixMarker(s, marker string) string {
	if len(s) >= len(marker) && s[len(s)-len(marker):] == marker {
		return s[:len(s)-len(marker)]
	}
	return s
}
