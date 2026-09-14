// Command pdfamount walks a Documents folder, downloads matching PDFs and
// extracts the payable amount, printing "file_id\ttitle\tamount".
//
// Usage: pdfamount <FOLDER_ID> [TITLE_FILTER_REGEX]
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

var amountRe = regexp.MustCompile(`(?i)(zu zahlender betrag|rechnungsbetrag)\s*[:\s]*([0-9][0-9.]*,[0-9]{2})`)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pdfamount <FOLDER_ID> [TITLE_FILTER_REGEX]")
		os.Exit(2)
	}
	folder := os.Args[1]
	filter := regexp.MustCompile(`(?i)rechnung`)
	if len(os.Args) >= 3 {
		filter = regexp.MustCompile(os.Args[2])
	}
	ctx := context.Background()
	c := onlyoffice.NewClient(onlyoffice.GetEnvironmentCredentials())

	files := listAll(ctx, c, folder)
	for _, f := range files {
		if !filter.MatchString(f.title) {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(f.title), ".pdf") {
			continue
		}
		amount, err := pdfAmount(ctx, c, f.id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", f.title, err)
			continue
		}
		if amount == "" {
			continue
		}
		fmt.Printf("%s\t%s\t%s\n", f.id, f.title, amount)
	}
}

type file struct{ id, title string }

func listAll(ctx context.Context, c *onlyoffice.Client, folder string) []file {
	seen := map[string]bool{}
	var out []file
	var walk func(string)
	walk = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		time.Sleep(300 * time.Millisecond)
		l, err := c.ListDavFolder(ctx, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "list %s: %v\n", id, err)
			return
		}
		for _, f := range l.Files {
			out = append(out, file{f.ID, f.Title})
		}
		for _, sub := range l.Folders {
			walk(sub.ID)
		}
	}
	walk(folder)
	return out
}

func pdfAmount(ctx context.Context, c *onlyoffice.Client, id string) (string, error) {
	time.Sleep(time.Second)
	tmp, err := os.CreateTemp("", "pdf-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	derr := onlyoffice.DoRetry(ctx, onlyoffice.DefaultRetryPolicy(), func() error {
		_ = tmp.Truncate(0)
		_, _ = tmp.Seek(0, 0)
		_, err := c.DownloadFile(ctx, id, tmp)
		return err
	})
	if derr != nil {
		tmp.Close()
		return "", derr
	}
	tmp.Close()
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", tmp.Name(), "-")
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	m := amountRe.FindStringSubmatch(buf.String())
	if m == nil {
		return "", nil
	}
	return parseDe(m[2]), nil
}

// parseDe turns "1.234,56" into 1234.56.
func parseDe(s string) string {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
