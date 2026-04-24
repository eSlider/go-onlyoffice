// Command oo is a thin CLI over the github.com/eslider/go-onlyoffice library.
//
// Command tree is subject-based (mirrors the library split and the `tea` CLI):
//
//	oo calendar      list | events | add | delete
//	oo projects      list | get | milestones | create | update | delete
//	oo tasks         list | get | create | update | delete | subtask add
//	oo users         list | self            (alias: oo whoami)
//	oo contacts      list | get | delete | info-add
//	oo persons       list | create | delete
//	oo companies     list | create | delete
//	oo opportunities list | get | create | delete | stages | member-add
//	oo cases         list | create | delete | member-add
//	oo crm-tasks     list | create | delete | categories
//	oo applications  sync
//
// Every list supports `--output/-o json|table` (table is the default).
//
// Build & install:
//
//	go install github.com/eslider/go-onlyoffice/cmd/oo@latest
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
