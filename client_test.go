//go:build integration

package onlyoffice

// Integration tests — hit a live OnlyOffice Workspace instance. Credentials
// come from ONLYOFFICE_URL / ONLYOFFICE_USER / ONLYOFFICE_PASS (aliases
// _HOST / _NAME / _PASSWORD also accepted). Without credentials every test
// here skips.
//
// Run with:
//
//	go test -tags=integration ./...
//
// These tests are destructive on the target instance. They create projects
// with titles prefixed "go-onlyoffice-test-" and clean up afterwards. Do not
// run against an instance you don't own.

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

func skipWithoutCredentials(t *testing.T) Credentials {
	t.Helper()
	c := GetEnvironmentCredentials()
	if c.Url == "" || c.User == "" || c.Password == "" {
		t.Skip("ONLYOFFICE_URL/USER/PASS not set — skipping integration test")
	}
	return c
}

func liveClient(t *testing.T) *Client {
	t.Helper()
	c := NewClient(skipWithoutCredentials(t))
	c.SetDefaults(GetEnvironmentDefaults())
	return c
}

const testProjectPrefix = "go-onlyoffice-test-"

func cleanupTestProjects(t *testing.T, c *Client) {
	t.Helper()
	projects, err := c.GetProjects()
	if err != nil {
		t.Logf("cleanup: GetProjects: %v", err)
		return
	}
	for _, p := range projects {
		if p.Title != nil && strings.HasPrefix(*p.Title, testProjectPrefix) {
			if _, err := c.DeleteProject(*p.ID); err != nil {
				t.Logf("cleanup: DeleteProject %d: %v", *p.ID, err)
			}
		}
	}
}

func TestIntegrationAuthenticateContext(t *testing.T) {
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.AuthenticateContext(ctx); err != nil {
		t.Fatalf("AuthenticateContext: %v", err)
	}
	if c.token == nil || c.token.Value == "" {
		t.Fatal("token not cached after AuthenticateContext")
	}
	// Second call must hit the cache.
	cachedValue := c.token.Value
	if err := c.AuthenticateContext(ctx); err != nil {
		t.Fatalf("AuthenticateContext (cached): %v", err)
	}
	if c.token.Value != cachedValue {
		t.Fatal("cached token was replaced unexpectedly")
	}
	// Invalidate forces re-auth.
	c.InvalidateToken()
	if c.token != nil {
		t.Fatal("InvalidateToken did not clear the cache")
	}
	if err := c.AuthenticateContext(ctx); err != nil {
		t.Fatalf("AuthenticateContext (after invalidate): %v", err)
	}
	if c.token == nil {
		t.Fatal("post-invalidate auth left token nil")
	}
}

func TestIntegrationGetProjectsAndSelf(t *testing.T) {
	c := liveClient(t)
	projects, err := c.GetProjects()
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if projects == nil {
		t.Fatal("nil slice from GetProjects")
	}
	ctx := context.Background()
	uid, err := c.SelfUserID(ctx)
	if err != nil {
		t.Fatalf("SelfUserID: %v", err)
	}
	if uid == "" {
		t.Fatal("empty self user id")
	}
}

func TestIntegrationProjectAndTaskLifecycle(t *testing.T) {
	c := liveClient(t)
	t.Cleanup(func() { cleanupTestProjects(t, c) })

	suffix := time.Now().UTC().Format("20060102-150405")
	title := testProjectPrefix + suffix
	project, err := c.CreateProject(NewProjectRequest{
		Title:       title,
		Description: "integration test from go-onlyoffice",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if project.ID == nil {
		t.Fatal("created project without id")
	}

	updated, err := c.UpdateProject(ProjectUpdateRequest{
		ID:            *project.ID,
		Title:         title + " (updated)",
		Description:   "updated",
		ResponsibleID: *project.Responsible.ID,
	})
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if updated.Title == nil || *updated.Title != title+" (updated)" {
		t.Errorf("title not updated: %+v", updated.Title)
	}

	start := Time(time.Now().AddDate(0, 0, -2))
	deadline := Time(time.Now().AddDate(0, 0, 2))
	task, err := c.CreateProjectTask(NewProjectTaskRequest{
		ProjectId:   *project.ID,
		Title:       "integration parent task",
		Description: "from go-onlyoffice integration suite",
		StartDate:   start,
		Deadline:    deadline,
		Priority:    int(TaskPriorityNormal),
	})
	if err != nil {
		t.Fatalf("CreateProjectTask: %v", err)
	}
	if task.ID == nil {
		t.Fatal("created task without id")
	}

	newStart := Time(time.Now().AddDate(0, 0, -14))
	newDeadline := Time(time.Now().AddDate(0, 0, 3))
	if _, err := c.UpdateProjectTask(ProjectTaskUpdateRequest{
		ID:          *task.ID,
		Title:       "integration parent task (updated)",
		Description: "updated",
		StartDate:   &newStart,
		Deadline:    &newDeadline,
	}); err != nil {
		t.Fatalf("UpdateProjectTask: %v", err)
	}

	// Subtask uses the form-encoded helper; exercises httpx.postForm path.
	ctx := context.Background()
	sub, err := c.AddSubtask(ctx, strconv.Itoa(*task.ID), "integration subtask")
	if err != nil {
		t.Fatalf("AddSubtask: %v", err)
	}
	if _, ok := sub["id"]; !ok {
		t.Errorf("subtask response missing id: %+v", sub)
	}
}

func TestIntegrationCalendarAndCRMRead(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	start := time.Now().Format("2006-01-02")
	end := time.Now().AddDate(0, 0, 14).Format("2006-01-02")

	if _, err := c.ListCalendars(ctx, start, end); err != nil {
		t.Errorf("ListCalendars: %v", err)
	}
	if _, err := c.ListEvents(ctx, start, end); err != nil {
		t.Errorf("ListEvents: %v", err)
	}
	if _, _, err := c.ListContacts(ctx, 5, 0, ""); err != nil {
		t.Errorf("ListContacts: %v", err)
	}
	if _, _, err := c.ListOpportunities(ctx, 5, 0); err != nil {
		t.Errorf("ListOpportunities: %v", err)
	}
	if _, err := c.ListDealStages(ctx); err != nil {
		t.Errorf("ListDealStages: %v", err)
	}
}

// TestIntegrationDryRunList covers the tiniest project-tasks read that
// inventar-sync depends on; confirms GetTasks deserialises real responses.
func TestIntegrationListTasks(t *testing.T) {
	c := liveClient(t)
	defaults := GetEnvironmentDefaults()
	// ProjectID default is "33"; skip the read-only list if the caller hasn't
	// pointed us at a valid project (we don't know which projects exist).
	if defaults.ProjectID == "" {
		t.Skip("ONLYOFFICE_PROJECT_ID not configured")
	}
	pid, err := strconv.Atoi(defaults.ProjectID)
	if err != nil || pid <= 0 {
		t.Skipf("ONLYOFFICE_PROJECT_ID %q is not a positive int", defaults.ProjectID)
	}
	if _, err := c.GetTasks(NewProjectGetTasksRequest(pid)); err != nil {
		t.Fatalf("GetTasks(%d): %v", pid, err)
	}
}
