package ui

import "testing"

func TestDividerAtBetweenMenuAndList(t *testing.T) {
	pw := PaneWidths{
		Menu: 30, List: 50, Detail: 40,
		Visibility: PaneVisibility{Menu: true, List: true, Detail: true},
	}
	if got := DividerAt(30, pw); got != 0 {
		t.Fatalf("divider at 30=%d want 0", got)
	}
	if got := DividerAt(80, pw); got != 1 {
		t.Fatalf("divider at 80=%d want 1", got)
	}
}

func TestDragPaneDividerRespectsMinimum(t *testing.T) {
	pw := PaneWidths{
		Menu: 40, List: 40, Detail: 40,
		Visibility: PaneVisibility{Menu: true, List: true, Detail: true},
	}
	next, ok := DragPaneDivider(pw, 0, -30)
	if ok {
		t.Fatalf("expected resize blocked by minimum, got %+v", next)
	}
	next, ok = DragPaneDivider(pw, 0, 5)
	if !ok || next.Menu != 45 || next.List != 35 {
		t.Fatalf("resize failed: ok=%v %+v", ok, next)
	}
}

func TestFitPaneWidthsScalesToTotal(t *testing.T) {
	vis := PaneVisibility{Menu: true, List: true, Detail: true}
	sizes := PaneWidths{Menu: 20, List: 30, Detail: 50, Visibility: vis}
	out := FitPaneWidths(120, vis, sizes)
	sum := out.Menu + out.List + out.Detail
	if sum != 120 {
		t.Fatalf("sum=%d want 120", sum)
	}
}
