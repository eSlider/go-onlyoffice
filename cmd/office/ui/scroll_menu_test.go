package ui

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestScrollFocusedPaneMenuUpDownMovesTree(t *testing.T) {
	m := NewModel(nil)
	m.focus = model.FocusMenu
	m.syncMenuContent()
	before := m.nav.Cursor()

	if m.scrollFocusedPane("down") {
		t.Fatal("down should not be consumed by menu viewport scroll")
	}
	m.moveDown()
	m.syncFocusedPane()
	if m.nav.Cursor() != before+1 {
		t.Fatalf("cursor=%d want %d", m.nav.Cursor(), before+1)
	}
}

func TestScrollFocusedPaneMenuPageKeysScrollViewport(t *testing.T) {
	m := NewModel(nil)
	m.focus = model.FocusMenu
	for i := 0; i < 40; i++ {
		m.nav.MoveDown()
	}
	m.syncMenuContent()
	if !m.scrollFocusedPane("pgup") {
		t.Fatal("pgup should scroll menu viewport")
	}
}

func TestHandleMenuHorizontalRightOpensLeaf(t *testing.T) {
	m := NewModel(nil)
	m.focus = model.FocusMenu
	// cursor starts on Projects leaf
	next, cmd := m.handleMenuHorizontal(1)
	if cmd == nil {
		t.Fatal("expected load list command")
	}
	if next.listSpec.Subject != model.SubjectProjects {
		t.Fatalf("expected projects list, got %+v", next.listSpec)
	}
}
