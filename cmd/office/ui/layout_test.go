package ui

import (
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func TestLayoutWidthsAllVisibleUsesFullWidth(t *testing.T) {
	vis := PaneVisibility{Menu: true, List: true, Detail: true}
	pw := LayoutWidths(120, vis)
	sum := pw.Menu + pw.List + pw.Detail
	if sum != 120 {
		t.Fatalf("sum=%d want 120", sum)
	}
	if pw.Menu != 12 || pw.List != 72 || pw.Detail != 36 {
		t.Fatalf("menu=%d list=%d detail=%d want 12/72/36 (10/60/30)", pw.Menu, pw.List, pw.Detail)
	}
}

func TestLayoutWidthsSinglePane(t *testing.T) {
	pw := LayoutWidths(100, PaneVisibility{Menu: false, List: true, Detail: false})
	if pw.List != 100 {
		t.Fatalf("list=%d want 100", pw.List)
	}
	if pw.Menu != 0 || pw.Detail != 0 {
		t.Fatalf("unexpected widths: %+v", pw)
	}
}

func TestLayoutWidthsTwoPanes(t *testing.T) {
	pw := LayoutWidths(80, PaneVisibility{Menu: true, List: false, Detail: true})
	if pw.Menu+pw.Detail != 80 {
		t.Fatalf("sum=%d want 80", pw.Menu+pw.Detail)
	}
	if pw.Menu != 20 || pw.Detail != 60 {
		t.Fatalf("menu=%d detail=%d want 20/60 (10/30 of pair)", pw.Menu, pw.Detail)
	}
}

func TestDetailPaneXRangeSkipsHidden(t *testing.T) {
	pw := PaneWidths{
		Menu: 20, List: 60, Detail: 0,
		Visibility: PaneVisibility{Menu: true, List: true, Detail: false},
	}
	start, end := DetailPaneXRange(pw)
	if start != 0 || end != 0 {
		t.Fatalf("hidden detail should have empty range, got %d..%d", start, end)
	}
}

func TestNextVisibleFocusSkipsHidden(t *testing.T) {
	vis := PaneVisibility{Menu: false, List: true, Detail: true}
	if got := NextVisibleFocus(model.FocusList, vis); got != model.FocusPreview {
		t.Fatalf("got %v", got)
	}
	if got := NextVisibleFocus(model.FocusPreview, vis); got != model.FocusList {
		t.Fatalf("got %v", got)
	}
}

func TestPrevVisibleFocusSkipsHidden(t *testing.T) {
	vis := PaneVisibility{Menu: true, List: false, Detail: true}
	if got := PrevVisibleFocus(model.FocusPreview, vis); got != model.FocusMenu {
		t.Fatalf("got %v", got)
	}
}

func TestLayoutWidthsLegacyHelper(t *testing.T) {
	menu, list, preview := LayoutWidthsLegacy(120)
	if menu+list+preview != 120 {
		t.Fatalf("widths exceed total")
	}
}

func TestKeyActionTogglePanes(t *testing.T) {
	if got := KeyAction("alt+1", model.FocusList); got != ActionToggleMenuPane {
		t.Fatalf("got %v", got)
	}
	if got := KeyAction("alt+2", model.FocusMenu); got != ActionToggleListPane {
		t.Fatalf("got %v", got)
	}
	if got := KeyAction("alt+3", model.FocusList); got != ActionToggleDetailPane {
		t.Fatalf("got %v", got)
	}
}
