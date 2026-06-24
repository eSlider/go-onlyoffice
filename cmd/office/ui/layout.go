package ui

import "github.com/eslider/go-onlyoffice/cmd/office/model"

// PaneVisibility controls which columns are shown.
type PaneVisibility struct {
	Menu, List, Detail bool
}

// PaneWidths is outer lipgloss width per column (sums to terminal width).
type PaneWidths struct {
	Menu, List, Detail int
	Visibility         PaneVisibility
}

func defaultPaneVisibility() PaneVisibility {
	return PaneVisibility{Menu: true, List: true, Detail: true}
}

const (
	defaultMenuShare  = 10
	defaultListShare  = 60
	defaultDetailShare = 30
)

// LayoutWidths splits total terminal width across visible panes (default 10% / 60% / 30%).
func LayoutWidths(total int, vis PaneVisibility) PaneWidths {
	if total < 1 {
		total = 80
	}
	v := vis
	if countVisible(v) == 0 {
		v = defaultPaneVisibility()
	}
	shareSum := 0
	if v.Menu {
		shareSum += defaultMenuShare
	}
	if v.List {
		shareSum += defaultListShare
	}
	if v.Detail {
		shareSum += defaultDetailShare
	}
	assign := func(visible bool, share int) int {
		if !visible || shareSum == 0 {
			return 0
		}
		return total * share / shareSum
	}
	out := PaneWidths{Visibility: v}
	out.Menu = assign(v.Menu, defaultMenuShare)
	out.List = assign(v.List, defaultListShare)
	out.Detail = assign(v.Detail, defaultDetailShare)
	fixPaneWidthSum(&out, total)
	return out
}

// DetailPaneXRange returns the [start, end) column span of the detail pane.
func DetailPaneXRange(pw PaneWidths) (start, end int) {
	if !pw.Visibility.Detail {
		return 0, 0
	}
	if pw.Visibility.Menu {
		start += pw.Menu
	}
	if pw.Visibility.List {
		start += pw.List
	}
	end = start + pw.Detail
	return start, end
}

func countVisible(v PaneVisibility) int {
	n := 0
	if v.Menu {
		n++
	}
	if v.List {
		n++
	}
	if v.Detail {
		n++
	}
	return n
}

// NextVisibleFocus cycles focus forward, skipping hidden panes.
func NextVisibleFocus(cur model.FocusPane, vis PaneVisibility) model.FocusPane {
	order := []model.FocusPane{model.FocusMenu, model.FocusList, model.FocusPreview}
	start := 0
	for i, p := range order {
		if p == cur {
			start = i
			break
		}
	}
	for step := 1; step <= len(order); step++ {
		p := order[(start+step)%len(order)]
		if paneVisible(p, vis) {
			return p
		}
	}
	return cur
}

// PrevVisibleFocus cycles focus backward, skipping hidden panes.
func PrevVisibleFocus(cur model.FocusPane, vis PaneVisibility) model.FocusPane {
	order := []model.FocusPane{model.FocusMenu, model.FocusList, model.FocusPreview}
	start := 0
	for i, p := range order {
		if p == cur {
			start = i
			break
		}
	}
	for step := 1; step <= len(order); step++ {
		p := order[(start-step+len(order))%len(order)]
		if paneVisible(p, vis) {
			return p
		}
	}
	return cur
}

func paneVisible(p model.FocusPane, vis PaneVisibility) bool {
	switch p {
	case model.FocusMenu:
		return vis.Menu
	case model.FocusList:
		return vis.List
	default:
		return vis.Detail
	}
}

func firstVisibleFocus(vis PaneVisibility) model.FocusPane {
	if vis.Menu {
		return model.FocusMenu
	}
	if vis.List {
		return model.FocusList
	}
	return model.FocusPreview
}
