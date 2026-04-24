// Package main creates a parent task and attaches two subtasks using the
// project-tasks API (POST /api/2.0/project/task/{id}.json).
//
//	export ONLYOFFICE_URL="https://your-instance.onlyoffice.com"
//	export ONLYOFFICE_USER="admin@example.com"
//	export ONLYOFFICE_PASS="your-password"
//	export ONLYOFFICE_PROJECT_ID="123"
//	go run ./examples/subtasks
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

func main() {
	creds := onlyoffice.GetEnvironmentCredentials()
	defaults := onlyoffice.GetEnvironmentDefaults()
	if creds.Url == "" || defaults.ProjectID == "" {
		fmt.Fprintln(os.Stderr, "ONLYOFFICE_URL and ONLYOFFICE_PROJECT_ID must be set")
		os.Exit(1)
	}

	client := onlyoffice.NewClient(creds)
	client.SetDefaults(defaults)
	ctx := context.Background()

	parent, err := client.AddTask(ctx, defaults.ProjectID, "Example parent", "Created by examples/subtasks", 0, "")
	if err != nil {
		log.Fatalf("add parent task: %v", err)
	}
	parentID := fmt.Sprint(parent["id"])
	fmt.Printf("parent task id=%s\n", parentID)

	for _, title := range []string{"first subtask", "second subtask"} {
		st, err := client.AddSubtask(ctx, parentID, title)
		if err != nil {
			log.Fatalf("add subtask %q: %v", title, err)
		}
		fmt.Printf("  + subtask id=%v title=%v\n", st["id"], st["title"])
	}

	// Cleanup — uncomment once you verified the subtasks in the OnlyOffice UI.
	// if _, err := client.DeleteTask(ctx, parentID); err != nil {
	// 	log.Printf("cleanup: %v", err)
	// }
}
