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

	cases := []model.ListSpec{
		{Subject: model.SubjectProjects},
		{Subject: model.SubjectTasks},
		{Subject: model.SubjectCalendars},
		{Subject: model.SubjectEvents},
		{Subject: model.SubjectContacts},
		{Subject: model.SubjectPersons},
		{Subject: model.SubjectCompanies},
		{Subject: model.SubjectOpportunities},
		{Subject: model.SubjectCases},
		{Subject: model.SubjectCRMTasks},
		{Subject: model.SubjectMailInbox},
		{Subject: model.SubjectUsers},
	}

	for _, spec := range cases {
		t.Run(string(spec.Subject), func(t *testing.T) {
			items, err := loader.List(ctx, spec)
			if err != nil {
				t.Fatalf("List(%s): %v", spec.Subject, err)
			}
			t.Logf("%s: %d items", spec.Subject, len(items))
			for i, it := range items {
				if it.ID == "" {
					t.Errorf("item[%d] missing ID", i)
				}
				if it.Title == "" {
					t.Errorf("item[%d] missing Title", i)
				}
			}
		})
	}
}

func TestIntegrationListProjectsMapsRealFields(t *testing.T) {
	loader, ctx := liveLoader(t)
	spec := model.ListSpec{Subject: model.SubjectProjects}
	items, err := loader.List(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Skip("no projects on instance")
	}
	detail, err := loader.Detail(ctx, items[0])
	if err != nil {
		t.Fatalf("Detail: %v", err)
	}
	if detail == nil {
		t.Fatal("nil detail")
	}
}

func TestIntegrationMailInboxDetail(t *testing.T) {
	loader, ctx := liveLoader(t)
	items, err := loader.List(ctx, model.ListSpec{Subject: model.SubjectMailInbox})
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
