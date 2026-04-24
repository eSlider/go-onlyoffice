// Command oo-cli is a thin CLI wrapper over the github.com/eslider/go-onlyoffice
// library. It exposes calendar, CRM, tasks, subtasks, and application-sync
// commands. Build & install:
//
//	go install github.com/eslider/go-onlyoffice/cmd/oo-cli@latest
package main

import (
	"fmt"
	"os"

	"github.com/eslider/go-onlyoffice/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
