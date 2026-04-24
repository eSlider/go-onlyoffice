// Command oo is a thin CLI over the github.com/eslider/go-onlyoffice library.
// It exposes calendar, CRM, tasks, subtasks, and application-sync commands.
//
// Build & install:
//
//	go install github.com/eslider/go-onlyoffice/cmd/oo@latest
//
// Rationale: previous versions shipped as `oo-cli` with the cobra tree living
// in `internal/cli`. Since the CLI is a consumer of the library — not a part
// of its public surface — cobra lives right here in package main, split by
// domain (calendar.go, crm.go, tasks.go, apps.go).
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
