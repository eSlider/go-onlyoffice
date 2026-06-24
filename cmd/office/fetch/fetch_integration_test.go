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

func TestIntegrationListProjects(t *testing.T) {
	if os.Getenv("ONLYOFFICE_URL") == "" && os.Getenv("ONLYOFFICE_HOST") == "" {
		t.Skip("ONLYOFFICE_URL not set")
	}
	client, err := bootstrap.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	loader := &fetch.Loader{Client: client}
	items, err := loader.List(context.Background(), model.SubjectProjects)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("projects: %d", len(items))
}

func TestIntegrationListMailInbox(t *testing.T) {
	if os.Getenv("ONLYOFFICE_URL") == "" && os.Getenv("ONLYOFFICE_HOST") == "" {
		t.Skip("ONLYOFFICE_URL not set")
	}
	client, err := bootstrap.NewClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	loader := &fetch.Loader{Client: client}
	items, err := loader.List(context.Background(), model.SubjectMailInbox)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mail messages: %d", len(items))
}
