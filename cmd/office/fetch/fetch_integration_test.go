//go:build integration

package fetch_test

import (
	"context"
	"os"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/internal/bootstrap"
	"github.com/eslider/go-onlyoffice/cmd/office/fetch"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func skipWithoutLiveAPI(t *testing.T) {
	t.Helper()
	if os.Getenv("ONLYOFFICE_URL") == "" && os.Getenv("ONLYOFFICE_HOST") == "" {
		t.Skip("ONLYOFFICE_URL not set")
	}
	if os.Getenv("ONLYOFFICE_USER") == "" && os.Getenv("ONLYOFFICE_NAME") == "" {
		t.Skip("ONLYOFFICE_USER not set")
	}
	if os.Getenv("ONLYOFFICE_PASS") == "" && os.Getenv("ONLYOFFICE_PASSWORD") == "" {
		t.Skip("ONLYOFFICE_PASS not set")
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

func TestIntegrationListAllSubjects(t *testing.T) {
	loader, ctx := liveLoader(t)

	cases := []struct {
		subject model.Subject
		skip    string
	}{
		{model.SubjectProjects, ""},
		{model.SubjectTasks, ""},
		{model.SubjectCalendars, ""},
		{model.SubjectEvents, ""},
		{model.SubjectContacts, ""},
		{model.SubjectPersons, ""},
		{model.SubjectCompanies, ""},
		{model.SubjectOpportunities, ""},
		{model.SubjectCases, ""},
		{model.SubjectCRMTasks, ""},
		{model.SubjectMailInbox, ""},
		{model.SubjectMailSent, ""},
		{model.SubjectMailDrafts, ""},
		{model.SubjectMailTrash, ""},
		{model.SubjectMailSpam, ""},
		{model.SubjectUsers, ""},
		{model.SubjectProjectFiles, "ONLYOFFICE_PROJECT_ID not set"},
	}

	for _, tc := range cases {
		t.Run(string(tc.subject), func(t *testing.T) {
			if tc.subject == model.SubjectProjectFiles {
				if os.Getenv("ONLYOFFICE_PROJECT_ID") == "" {
					t.Skip(tc.skip)
				}
			}
			items, err := loader.List(ctx, tc.subject)
			if err != nil {
				t.Fatalf("List(%s): %v", tc.subject, err)
			}
			t.Logf("%s: %d items", tc.subject, len(items))
			for i, it := range items {
				if it.ID == "" {
					t.Errorf("item[%d] missing ID: %+v", i, it)
				}
				if it.Title == "" {
					t.Errorf("item[%d] missing Title: id=%s kind=%s", i, it.ID, it.Kind)
				}
				if it.Kind == "" {
					t.Errorf("item[%d] missing Kind: id=%s", i, it.ID)
				}
			}
		})
	}
}

func TestIntegrationListProjectsMapsRealFields(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectProjects)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no projects on instance")
	}
	it := items[0]
	if it.Raw == nil {
		t.Fatal("expected Raw payload from API")
	}
	detail, err := loader.Detail(ctx, it)
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if detail == nil {
		t.Fatal("nil detail")
	}
	t.Logf("project id=%s title=%q keys=%d", it.ID, it.Title, len(detail))
}

func TestIntegrationMailInboxDetail(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectMailInbox)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("inbox empty")
	}
	detail, err := loader.Detail(ctx, items[0])
	if err != nil {
		t.Fatalf("GetMailMessage: %v", err)
	}
	if detail["subject"] == nil {
		t.Fatalf("message missing subject: %+v", detail)
	}
}

func TestIntegrationEventsDateRange(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.SubjectEvents)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("events in next 7 days: %d", len(items))
}
