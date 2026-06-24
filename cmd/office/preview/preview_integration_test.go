//go:build integration

package preview_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/internal/bootstrap"
	"github.com/eslider/go-onlyoffice/cmd/office/fetch"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

func skipWithoutLiveAPI(t *testing.T) {
	t.Helper()
	if os.Getenv("ONLYOFFICE_URL") == "" && os.Getenv("ONLYOFFICE_HOST") == "" {
		t.Skip("ONLYOFFICE_URL not set")
	}
}

func liveLoader(t *testing.T) (*fetch.Loader, context.Context) {
	t.Helper()
	skipWithoutLiveAPI(t)
	client, err := bootstrap.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return &fetch.Loader{Client: client}, context.Background()
}

func TestIntegrationPreviewProjectFromAPI(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectProjects)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no projects")
	}
	raw, err := loader.Detail(ctx, items[0])
	if err != nil {
		t.Fatal(err)
	}
	md := preview.EntityMarkdown(string(model.KindProject), raw)
	if md == "" {
		t.Fatal("empty markdown")
	}
	if !strings.Contains(md, items[0].Title) {
		t.Fatalf("markdown missing project title %q:\n%s", items[0].Title, md)
	}
	rendered, err := preview.RenderMarkdown(md, 80)
	if err != nil {
		t.Fatal(err)
	}
	if rendered == "" {
		t.Fatal("glamour produced empty output")
	}
}

func TestIntegrationPreviewContactFromAPI(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectContacts)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no contacts")
	}
	raw, err := loader.Detail(ctx, items[0])
	if err != nil {
		t.Fatal(err)
	}
	md := preview.ContactMarkdown(raw)
	if md == "" || !strings.Contains(md, "#") {
		t.Fatalf("unexpected contact markdown: %q", md)
	}
}

func TestIntegrationPreviewOpportunityFromAPI(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectOpportunities)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no opportunities")
	}
	raw, err := loader.Detail(ctx, items[0])
	if err != nil {
		t.Fatal(err)
	}
	md := preview.OpportunityMarkdown(raw)
	if !strings.Contains(md, items[0].Title) {
		t.Fatalf("markdown missing deal title:\n%s", md)
	}
}

func TestIntegrationPreviewMailFromAPI(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectMailInbox)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("inbox empty")
	}
	raw, err := loader.Detail(ctx, items[0])
	if err != nil {
		t.Fatal(err)
	}
	md := preview.MailMarkdown(raw)
	if !strings.Contains(md, items[0].Title) {
		t.Fatalf("markdown missing subject %q:\n%s", items[0].Title, md)
	}
}

func TestIntegrationPreviewEventFromAPI(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectEvents)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no events in range")
	}
	md := preview.EventMarkdown(items[0].Raw)
	if !strings.Contains(md, items[0].Title) {
		t.Fatalf("markdown missing event title:\n%s", md)
	}
}

func TestIntegrationPreviewTaskFromAPI(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectTasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no tasks")
	}
	raw, err := loader.Detail(ctx, items[0])
	if err != nil {
		t.Fatal(err)
	}
	md := preview.TaskMarkdown(raw)
	if !strings.Contains(md, items[0].Title) {
		t.Fatalf("markdown missing task title:\n%s", md)
	}
}

func TestIntegrationPreviewCSVFromDownloadedFile(t *testing.T) {
	skipWithoutLiveAPI(t)
	if os.Getenv("ONLYOFFICE_PROJECT_ID") == "" {
		t.Skip("ONLYOFFICE_PROJECT_ID not set")
	}
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectProjectFiles)
	if err != nil {
		t.Fatal(err)
	}
	var csvItem *model.Item
	for i := range items {
		if strings.HasSuffix(strings.ToLower(items[i].Title), ".csv") {
			csvItem = &items[i]
			break
		}
	}
	if csvItem == nil {
		t.Skip("no csv file in project documents")
	}
	var buf strings.Builder
	n, err := loader.Client.DownloadFile(ctx, csvItem.ID, &buf)
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	if n == 0 {
		t.Fatal("empty download")
	}
	md, err := preview.CSVToMarkdownTable([]byte(buf.String()))
	if err != nil {
		t.Fatalf("CSVToMarkdownTable: %v", err)
	}
	if !strings.Contains(md, "|") {
		t.Fatalf("expected pipe table: %q", md)
	}
}
