package onlyoffice

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// Load environment variables
func init() {
	// Load from .env file in the project root
	_ = godotenv.Load(".env")
}

func skipWithoutCredentials(t *testing.T) {
	t.Helper()
	if os.Getenv("ONLYOFFICE_URL") == "" {
		t.Skip("ONLYOFFICE_URL is not set, skipping integration test")
	}
}

func TestNewClient(t *testing.T) {
	skipWithoutCredentials(t)

	// Create a new OnlyOffice client
	credentials := GetEnvironmentCredentials()
	client := NewClient(credentials)
	var token *Token
	var err error
	token, err = client.Auth(&credentials)

	if err != nil {
		t.Errorf("Failed to get token: %v", err)
	}
	if token == nil {
		t.Error("Value is empty")
	}

	// Clean up: remove all projects start with "Test project"
	projects, err := client.GetProjects()
	if err != nil {
		t.Errorf("Failed to get projects: %v", err)
	}

	for _, project := range projects {
		if strings.HasPrefix(*project.Title, "Test project") {
			prjStatus, err := client.DeleteProject(*project.ID)
			if err != nil {
				t.Errorf("Failed to delete project: %v", err)
			}
			if *prjStatus.ID != *project.ID {
				t.Errorf("Project ID is not equal: %v != %v", prjStatus.ID, project.ID)
			}
		}
	}

	// Test create project
	project, err := client.CreateProject(NewProjectRequest{
		Title:       "Test project",
		Description: "Test project description",
	})

	if err != nil {
		t.Errorf("Failed to create project: %v", err)
	}

	// Update project
	project, err = client.UpdateProject(ProjectUpdateRequest{
		ID:            *project.ID,
		Title:         "Test project updated",
		ResponsibleID: *project.Responsible.ID,
	})

	var task *Task
	// Test create project task
	task, err = client.CreateProjectTask(NewProjectTaskRequest{
		ProjectId:   *project.ID,
		Title:       "Test task",
		Description: "Test task description",
		Notify:      true,
		MilestoneId: 0,
		Priority:    0,
		// Deadline +2 days
		StartDate: Time(time.Now().AddDate(0, 0, -2)),
		Deadline:  Time(time.Now().AddDate(0, 0, 2)),
	})

	// Update project task
	startDate := Time(time.Now().AddDate(0, 0, -14))
	deadline := Time(time.Now().AddDate(0, 0, 3))
	task, err = client.UpdateProjectTask(ProjectTaskUpdateRequest{
		ID:          *task.ID,
		Title:       "Test task updated",
		Description: "Test task description updated",
		StartDate:   &startDate,
		Deadline:    &deadline,
	})

	if err != nil {
		t.Errorf("Failed to create task: %v", err)
	}

	if task == nil {
		t.Error("Value is empty")
	}

	// Delete project
	prj, err := client.DeleteProject(*project.ID)
	if err != nil {
		t.Errorf("Failed to delete project: %v", err)
	}

	if *prj.ID != *project.ID {
		t.Errorf("Project ID is not equal: %v != %v", prj.ID, project.ID)
	}

	// Test list projects
	projects, err = client.GetProjects()
	if err != nil {
		t.Errorf("Failed to get projects: %v", err)
	}
	if len(projects) == 0 {
		t.Error("Value is empty")
	}
}
