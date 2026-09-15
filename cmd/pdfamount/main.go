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

// amountPat is the amount capture shared by every amount regex.
const amountPat = `([0-9]+(?:[.,][0-9]+)*)`

// amountRE builds "<label> [optional (comment)] [: -] <number>".
func amountRE(label string) *regexp.Regexp {
	return regexp.MustCompile(
		`(?i)\b` + regexp.QuoteMeta(label) + `\b\s*(?:\([^)]*\))?\s*[:\-]?\s*` + amountPat)
}

// amountRes lists the payable-amount patterns in strict priority order: the
// first pattern with a usable amount wins, and a lower-priority label can never
// override a higher-priority one ("zu zahlender betrag" > "rechnungsbetrag" >
// "rechnungsendbetrag" > "gesamtbetrag" > "gesamtsumme (inkl. steuern)").
//
// "gesamtbetrag" and "gesamtsumme" are not in the original set but are the real
// labels on Diashop invoices ("Gesamtsumme (inkl. Steuern)"). The inclusive
// variant is matched before a plain "gesamtsumme". Everything after those
// primary labels is the broader fallback set, consulted only when no primary
// label yields an amount. Within one pattern the last usable amount is taken,
// because totals usually come last.
var amountRes = []*regexp.Regexp{
	amountRE("zu zahlender betrag"),
	amountRE("rechnungsbetrag"),
	amountRE("rechnungsendbetrag"),
	amountRE("gesamtbetrag"),
	regexp.MustCompile(`(?i)\bgesamtsumme\b\s*\(\s*inkl\.?\s*steuern\s*\)\s*[:\-]?\s*` + amountPat),
	amountRE("gesamtsumme"),
	amountRE("endbetrag"),
	amountRE("zahlbetrag"),
	amountRE("bruttobetrag"),
	amountRE("betrag"),
	amountRE("total"),
	amountRE("summe"),
}

// taxLineRe marks a line whose number is a tax rate/percentage: an explicit
// percent sign or a VAT/tax keyword. "Steuern" (plural, as in "inkl. Steuern")
// is handled separately so the inclusive total stays usable.
var taxLineRe = regexp.MustCompile(`(?i)%|\bMwSt\b|\bUSt\b|\bProzent\b`)

// steuerRe finds "Steuer"/"Umsatzsteuer" etc. RE2 has no lookahead, so the
// plural "Steuern" is excluded in isTaxLine.
var steuerRe = regexp.MustCompile(`(?i)steuer`)

// percentAfterRe detects a percent sign directly after a number (spaces ok).
var percentAfterRe = regexp.MustCompile(`^\s*%`)

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
// text, or "" if no usable amount matches.
//
// DKV invoices are special-cased first: they repeat a per-vehicle "TOTAL:" line
// and carry the real total only in the "Gesamtsummenaufstellung" section.
func extractAmount(text string) string {
	if v, ok := dkvGrandTotal(text); ok {
		return v
	}
	for _, re := range amountRes {
		if v, ok := lastUsableAmount(text, re); ok {
			return v
		}
	}
	return ""
}

// dkvGrandTotal extracts the total of a DKV "Gesamtsummenaufstellung" section.
//
// Rule: DKV invoices repeat a per-vehicle "TOTAL:" line, so the last TOTAL is
// not the invoice total. When a "Gesamtsummenaufstellung" section exists, its
// total wins over every "TOTAL:" line: the first amount after the "»" marker,
// or, if there is none, the last amount in the section. The section ends at the
// page break (form feed) or end of text.
func dkvGrandTotal(text string) (string, bool) {
	idx := strings.Index(strings.ToLower(text), "gesamtsummenaufstellung")
	if idx < 0 {
		return "", false
	}
	section := text[idx:]
	if ff := strings.IndexByte(section, '\f'); ff >= 0 {
		section = section[:ff]
	}
	if m := strings.Index(section, "»"); m >= 0 {
		if v, ok := firstAmount(section[m:]); ok {
			return v, true
		}
	}
	return lastAmount(section)
}

// lastUsableAmount returns the last amount matched by re that is not a tax rate
// or percentage. Within one label the last usable amount wins.
func lastUsableAmount(text string, re *regexp.Regexp) (string, bool) {
	ms := re.FindAllStringSubmatchIndex(text, -1)
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		if isTaxRate(text, m[2], m[3]) {
			continue
		}
		if v, ok := normalizeAmount(text[m[2]:m[3]]); ok {
			return v, true
		}
	}
	return "", false
}

// isTaxRate reports whether the number at text[start:end] is a tax rate or a
// percentage instead of a payable amount. A candidate is rejected when the
// token right after the number is "%" or the number's line carries a percent
// sign or a tax keyword. Rejecting is deliberate: office matching treats a
// known-but-different amount as a hard disqualifier, so an empty result is
// safer than the VAT rate.
func isTaxRate(text string, start, end int) bool {
	if percentAfterRe.MatchString(text[end:]) {
		return true
	}
	lineStart := strings.LastIndexByte(text[:start], '\n') + 1
	line := text[lineStart:]
	if n := strings.IndexByte(text[end:], '\n'); n >= 0 {
		line = text[lineStart : end+n]
	}
	return isTaxLine(line)
}

// isTaxLine reports whether a line looks like a tax rate rather than a payable
// amount. "Steuern" is treated as a qualifier ("inkl. Steuern"), not a rate.
func isTaxLine(line string) bool {
	if taxLineRe.MatchString(line) {
		return true
	}
	for _, loc := range steuerRe.FindAllStringIndex(line, -1) {
		if loc[1] >= len(line) || (line[loc[1]] != 'n' && line[loc[1]] != 'N') {
			return true
		}
	}
	return false
}

// numberRe finds bare numbers (with optional thousands/decimal separators).
var numberRe = regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)*`)

func firstAmount(s string) (string, bool) {
	for _, m := range numberRe.FindAllString(s, -1) {
		if v, ok := normalizeAmount(m); ok {
			return v, true
		}
	}
	return "", false
}

func lastAmount(s string) (string, bool) {
	ms := numberRe.FindAllString(s, -1)
	for i := len(ms) - 1; i >= 0; i-- {
		if v, ok := normalizeAmount(ms[i]); ok {
			return v, true
		}
	}
	return "", false
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
