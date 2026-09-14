// Command ooscan recursively lists OnlyOffice Documents folders into a TSV
// index: file_id, folder_id, path, title.
//
// Usage: ooscan <FOLDER_ID> [<FOLDER_ID>...]
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

func main() {
	ctx := context.Background()
	c := onlyoffice.NewClient(onlyoffice.GetEnvironmentCredentials())
	seen := map[string]bool{}
	for _, root := range os.Args[1:] {
		walk(ctx, c, root, "", 0, seen)
	}
}

func walk(ctx context.Context, c *onlyoffice.Client, folderID, path string, depth int, seen map[string]bool) {
	if depth > 8 || seen[folderID] {
		return
	}
	seen[folderID] = true
	// Throttle: OnlyOffice rate-limits (429) and the host must not be flooded.
	time.Sleep(350 * time.Millisecond)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var l *onlyoffice.DavListing
	derr := onlyoffice.DoRetry(ctx, onlyoffice.DefaultRetryPolicy(), func() error {
		var err error
		l, err = c.ListDavFolder(ctx, folderID)
		return err
	})
	if derr != nil {
		fmt.Fprintf(os.Stderr, "list %s (%s): %v\n", path, folderID, derr)
		return
	}
	for _, f := range l.Files {
		fmt.Printf("%s\t%s\t%s\t%s\n", f.ID, folderID, path, f.Title)
	}
	for _, sub := range l.Folders {
		walk(ctx, c, sub.ID, path+"/"+sub.Title, depth+1, seen)
	}
}
