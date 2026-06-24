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
	title, desc, err := loader.TaskFields(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	restoreTitle, restoreDesc := title, desc
	t.Cleanup(func() {
		_ = loader.UpdateTask(context.Background(), item.ID, restoreTitle, restoreDesc)
	})
	newTitle := title + " (office TUI test)"
	newDesc := desc + "\n\n_edited by office integration test_"
	if err := loader.UpdateTask(ctx, item.ID, newTitle, newDesc); err != nil {
		t.Fatal(err)
	}
	gotTitle, gotDesc, err := loader.TaskFields(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	if gotTitle != newTitle {
		t.Fatalf("title: got %q want %q", gotTitle, newTitle)
	}
	if gotDesc != newDesc {
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
