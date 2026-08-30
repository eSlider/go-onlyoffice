package xlspipe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func parseCalcFloat(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	var val float64
	_, err := fmt.Sscan(s, &val)
	return val, err
}

func TestBuildCutoverPortugalWorkbook_Formulas(t *testing.T) {
	f, err := BuildCutoverPortugalWorkbook(DefaultCutoverPortugal())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "cutover.xlsx")
	if err := Save(f, path); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() < 4096 {
		t.Fatalf("xlsx too small: %v", err)
	}

	opened, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()

	cases := []struct {
		sheet, cell string
		want        float64
		tol         float64
	}{
		{SheetDom6, "C8", 1522.89, 0.02},
		{SheetJue3, "C8", 1549.09, 0.02},
		{SheetWedHyp, "C11", 1891.95, 0.05},
		{SheetSummary, "D2", 1522.89, 0.02},
		{SheetSummary, "D7", 1522.89, 0.02},
	}
	for _, tc := range cases {
		got, err := opened.CalcCellValue(tc.sheet, tc.cell)
		if err != nil {
			t.Fatalf("%s!%s calc: %v", tc.sheet, tc.cell, err)
		}
		val, err := parseCalcFloat(got)
		if err != nil {
			t.Fatalf("%s!%s parse %q: %v", tc.sheet, tc.cell, got, err)
		}
		if diff := val - tc.want; diff < -tc.tol || diff > tc.tol {
			t.Fatalf("%s!%s = %v want ~%v", tc.sheet, tc.cell, val, tc.want)
		}
	}
}
