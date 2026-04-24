// Package main shows how to drive the internal applications package — the same
// logic that powers `oo-cli applications-sync` — directly from Go code.
//
//	export ONLYOFFICE_URL="https://your-instance.onlyoffice.com"
//	export ONLYOFFICE_USER="admin@example.com"
//	export ONLYOFFICE_PASS="your-password"
//	go run ./examples/applications /path/to/cv/applications
//
// Without --apply equivalent, it prints a summary but performs no writes.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/internal/applications"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: applications <applications-root>")
		os.Exit(2)
	}
	root := os.Args[1]

	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" {
		fmt.Fprintln(os.Stderr, "ONLYOFFICE_URL is not set")
		os.Exit(1)
	}
	client := onlyoffice.NewClient(creds)

	paths, err := applications.Discover(root)
	if err != nil {
		log.Fatalf("discover: %v", err)
	}
	fmt.Printf("discovered %d application READMEs under %s\n", len(paths), root)

	var apps []applications.Data
	for _, p := range paths {
		d, err := applications.ParseReadme(p)
		if err != nil {
			log.Printf("parse %s: %v", p, err)
			continue
		}
		apps = append(apps, d)
	}

	stats := applications.Sync(context.Background(), client, apps, true /*dryRun*/, true /*verbose*/)
	fmt.Printf("\ndry-run stats: %+v\n", stats)
}
