package ui

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestLayoutProjectTableAllColumnsVisible(t *testing.T) {
	cols := model.BuildColumns(model.SubjectProjects, nil)
	lay := layoutProjectTable(cols, 80)
	if len(lay.indices) != 7 {
		t.Fatalf("got %d visible columns, want 7", len(lay.indices))
	}
	sum := 0
	for _, i := range lay.indices {
		sum += lay.widths[i]
	}
	if sum != 80 {
		t.Fatalf("width sum=%d want 80", sum)
	}
}
