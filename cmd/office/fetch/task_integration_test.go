//go:build integration

package fetch_test

import (
	"context"
	"os"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestIntegrationUpdateTaskTitleDescription(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.ListSpec{Subject: model.SubjectTasks})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no tasks")
	}
	item := items[0]
	fields, err := loader.DetailForm(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	restore := fields
	t.Cleanup(func() {
		_ = loader.UpdateTask(context.Background(), item.ID, restore)
	})
	fields.Primary = fields.Primary + " (office TUI test)"
	fields.Secondary = fields.Secondary + "\n\n_edited by office integration test_"
	if err := loader.UpdateTask(ctx, item.ID, fields); err != nil {
		t.Fatal(err)
	}
	got, err := loader.DetailForm(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	if got.Primary != fields.Primary {
		t.Fatalf("title: got %q want %q", got.Primary, fields.Primary)
	}
	if got.Secondary != fields.Secondary {
		t.Fatalf("description mismatch")
	}
}

func TestIntegrationTaskFieldsFromLiveAPI(t *testing.T) {
	if os.Getenv("ONLYOFFICE_URL") == "" && os.Getenv("ONLYOFFICE_HOST") == "" {
		t.Skip("ONLYOFFICE_URL not set")
	}
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.ListSpec{Subject: model.SubjectTasks})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no tasks")
	}
	title, desc, err := loader.TaskFields(ctx, items[0])
	if err != nil {
		t.Fatal(err)
	}
	if title == "" {
		t.Fatal("empty title")
	}
	_ = desc
}
