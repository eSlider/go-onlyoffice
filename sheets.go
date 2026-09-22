package onlyoffice

// Spreadsheet (XLS/XLSX/ODS) sheet-aware export to CSV / JSON.
//
// The OnlyOffice DocumentServer converter can emit CSV, but only for the first
// worksheet and it ignores any sheet selector (verified against a live DS).
// So sheet selection and JSON use a local reader (excelize). This is data
// extraction, not a document-format conversion pipeline.

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// WorkbookSheetNames returns worksheet names in workbook order.
func WorkbookSheetNames(data []byte) ([]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	defer f.Close()
	return f.GetSheetList(), nil
}

// workbookSheet resolves a 0-based sheet index to its name. A negative index
// selects the first sheet; an out-of-range index is an error.
func workbookSheet(f *excelize.File, index int) (string, error) {
	names := f.GetSheetList()
	if len(names) == 0 {
		return "", fmt.Errorf("workbook has no sheets")
	}
	if index < 0 {
		index = 0
	}
	if index >= len(names) {
		return "", fmt.Errorf("sheet index %d out of range (0..%d)", index, len(names)-1)
	}
	return names[index], nil
}

// WorkbookSheetCSV renders one worksheet as CSV. sheetIndex is 0-based
// (negative = first). delimiter 0 keeps the default comma. header, when true,
// is kept as the first CSV row (it is always kept; the flag only affects JSON).
func WorkbookSheetCSV(data []byte, sheetIndex int, delimiter rune) (string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("open workbook: %w", err)
	}
	defer f.Close()
	name, err := workbookSheet(f, sheetIndex)
	if err != nil {
		return "", err
	}
	rows, err := f.GetRows(name)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	if delimiter != 0 {
		w.Comma = delimiter
	}
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()
	return b.String(), w.Error()
}

// WorkbookSheetJSON renders one worksheet as a list of row objects, using the
// first row as the field names (blank headers become col1, col2, …). sheetIndex
// is 0-based (negative = first).
func WorkbookSheetJSON(data []byte, sheetIndex int) ([]map[string]any, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	defer f.Close()
	name, err := workbookSheet(f, sheetIndex)
	if err != nil {
		return nil, err
	}
	rows, err := f.GetRows(name)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	header := rows[0]
	out := make([]map[string]any, 0, len(rows)-1)
	for _, row := range rows[1:] {
		m := make(map[string]any, len(header))
		for i, h := range header {
			key := strings.TrimSpace(h)
			if key == "" {
				key = fmt.Sprintf("col%d", i+1)
			}
			if i < len(row) {
				m[key] = row[i]
			} else {
				m[key] = ""
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// WorkbookSheetCSVTo is WorkbookSheetCSV writing into w.
func WorkbookSheetCSVTo(data []byte, sheetIndex int, delimiter rune, w io.Writer) error {
	s, err := WorkbookSheetCSV(data, sheetIndex, delimiter)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}
