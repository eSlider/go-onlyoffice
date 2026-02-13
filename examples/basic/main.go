// Package main demonstrates basic usage of the onlyoffice client library.
//
// Set environment variables before running:
//
//	export ONLYOFFICE_URL="https://your-instance.onlyoffice.com"
//	export ONLYOFFICE_USER="admin@example.com"
//	export ONLYOFFICE_PASS="your-password"
//	go run ./examples/basic/
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

func main() {
	// Load credentials from environment
	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" {
		fmt.Fprintln(os.Stderr, "ONLYOFFICE_URL is not set")
		os.Exit(1)
	}

	client := onlyoffice.NewClient(creds)

	// List all projects
	projects, err := client.GetProjects()
	if err != nil {
		log.Fatalf("Failed to get projects: %v", err)
	}

	fmt.Printf("Found %d projects:\n", len(projects))
	for _, p := range projects {
		fmt.Printf("  - [%d] %s (tasks: %d)\n", *p.ID, *p.Title, safeInt(p.TaskCountTotal))
	}

	// List users
	users, err := client.GetUsers()
	if err != nil {
		log.Fatalf("Failed to get users: %v", err)
	}

	fmt.Printf("\nFound %d users:\n", len(users))
	for _, u := range users {
		fmt.Printf("  - %s (%s)\n", safeStr(u.DisplayName), safeStr(u.Email))
	}

	// Create a demo project (uncomment to test)
	// project, err := client.CreateProject(onlyoffice.NewProjectRequest{
	// 	Title:       "Demo Project",
	// 	Description: "Created via API example",
	// })
	// if err != nil {
	// 	log.Fatalf("Failed to create project: %v", err)
	// }
	// fmt.Printf("\nCreated project: %s (ID: %d)\n", *project.Title, *project.ID)

	// Create a task in first project (uncomment to test)
	// if len(projects) > 0 {
	// 	task, err := client.CreateProjectTask(onlyoffice.NewProjectTaskRequest{
	// 		Title:       "Demo Task",
	// 		Description: "Created via API example",
	// 		ProjectId:   *projects[0].ID,
	// 		StartDate:   onlyoffice.Time(time.Now()),
	// 		Deadline:    onlyoffice.Time(time.Now().AddDate(0, 0, 7)),
	// 	})
	// 	if err != nil {
	// 		log.Fatalf("Failed to create task: %v", err)
	// 	}
	// 	fmt.Printf("Created task: %s (ID: %d)\n", *task.Title, *task.ID)
	// }

	_ = time.Now // suppress unused import if examples are commented out
}

func safeStr(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func safeInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
