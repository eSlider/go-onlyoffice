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

// LayoutWidths splits total terminal width evenly across visible panes.
func LayoutWidths(total int, vis PaneVisibility) PaneWidths {
	if total < 1 {
		total = 80
	}
	v := vis
	n := countVisible(v)
	if n == 0 {
		v = defaultPaneVisibility()
		n = 3
	}
	base := total / n
	rem := total % n
	out := PaneWidths{Visibility: v}
	if v.Menu {
		out.Menu = base
		if rem > 0 {
			out.Menu++
			rem--
		}
	}
	if v.List {
		out.List = base
		if rem > 0 {
			out.List++
			rem--
		}
	}
	if v.Detail {
		out.Detail = base
		if rem > 0 {
			out.Detail++
			rem--
		}
	}
	return out
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
