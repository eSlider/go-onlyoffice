// Command oo is a thin CLI over the github.com/eslider/go-onlyoffice library.
//
// Command tree is subject-based (mirrors the library split and the `tea` CLI):
//
//	oo calendar      list | events | add | delete
//	oo projects      list | get | milestones | milestone-create | milestone-delete | board-sync | create | update | delete | contacts (add|remove) | team (list|add|remove|set) | link-authors | link-git | files (list|upload|replace-in|update|download|rename|delete|dedupe|as-md|put-md|put-txt|put-xlsx)
//	oo tasks         list | get | create | update | delete | subtask add | files (list|upload|detach)
//	oo users         list | self | get | create | update | delete | block | unblock | password | check   (alias: oo whoami)
//	oo link          FILE_ID [FILE_ID...]   DocEditor deep links (Products/Files/DocEditor.aspx?fileid=…)
//	oo contacts      list | get | delete | info-add | merge | dedupe-info | tags | tag-add | tag-create | tag-remove
//	oo persons       list | create | delete | dedupe
//	oo companies     list | create | delete | dedupe | dedupe-persons
//	oo opportunities list | get | create | update | delete | stages | member-add | dedupe | dedupe-members | fix-titles
//	oo cases         list | create | delete | member-add
//	oo crm-tasks     list | create | delete | categories | reassign-self
//	oo crm           audit | cleanup
//	oo mails         accounts | folders | list | get | download-attachment | draft | attach | draft-invoice | send | delete
//	oo invoices      list | get | create | update | pdf | pdf-cleanup | status | delete | items …
//	oo docs          tools | convert | pdf | presigned | csv | json | optimize | ocr | hocr | as-md | put-md | put-txt | put-xlsx
//	oo catalog       match | merge | apply | scan-contacts | scan-projects | scan-thunderbird
//	oo dav           ls | move | copy | mkdir | rename-file | rename-folder | download | fileops
//	oo search        QUERY [--content] [--folder ID] [--limit N] [--backend oo|own] [--json]
//	oo index         folder FOLDER_ID | files FILE_ID... [--recursive] [--exts pdf] [--dry-run]
//
// CRM association rules: docs/crm-associations.md
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
