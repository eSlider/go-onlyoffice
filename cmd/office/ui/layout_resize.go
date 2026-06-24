package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

const (
	minPaneOuterWidth = 14
	resizeHitSlop     = 1
)

type paneResizeState struct {
	active   bool
	divider  int
	anchorX  int
	anchorPW PaneWidths
}

// FitPaneWidths scales stored pane widths to fill total width for visible panes.
func FitPaneWidths(total int, vis PaneVisibility, sizes PaneWidths) PaneWidths {
	if total < 1 {
		total = 80
	}
	out := PaneWidths{Visibility: vis}
	if countVisible(vis) == 0 {
		return LayoutWidths(total, vis)
	}
	sum := 0
	if vis.Menu {
		sum += sizes.Menu
	}
	if vis.List {
		sum += sizes.List
	}
	if vis.Detail {
		sum += sizes.Detail
	}
	if sum <= 0 {
		return LayoutWidths(total, vis)
	}
	assign := func(visible bool, stored int) int {
		if !visible {
			return 0
		}
		return stored * total / sum
	}
	out.Menu = assign(vis.Menu, sizes.Menu)
	out.List = assign(vis.List, sizes.List)
	out.Detail = assign(vis.Detail, sizes.Detail)
	fixPaneWidthSum(&out, total)
	return out
}

func fixPaneWidthSum(pw *PaneWidths, total int) {
	if pw == nil || total < 1 {
		return
	}
	type slot struct {
		visible bool
		width   *int
	}
	slots := []slot{
		{pw.Visibility.Menu, &pw.Menu},
		{pw.Visibility.List, &pw.List},
		{pw.Visibility.Detail, &pw.Detail},
	}
	sum := 0
	var last *int
	for _, s := range slots {
		if !s.visible {
			continue
		}
		sum += *s.width
		last = s.width
	}
	if last == nil {
		return
	}
	*last += total - sum
	if *last < minPaneOuterWidth {
		*last = minPaneOuterWidth
	}
}

// DividerPositions returns x coordinates of vertical pane separators.
func DividerPositions(pw PaneWidths) []int {
	var out []int
	x := 0
	if pw.Visibility.Menu {
		x += pw.Menu
		if pw.Visibility.List || pw.Visibility.Detail {
			out = append(out, x)
		}
	}
	if pw.Visibility.List {
		x += pw.List
		if pw.Visibility.Detail {
			out = append(out, x)
		}
	}
	return out
}

// DividerAt returns the divider index under x, or -1.
func DividerAt(x int, pw PaneWidths) int {
	for i, pos := range DividerPositions(pw) {
		if x >= pos-resizeHitSlop && x <= pos+resizeHitSlop {
			return i
		}
	}
	return -1
}

// DragPaneDivider adjusts adjacent pane widths by deltaX pixels.
func DragPaneDivider(pw PaneWidths, divider, deltaX int) (PaneWidths, bool) {
	if deltaX == 0 {
		return pw, false
	}
	changed := false
	switch divider {
	case 0:
		if pw.Visibility.Menu && pw.Visibility.List {
			if resizePair(&pw.Menu, &pw.List, deltaX) {
				changed = true
			}
		} else if pw.Visibility.Menu && pw.Visibility.Detail {
			if resizePair(&pw.Menu, &pw.Detail, deltaX) {
				changed = true
			}
		}
	case 1:
		if pw.Visibility.List && pw.Visibility.Detail {
			if resizePair(&pw.List, &pw.Detail, deltaX) {
				changed = true
			}
		}
	}
	return pw, changed
}

func resizePair(left, right *int, delta int) bool {
	if left == nil || right == nil {
		return false
	}
	newLeft := *left + delta
	newRight := *right - delta
	if newLeft < minPaneOuterWidth || newRight < minPaneOuterWidth {
		return false
	}
	*left = newLeft
	*right = newRight
	return true
}

func (m Model) paneResizeMaxY() int {
	return m.paneHeight() + 1
}

func (m *Model) handlePaneResizeMouse(msg tea.MouseMsg) bool {
	if msg.Y >= m.paneResizeMaxY() {
		if m.resize.active && msg.Action == tea.MouseActionRelease {
			m.resize.active = false
			return true
		}
		return false
	}

	pw := m.paneLayout()
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return false
		}
		if div := DividerAt(msg.X, pw); div >= 0 {
			m.paneSizes = pw
			m.customPaneLayout = true
			m.resize = paneResizeState{
				active:   true,
				divider:  div,
				anchorX:  msg.X,
				anchorPW: pw,
			}
			return true
		}
	case tea.MouseActionRelease:
		if m.resize.active {
			m.resize.active = false
			return true
		}
	case tea.MouseActionMotion:
		if m.resize.active {
			delta := msg.X - m.resize.anchorX
			next, ok := DragPaneDivider(m.resize.anchorPW, m.resize.divider, delta)
			if ok {
				m.paneSizes = next
				m.layoutViewports()
				m.syncAllContent()
			}
			return true
		}
	}
	return m.resize.active
}
