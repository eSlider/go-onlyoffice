package xlspipe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

const commentAuthor = "oo"

// Save writes the workbook to path (creates parent dirs).
func Save(f *excelize.File, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return f.SaveAs(path)
}

// quoteSheet returns an Excel sheet reference safe for formulas ('Name'!A1).
func quoteSheet(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

// defineInput registers a named range on sheet!B{row}, sets value, note, optional comment.
func defineInput(f *excelize.File, sheet, name string, row int, value float64, note, comment string) error {
	cell := fmt.Sprintf("B%d", row)
	if err := f.SetCellFloat(sheet, cell, value, 2, 64); err != nil {
		return err
	}
	if note != "" {
		if err := f.SetCellStr(sheet, fmt.Sprintf("C%d", row), note); err != nil {
			return err
		}
	}
	ref := fmt.Sprintf("%s!$B$%d", quoteSheet(sheet), row)
	if err := f.SetDefinedName(&excelize.DefinedName{Name: name, RefersTo: ref}); err != nil {
		return err
	}
	if comment != "" {
		return addCellComment(f, sheet, cell, comment)
	}
	return nil
}

func addCellComment(f *excelize.File, sheet, cell, text string) error {
	return f.AddComment(sheet, excelize.Comment{
		Cell:   cell,
		Author: commentAuthor,
		Text:   text,
		Width:  280,
		Height: 120,
	})
}

func setHeaders(f *excelize.File, sheet string, headers ...string) error {
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellStr(sheet, cell, h); err != nil {
			return err
		}
	}
	return nil
}

func setFormulaRow(f *excelize.File, sheet string, row int, label string, low, mid, high, note string) error {
	if err := f.SetCellStr(sheet, fmt.Sprintf("A%d", row), label); err != nil {
		return err
	}
	for col, formula := range []string{low, mid, high} {
		cell, err := excelize.CoordinatesToCellName(col+2, row)
		if err != nil {
			return err
		}
		if err := f.SetCellFormula(sheet, cell, formula); err != nil {
			return err
		}
	}
	if note != "" {
		return f.SetCellStr(sheet, fmt.Sprintf("E%d", row), note)
	}
	return nil
}

func styleSheet(f *excelize.File, sheet, inputsSheet string) error {
	widths := map[string]float64{"A": 34, "B": 12, "C": 12, "D": 12, "E": 40}
	for col, w := range widths {
		if err := f.SetColWidth(sheet, col, col, w); err != nil {
			return err
		}
	}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8F0FE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
	})
	if err != nil {
		return err
	}
	euroStyle, err := f.NewStyle(&excelize.Style{NumFmt: 4})
	if err != nil {
		return err
	}
	inputStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 4,
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"#FFF9E6"}, Pattern: 1},
	})
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", "E1", headerStyle); err != nil {
		return err
	}
	if sheet == inputsSheet {
		return f.SetCellStyle(sheet, "B2", "B30", inputStyle)
	}
	if err := f.SetCellStyle(sheet, "B2", "D20", euroStyle); err != nil {
		return err
	}
	return nil
}

func finalizeWorkbook(f *excelize.File) error {
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: boolPtr(true)}); err != nil {
		return err
	}
	return f.UpdateLinkedValue()
}

func boolPtr(v bool) *bool { return &v }
