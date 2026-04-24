// Package main shows how to list calendars and events, and create a new event
// via the OnlyOffice Calendar API.
//
//	export ONLYOFFICE_URL="https://your-instance.onlyoffice.com"
//	export ONLYOFFICE_USER="admin@example.com"
//	export ONLYOFFICE_PASS="your-password"
//	# optional: default calendar for AddEvent
//	export ONLYOFFICE_CALENDAR_ID="42"
//	go run ./examples/calendar
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

func main() {
	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" {
		fmt.Fprintln(os.Stderr, "ONLYOFFICE_URL is not set")
		os.Exit(1)
	}

	client := onlyoffice.NewClient(creds)
	client.SetDefaults(onlyoffice.GetEnvironmentDefaults())

	ctx := context.Background()
	start := time.Now().Format("2006-01-02")
	end := time.Now().AddDate(0, 1, 0).Format("2006-01-02")

	calendars, err := client.ListCalendars(ctx, start, end)
	if err != nil {
		log.Fatalf("list calendars: %v", err)
	}
	fmt.Printf("Calendars (%d):\n", len(calendars))
	for _, cal := range calendars {
		fmt.Printf("  - id=%v title=%v\n", cal["objectId"], cal["title"])
	}

	events, err := client.ListEvents(ctx, start, end)
	if err != nil {
		log.Fatalf("list events: %v", err)
	}
	fmt.Printf("\nEvents in [%s..%s]: %d\n", start, end, len(events))
	for _, e := range events {
		fmt.Printf("  - %v @ %v\n", e["title"], e["start"])
	}

	// Create demo event — uncomment to exercise.
	// evStart := time.Now().Add(time.Hour).Format(time.RFC3339)
	// evEnd := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	// created, err := client.AddEvent(ctx, "", "Library demo", evStart, evEnd, "via examples/calendar", false)
	// if err != nil {
	// 	log.Fatalf("add event: %v", err)
	// }
	// fmt.Printf("created event id=%v\n", created["objectId"])
}
