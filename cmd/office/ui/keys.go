package ui

import "github.com/eslider/go-onlyoffice/cmd/office/model"

// Action is a keyboard command outcome for the TUI.
type Action int

const (
	ActionNone Action = iota
	ActionMoveUp
	ActionMoveDown
	ActionMoveLeft
	ActionMoveRight
	ActionSort
	ActionToggleSelect
	ActionFocusDetail
	ActionToggleMenuPane
	ActionToggleListPane
	ActionToggleDetailPane
	ActionFilter
	ActionNextPane
	ActionPrevPane
	ActionRefresh
	ActionQuit
)

// KeyAction maps a key string and focused pane to an action.
func KeyAction(key string, pane model.FocusPane) Action {
	switch key {
	case "q", "ctrl+c":
		return ActionQuit
	case "tab":
		if pane != model.FocusPreview {
			return ActionNextPane
		}
	case "shift+tab", "backtab":
		if pane != model.FocusPreview {
			return ActionPrevPane
		}
	case "r":
		return ActionRefresh
	case "f", "/":
		if pane == model.FocusList {
			return ActionFilter
		}
		if pane == model.FocusMenu {
			return ActionFilter
		}
	case "up", "k":
		return ActionMoveUp
	case "down", "j":
		return ActionMoveDown
	case "s":
		if pane == model.FocusList {
			return ActionSort
		}
	case "left", "h":
		return ActionMoveLeft
	case "right", "l":
		return ActionMoveRight
	case " ":
		if pane == model.FocusList {
			return ActionToggleSelect
		}
	case "v", "p":
		if pane == model.FocusList {
			return ActionFocusDetail
		}
	case "alt+1":
		return ActionToggleMenuPane
	case "alt+2":
		return ActionToggleListPane
	case "alt+3":
		return ActionToggleDetailPane
	}
	return ActionNone
}

// LayoutWidthsLegacy is kept for tests that expect the old signature.
func LayoutWidthsLegacy(total int) (menu, list, preview int) {
	pw := LayoutWidths(total, defaultPaneVisibility())
	return pw.Menu, pw.List, pw.Detail
}

// ResolveMoveUp returns the action for upward navigation keys.
func ResolveMoveUp(pane model.FocusPane) Action {
	switch pane {
	case model.FocusMenu, model.FocusList, model.FocusPreview:
		return ActionMoveUp
	default:
		return ActionNone
	}
}
