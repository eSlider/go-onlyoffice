// Command kontolink fills the "Link" column of a Kontoblatt ("ungeklärte
// Posten") XLSX by matching each row to an OnlyOffice document.
//
// Strategy (deterministic, conservative — no LLM):
//  1. Beleg token (letters/digits from the "Beleg" column) appears in the file
//     title; among candidates prefer (a) the row's month, (b) real invoices over
//     copies/dupes, and require the result to be unique;
//  2. else supplier + row month + "rechnung", again unique.
//
// A file is linked at most once (rows already carrying a link are kept and their
// file counts as used). Ambiguous rows are left UNLINKED for manual review.
//
// Usage: kontolink <IN_XLSX> <INDEX_TSV> <OUT_XLSX>
package main

import (
	"context"

	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/xuri/excelize/v2"
)

var (
	dateRe = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`)
	nonAln = regexp.MustCompile(`[^0-9a-z]+`)
	fileID = regexp.MustCompile(`fileid=(\d+)`)
)

func parseDay(s string) (time.Time, bool) {
	t, err := time.Parse("02.01.2006", strings.TrimSpace(s))
	return t, err == nil
}

func titleDay(title string) (time.Time, bool) {
	if len(title) >= 10 {
		if t, err := time.Parse("2006-01-02", title[:10]); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// nearest picks the candidate whose title date is closest to rd. Ties and
// undated candidates (when >1) are rejected.
func nearest(cands []entry, rd time.Time) (entry, bool) {
	if len(cands) == 1 {
		return cands[0], true
	}
	best, bestD, tie := -1, 0.0, false
	for i, e := range cands {
		td, ok := titleDay(e.title)
		if !ok {
			continue
		}
		d := td.Sub(rd).Hours() / 24
		if d < 0 {
			d = -d
		}
		if best < 0 || d < bestD {
			best, bestD, tie = i, d, false
		} else if d == bestD {
			tie = true
		}
	}
	if best < 0 || tie {
		return entry{}, false
	}
	return cands[best], true
}

// portalBaseFromEnv resolves the portal base URL used to build document links.
func portalBaseFromEnv() string {
	for _, k := range []string{"ONLYOFFICE_URL", "ONLYOFFICE_HOST", "OO_URL"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return strings.TrimRight(v, "/")
		}
	}
	return ""
}

type entry struct {
	id, path, title, norm string
}

func norm(s string) string { return nonAln.ReplaceAllString(strings.ToLower(s), "") }

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: kontolink <IN_XLSX> <INDEX_TSV> <OUT_XLSX>")
		os.Exit(2)
	}
	in, idxPath, out := os.Args[1], os.Args[2], os.Args[3]

	portal := portalBaseFromEnv()
	if portal == "" {
		fmt.Fprintln(os.Stderr, "set ONLYOFFICE_URL (or ONLYOFFICE_HOST/OO_URL) to build document links")
		os.Exit(2)
	}

	idxRaw, err := os.ReadFile(idxPath)
	if err != nil {
		panic(err)
	}
	var entries []entry
	for _, line := range strings.Split(string(idxRaw), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 || parts[0] == "" {
			continue
		}
		entries = append(entries, entry{id: parts[0], path: parts[2], title: parts[3], norm: norm(parts[3])})
	}

	f, err := excelize.OpenFile(in)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	sheet := f.GetSheetList()[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		panic(err)
	}

	// optional 5th arg: amounts TSV "file_id\ttitle\tamount" (see cmd/pdfamount)
	var amts []amtEntry
	if len(os.Args) >= 6 && os.Args[5] != "" {
		amts = loadAmounts(os.Args[5])
	}

	used := map[string]bool{}
	for _, r := range rows {
		if m := fileID.FindStringSubmatch(cell(r, 7)); m != nil {
			used[m[1]] = true
		}
	}

	var linked, byBeleg, bySupplier, byAmount, unmatched, ambiguous int
	for i, r := range rows {
		if i == 0 || !dateRe.MatchString(cell(r, 0)) || strings.TrimSpace(cell(r, 7)) != "" {
			continue
		}
		beleg := norm(cell(r, 3))
		supplier := supplierNorm(cell(r, 2))
		month := monthYear(cell(r, 0))
		rd, _ := parseDay(cell(r, 0))

		e, kind, ok := pick(entries, used, beleg, supplier, month, rd)
		if !ok {
			if ae, aok := amountPick(amts, used, supplier, rowAmount(r), rd); aok {
				e, kind, ok = entry{id: ae.id, title: ae.title}, "amount", true
			}
		}
		if !ok {
			if beleg != "" {
				ambiguous++
			} else {
				unmatched++
			}
			continue
		}
		ref, _ := excelize.CoordinatesToCellName(8, i+1)
		if err := f.SetCellValue(sheet, ref, portal+"/Products/Files/DocEditor.aspx?fileid="+e.id); err != nil {
			panic(err)
		}
		used[e.id] = true
		linked++
		switch kind {
		case "beleg":
			byBeleg++
		case "supplier":
			bySupplier++
		case "amount":
			byAmount++
		}
		fmt.Printf("row %3d  %-30s -> %s  [%s]\n", i+1, cell(r, 2), e.title, kind)
	}

	if err := f.SaveAs(out); err != nil {
		panic(err)
	}
	fmt.Printf("\nlinked=%d (beleg=%d, supplier=%d, amount=%d), ambiguous=%d, no-candidate=%d\n",
		linked, byBeleg, bySupplier, byAmount, ambiguous, unmatched)

	// Optional 4th arg: source OnlyOffice file id. Try to update it in place;
	// if it is locked (OnlyOffice 500), upload a "(links)" copy next to it.
	if len(os.Args) >= 5 && os.Args[4] != "" {
		c := onlyoffice.NewClient(onlyoffice.GetEnvironmentCredentials())
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		var src *onlyoffice.FileEntry
		if derr := onlyoffice.DoRetry(ctx, onlyoffice.DefaultRetryPolicy(), func() error {
			var err error
			src, err = c.GetFile(ctx, os.Args[4])
			return err
		}); derr != nil {
			panic(derr)
		}
		folder, title := "", ""
		if src.FolderID != nil {
			folder = src.FolderID.String()
		}
		if src.Title != nil {
			title = *src.Title
		}
		uderr := onlyoffice.DoRetry(ctx, onlyoffice.DefaultRetryPolicy(), func() error {
			_, err := c.UpdateFile(ctx, os.Args[4], out)
			return err
		})
		if uderr == nil {
			fmt.Printf("updated file %s in place\n", os.Args[4])
			return
		}
		fmt.Printf("in-place update failed (locked?); uploading a copy to folder %s\n", folder)
		ext := filepath.Ext(title)
		name := strings.TrimSuffix(title, ext) + " (links)" + ext
		tmp := filepath.Join(os.TempDir(), name)
		data, _ := os.ReadFile(out)
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			panic(err)
		}
		if derr := onlyoffice.DoRetry(ctx, onlyoffice.DefaultRetryPolicy(), func() error {
			_, _, err := c.UploadToFolderReplacing(ctx, folder, tmp)
			return err
		}); derr != nil {
			panic(derr)
		}
		fmt.Printf("uploaded copy: %s -> folder %s\n", name, folder)
	}
}

func cell(r []string, i int) string {
	if i < len(r) {
		return strings.TrimSpace(r[i])
	}
	return ""
}

func supplierNorm(s string) string {
	s = strings.ToUpper(s)
	if i := strings.Index(s, ","); i >= 0 {
		s = s[:i]
	}
	for _, w := range []string{"RE FEHLT", "GS FEHLT", "WOFR", "WOFÜR"} {
		s = strings.ReplaceAll(s, w, "")
	}
	return norm(s)
}

func monthYear(date string) string {
	if len(date) == 10 {
		return date[6:10] + "-" + date[3:5]
	}
	return ""
}

// pick returns an unused candidate. Beleg match wins; supplier+month is a
// fallback. When several candidates qualify, the one closest in time to the row
// date wins; a tie is rejected (ambiguous) rather than guessed.
func pick(entries []entry, used map[string]bool, beleg, supplier, month string, rd time.Time) (entry, string, bool) {
	free := func(e entry) bool { return !used[e.id] }

	if len(beleg) >= 5 {
		var inMonth []entry
		for _, e := range entries {
			if free(e) && belegMatches(e.norm, beleg) &&
				(month == "" || strings.Contains(e.title, month)) {
				inMonth = append(inMonth, e)
			}
		}
		if supplier != "" {
			var s []entry
			for _, e := range inMonth {
				if strings.Contains(e.norm, supplier) {
					s = append(s, e)
				}
			}
			if len(s) > 0 {
				inMonth = s
			}
		}
		inMonth = topRank(inMonth)
		if e, ok := nearest(inMonth, rd); ok {
			return e, "beleg", true
		}
		// A Beleg is present but no file carries it: do NOT fall back to a
		// supplier guess (that links the wrong invoice).
		return entry{}, "", false
	}

	if supplier != "" && month != "" {
		var c []entry
		for _, e := range entries {
			if free(e) && strings.Contains(e.norm, supplier) &&
				strings.Contains(e.title, month) && strings.Contains(e.norm, "rechnung") {
				c = append(c, e)
			}
		}
		c = topRank(c)
		if e, ok := nearest(c, rd); ok {
			return e, "supplier", true
		}
	}
	return entry{}, "", false
}

// topRank keeps only the highest-ranked candidates (real invoice over copy /
// dupe / op), so a tie with a duplicate does not mask the real file.
func topRank(cands []entry) []entry {
	if len(cands) < 2 {
		return cands
	}
	best := 0
	for _, e := range cands {
		if rank(e) > best {
			best = rank(e)
		}
	}
	out := cands[:0]
	for _, e := range cands {
		if rank(e) == best {
			out = append(out, e)
		}
	}
	return out
}

func rank(e entry) int {
	s := 0
	if strings.Contains(e.path, "/2025") || strings.Contains(e.path, "/2024") {
		s += 4
	}
	if strings.Contains(e.norm, "rechnung") {
		s += 2
	}
	if strings.Contains(e.norm, "dupe") || strings.Contains(e.norm, "copy") ||
		strings.Contains(e.norm, "op") {
		s--
	}
	return s
}

// belegMatches reports whether a Beleg identifies the file: the whole normalized
// Beleg appears, or (for long numeric Belege, e.g. "24/641393110") an 8-digit
// window of its longest digit run appears.
func belegMatches(titleNorm, beleg string) bool {
	if strings.Contains(titleNorm, beleg) {
		return true
	}
	run := longestDigitRun(beleg)
	for i := 0; i+8 <= len(run); i++ {
		if strings.Contains(titleNorm, run[i:i+8]) {
			return true
		}
	}
	return false
}

func longestDigitRun(s string) string {
	var best, cur strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			cur.WriteRune(r)
			if cur.Len() > best.Len() {
				best.Reset()
				best.WriteString(cur.String())
			}
		} else {
			cur.Reset()
		}
	}
	return best.String()
}

type amtEntry struct {
	id      string
	title   string
	norm    string
	amount  float64
	date    time.Time
	hasDate bool
}

func loadAmounts(path string) []amtEntry {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []amtEntry
	for _, line := range strings.Split(string(raw), "\n") {
		p := strings.Split(line, "\t")
		if len(p) < 3 {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(p[2]), 64)
		if err != nil {
			continue
		}
		e := amtEntry{id: p[0], title: p[1], norm: norm(p[1]), amount: v}
		if len(p[1]) >= 10 {
			if t, err := time.Parse("2006-01-02", p[1][:10]); err == nil {
				e.date, e.hasDate = t, true
			}
		}
		out = append(out, e)
	}
	return out
}

func rowAmount(r []string) float64 {
	if v := parseAmount(cell(r, 4)); v != 0 {
		return v
	}
	return parseAmount(cell(r, 5))
}

func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// amountPick matches a row to an O2 invoice by amount + nearest date. Scoped to
// Telefonica/O2 rows and O2 files, so it cannot cross-link other suppliers.
func amountPick(amts []amtEntry, used map[string]bool, supplier string, amt float64, rd time.Time) (amtEntry, bool) {
	if amt <= 0 || len(amts) == 0 {
		return amtEntry{}, false
	}
	if !strings.Contains(supplier, "telefonica") && !strings.Contains(supplier, "o2") {
		return amtEntry{}, false
	}
	var cands []amtEntry
	for _, a := range amts {
		if used[a.id] || !strings.Contains(a.norm, "o2") {
			continue
		}
		d := a.amount - amt
		if d < 0 {
			d = -d
		}
		if d > 0.005 {
			continue
		}
		if a.hasDate && !rd.IsZero() {
			days := a.date.Sub(rd).Hours() / 24
			if days < 0 {
				days = -days
			}
			if days > 75 {
				continue
			}
		}
		cands = append(cands, a)
	}
	if len(cands) == 1 {
		return cands[0], true
	}
	best, bestD, tie := -1, 0.0, false
	for i, a := range cands {
		if !a.hasDate {
			continue
		}
		d := a.date.Sub(rd).Hours() / 24
		if d < 0 {
			d = -d
		}
		if best < 0 || d < bestD {
			best, bestD, tie = i, d, false
		} else if d == bestD {
			tie = true
		}
	}
	if best < 0 || tie {
		return amtEntry{}, false
	}
	return cands[best], true
}
