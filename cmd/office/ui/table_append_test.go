package ui

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestDataTableAppendItemsDedupes(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(40, 10)
	tbl.SetData(model.ListSpec{Subject: model.SubjectMailInbox}, []model.Item{
		{ID: "1", Title: "A", Kind: model.KindMail},
	})
	added := tbl.AppendItems([]model.Item{
		{ID: "1", Title: "dup", Kind: model.KindMail},
		{ID: "2", Title: "B", Kind: model.KindMail},
	})
	if added != 1 || len(tbl.Items()) != 2 {
		t.Fatalf("added=%d len=%d", added, len(tbl.Items()))
	}
}

func TestDataTableNearEnd(t *testing.T) {
	tbl := newDataTable()
	tbl.SetSize(40, 10)
	items := make([]model.Item, 10)
	for i := range items {
		items[i] = model.Item{ID: string(rune('0' + i)), Title: "x", Kind: model.KindMail}
	}
	tbl.SetData(model.ListSpec{Subject: model.SubjectMailInbox}, items)
	tbl.cursorRow = 7
	if !tbl.NearEnd(3) {
		t.Fatal("expected near end")
	}
	tbl.cursorRow = 0
	if tbl.NearEnd(3) {
		t.Fatal("expected not near end")
	}
}
