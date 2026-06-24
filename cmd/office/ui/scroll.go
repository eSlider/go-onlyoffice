package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func (m *Model) scrollFocusedPane(key string) bool {
	switch m.focus {
	case model.FocusMenu:
		return scrollViewport(&m.menuVP, key)
	case model.FocusList:
		switch key {
		case "pgdown", "pgdn", "f", "ctrl+d":
			m.listTable.PageScroll(1)
			return true
		case "pgup", "b", "ctrl+u":
			m.listTable.PageScroll(-1)
			return true
		}
		return false
	case model.FocusPreview:
		if m.detail.Zone() != detailZoneContent || m.detail.mode != detailDocument {
			return false
		}
		return scrollViewport(&m.detail.docVP, key)
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

func (m *Model) paneInnerWidth(outer int) int {
	if outer <= 0 {
		return 8
	}
	w := outer - 2 // border only; content fills inner box
	if w < 8 {
		w = 8
	}
	return w
}

func scrollViewport(vp *viewport.Model, key string) bool {
	switch key {
	case "pgdown", "pgdn", "f", "ctrl+d":
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
