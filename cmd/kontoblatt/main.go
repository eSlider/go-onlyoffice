// Command kontoblatt builds a summary ("сводная таблица") of a Kontoblatt XLSX
// (Datum, Gegenkonto, Buchungstext, Beleg, Soll, Haben, Bemerkung) and uploads
// it back to the same OnlyOffice folder as the source file.
//
// Usage: kontoblatt <FILE_ID> <LOCAL_XLSX>
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/xuri/excelize/v2"
)

type agg struct {
	count   int
	soll    float64
	haben   float64
	reFehlt int
}

type rec struct {
	date, month, konto, text string
	soll, haben              float64
	reFehlt                  bool
}

var dateRe = regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`)

func parseAmount(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", "") // German thousands separator
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func cell(row []string, i int) string {
	if i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: kontoblatt <FILE_ID> <LOCAL_XLSX>")
		os.Exit(2)
	}
	fileID, path := os.Args[1], os.Args[2]
	ctx := context.Background()

	f, err := excelize.OpenFile(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var recs []rec
	for _, sh := range f.GetSheetList() {
		rows, err := f.GetRows(sh)
		if err != nil {
			continue
		}
		for _, r := range rows {
			d := cell(r, 0)
			if !dateRe.MatchString(d) {
				continue
			}
			text := cell(r, 2)
			recs = append(recs, rec{
				date:    d,
				month:   d[3:10],
				konto:   cell(r, 1),
				text:    text,
				soll:    parseAmount(cell(r, 4)),
				haben:   parseAmount(cell(r, 5)),
				reFehlt: strings.Contains(strings.ToUpper(text), "FEHLT"),
			})
		}
	}

	byKonto := map[string]*agg{}
	byMonth := map[string]*agg{}
	getK := func(k string) *agg {
		if byKonto[k] == nil {
			byKonto[k] = &agg{}
		}
		return byKonto[k]
	}
	getM := func(k string) *agg {
		if byMonth[k] == nil {
			byMonth[k] = &agg{}
		}
		return byMonth[k]
	}
	var tot agg
	for _, r := range recs {
		k := getK(r.konto)
		k.count++
		k.soll += r.soll
		k.haben += r.haben
		if r.reFehlt {
			k.reFehlt++
		}
		m := getM(r.month)
		m.count++
		m.soll += r.soll
		m.haben += r.haben
		if r.reFehlt {
			m.reFehlt++
		}
		tot.count++
		tot.soll += r.soll
		tot.haben += r.haben
		if r.reFehlt {
			tot.reFehlt++
		}
	}

	out := excelize.NewFile()
	defer out.Close()
	writeSheet(out, "Nach Gegenkonto", "Gegenkonto", byKonto, tot)
	writeSheet(out, "Nach Monat", "Monat", byMonth, tot)
	outPath := "/tmp/opencode/kontoblatt-zusammenfassung.xlsx"
	if err := out.SaveAs(outPath); err != nil {
		panic(err)
	}

	// upload next to the source file
	creds := onlyoffice.GetEnvironmentCredentials()
	c := onlyoffice.NewClient(creds)
	var src *onlyoffice.FileEntry
	if derr := onlyoffice.DoRetry(ctx, onlyoffice.DefaultRetryPolicy(), func() error {
		var err error
		src, err = c.GetFile(ctx, fileID)
		return err
	}); derr != nil {
		panic(derr)
	}
	folder := ""
	if src.FolderID != nil {
		folder = src.FolderID.String()
	}
	title := ""
	if src.Title != nil {
		title = *src.Title
	}
	fmt.Printf("source: id=%s title=%q folder=%s\n", fileID, title, folder)

	name := "Kontoblatt-1591-2025-Zusammenfassung.xlsx"
	tmp := "/tmp/opencode/" + name
	data, _ := os.ReadFile(outPath)
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		panic(err)
	}
	var entry *onlyoffice.FileEntry
	if derr := onlyoffice.DoRetry(ctx, onlyoffice.DefaultRetryPolicy(), func() error {
		var err error
		entry, _, err = c.UploadToFolderReplacing(ctx, folder, tmp)
		return err
	}); derr != nil {
		panic(derr)
	}
	fmt.Printf("uploaded: %s -> folder %s (id %v)\n", name, folder, entry.ID)

	// print the summary
	printAgg("Nach Gegenkonto", byKonto, tot)
	printAgg("Nach Monat", byMonth, tot)
}

func writeSheet(f *excelize.File, sheet, key string, m map[string]*agg, tot agg) {
	f.NewSheet(sheet)
	rows := [][]any{{key, "Anzahl", "Soll", "Haben", "Saldo", `davon "fehlt"`}}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		a := m[k]
		rows = append(rows, []any{k, a.count, a.soll, a.haben, a.soll - a.haben, a.reFehlt})
	}
	rows = append(rows, []any{"GESAMT", tot.count, tot.soll, tot.haben, tot.soll - tot.haben, tot.reFehlt})
	for i, row := range rows {
		for j, v := range row {
			cellRef, _ := excelize.CoordinatesToCellName(j+1, i+1)
			_ = f.SetCellValue(sheet, cellRef, v)
		}
	}
}

func printAgg(title string, m map[string]*agg, tot agg) {
	fmt.Printf("\n== %s ==\n", title)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("%-12s %6s %12s %12s %12s %7s\n", "key", "count", "soll", "haben", "saldo", "fehlt")
	for _, k := range keys {
		a := m[k]
		fmt.Printf("%-12s %6d %12.2f %12.2f %12.2f %7d\n", k, a.count, a.soll, a.haben, a.soll-a.haben, a.reFehlt)
	}
	fmt.Printf("%-12s %6d %12.2f %12.2f %12.2f %7d\n", "GESAMT", tot.count, tot.soll, tot.haben, tot.soll-tot.haben, tot.reFehlt)
}
