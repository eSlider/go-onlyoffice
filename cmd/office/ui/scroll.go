package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func (m *Model) scrollFocusedPane(key string) bool {
	switch m.focus {
	case model.FocusMenu:
		switch key {
		case "pgdown", "pgdn", "ctrl+d", "pgup", "b", "ctrl+u", "home", "g", "end", "G":
			return scrollViewport(&m.menuVP, key)
		}
		return false
	case model.FocusList:
		switch key {
		case "pgdown", "pgdn", "ctrl+d":
			m.listTable.PageScroll(1)
			return true
		case "pgup", "b", "ctrl+u":
			m.listTable.PageScroll(-1)
			return true
		}
		return false
	case model.FocusPreview:
		if m.detail.Zone() != detailZoneContent {
			return false
		}
		return m.detail.ScrollDocument(key)
	default:
		return false
	}
}

func (m *Model) paneHeight() int {
	h := m.height - 3
	if h < 3 {
		h = 3
	}
	return h
}

const (
	paneBorderChars  = 2 // left + right border drawn outside lipgloss Width
	panePaddingChars = 2 // Padding(0, 1) on paneStyle
)

// paneLipglossWidth maps an on-screen pane width to lipgloss Style.Width.
// Bordered panes render two cells wider than the Width value.
func paneLipglossWidth(rendered int) int {
	if rendered <= 0 {
		return 8
	}
	w := rendered - paneBorderChars
	if w < 8 {
		w = 8
	}
	return w
}

// paneContentWidth is the usable inner width for viewports and tables.
func paneContentWidth(rendered int) int {
	if rendered <= 0 {
		return 8
	}
	w := rendered - paneBorderChars - panePaddingChars
	if w < 8 {
		w = 8
	}
	return w
}

func (m *Model) paneInnerWidth(rendered int) int {
	return paneContentWidth(rendered)
}

func scrollViewport(vp *viewport.Model, key string) bool {
	switch key {
	case "up", "k":
		vp.LineUp(1)
		return true
	case "down", "j":
		vp.LineDown(1)
		return true
	case "pgdown", "pgdn", "ctrl+d":
		vp.ViewDown()
		return true
	case "pgup", "b", "ctrl+u":
		vp.ViewUp()
		return true
	case "home", "g":
		vp.GotoTop()
		return true
	case "end", "G":
		vp.GotoBottom()
		return true
	}
	return false
}
