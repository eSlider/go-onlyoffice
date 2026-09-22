package onlyoffice

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWorkbookSheetExport(t *testing.T) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "first")
	_ = f.SetCellValue("first", "A1", "name")
	_ = f.SetCellValue("first", "B1", "age")
	_ = f.SetCellValue("first", "A2", "Alice")
	_ = f.SetCellValue("first", "B2", 30)
	_, _ = f.NewSheet("second")
	_ = f.SetCellValue("second", "A1", "city")
	_ = f.SetCellValue("second", "A2", "Berlin")
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()

	names, err := WorkbookSheetNames(data)
	if err != nil || len(names) != 2 {
		t.Fatalf("names=%v err=%v", names, err)
	}

	csv0, err := WorkbookSheetCSV(data, 0, 0)
	if err != nil || !strings.Contains(csv0, "Alice") {
		t.Fatalf("sheet0 csv=%q err=%v", csv0, err)
	}
	csv1, err := WorkbookSheetCSV(data, 1, 0)
	if err != nil || !strings.Contains(csv1, "Berlin") || strings.Contains(csv1, "Alice") {
		t.Fatalf("sheet1 csv=%q err=%v", csv1, err)
	}
	// negative index = first sheet
	csvDefault, _ := WorkbookSheetCSV(data, -1, 0)
	if !strings.Contains(csvDefault, "Alice") {
		t.Fatalf("default csv=%q", csvDefault)
	}

	js, err := WorkbookSheetJSON(data, 1)
	if err != nil || len(js) != 1 || js[0]["city"] != "Berlin" {
		t.Fatalf("sheet1 json=%#v err=%v", js, err)
	}

	if _, err := WorkbookSheetCSV(data, 5, 0); err == nil {
		t.Fatal("expected out-of-range error")
	}
}
