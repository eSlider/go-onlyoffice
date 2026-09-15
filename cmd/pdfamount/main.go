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

// amountLabels are the payable-amount labels in priority order: the first
// label present in a document wins. Within one label the last amount is taken,
// because totals usually come last.
//
// "gesamtsumme" is not in the original list but is the real label on Diashop
// invoices: "Gesamtsumme (inkl. Steuern)".
var amountLabels = []string{
	"zu zahlender betrag",
	"rechnungsbetrag",
	"rechnungsendbetrag",
	"endbetrag",
	"zahlbetrag",
	"bruttobetrag",
	"gesamtbetrag",
	"gesamtsumme",
	"betrag",
	"total",
	"summe",
}

// amountRes matches "<label> [optional (comment)] [: -] <number>" for every
// label, in the same priority order as amountLabels.
var amountRes = func() []*regexp.Regexp {
	res := make([]*regexp.Regexp, 0, len(amountLabels))
	for _, label := range amountLabels {
		res = append(res, regexp.MustCompile(
			`(?i)\b`+regexp.QuoteMeta(label)+`\b\s*(?:\([^)]*\))?\s*[:\-]?\s*([0-9]+(?:[.,][0-9]+)*)`))
	}
	return res
}()

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
	return extractAmount(buf.String()), nil
}

// extractAmount returns the normalised ("1234.56") payable amount found in
// text, or "" if no known label matches.
func extractAmount(text string) string {
	for _, re := range amountRes {
		ms := re.FindAllStringSubmatch(text, -1)
		if len(ms) == 0 {
			continue
		}
		if v, ok := normalizeAmount(ms[len(ms)-1][1]); ok {
			return v
		}
	}
	return ""
}

// normalizeAmount turns "1.234,56" (DE), "1,234.56" (EN) or "1234.56" into
// "1234.56". The rightmost separator is decimal only when followed by one or
// two digits; otherwise every separator is a thousands separator.
func normalizeAmount(s string) (string, bool) {
	last := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '.' || s[i] == ',' {
			last = i
		}
	}
	var dec byte
	if last >= 0 {
		digits := 0
		for i := last + 1; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				return "", false
			}
			digits++
		}
		if digits == 1 || digits == 2 {
			dec = s[last]
		}
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			b.WriteByte(c)
		case (c == '.' || c == ',') && c == dec:
			b.WriteByte('.')
		case c == '.' || c == ',':
			// thousands separator
		default:
			return "", false
		}
	}
	v, err := strconv.ParseFloat(b.String(), 64)
	if err != nil {
		return "", false
	}
	return strconv.FormatFloat(v, 'f', 2, 64), true
}
