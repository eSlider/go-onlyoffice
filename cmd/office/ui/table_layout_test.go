package ui

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestLayoutFlexTableAllProjectColumnsVisible(t *testing.T) {
	cols := model.BuildColumns(model.SubjectProjects, nil)
	flex, ok := model.TableFlexLayoutFor(model.SubjectProjects)
	if !ok {
		t.Fatal("expected flex layout for projects")
	}
	lay := layoutFlexTable(cols, 80, flex.FlexColumnKey, flex.MinFlexWidth)
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

func TestLayoutFlexTableUserEmailAbsorbsWidth(t *testing.T) {
	cols := model.BuildColumns(model.SubjectUsers, nil)
	flex, ok := model.TableFlexLayoutFor(model.SubjectUsers)
	if !ok {
		t.Fatal("expected flex layout for users")
	}
	lay := layoutFlexTable(cols, 100, flex.FlexColumnKey, flex.MinFlexWidth)
	sum := 0
	emailW := 0
	for _, i := range lay.indices {
		sum += lay.widths[i]
		if cols[i].Key == "email" {
			emailW = lay.widths[i]
		}
	}
	if sum != 100 {
		t.Fatalf("sum=%d want 100", sum)
	}
	if emailW < 20 {
		t.Fatalf("email width=%d should absorb remainder", emailW)
	}
}
