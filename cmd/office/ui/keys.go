package ui

import "github.com/eslider/go-onlyoffice/cmd/office/model"

// Action is a keyboard command outcome for the TUI.
type Action int

const (
	ActionNone Action = iota
	ActionMoveUp
	ActionMoveDown
	ActionToggleSelect
	ActionOpenPreview
	ActionNextPane
	ActionRefresh
	ActionQuit
	ActionOpenVex
)

// KeyAction maps a key string and focused pane to an action.
func KeyAction(key string, pane model.FocusPane) Action {
	switch key {
	case "q", "ctrl+c":
		return ActionQuit
	case "tab":
		return ActionNextPane
	case "r":
		return ActionRefresh
	case "up", "k":
		return ActionMoveUp
	case "down", "j":
		return ActionMoveDown
	case " ":
		if pane == model.FocusList {
			return ActionToggleSelect
		}
	case "enter":
		if pane == model.FocusList {
			return ActionOpenPreview
		}
	case "v":
		if pane == model.FocusList {
			return ActionOpenVex
		}
	}
	return ActionNone
}

// LayoutWidths splits total terminal width into menu, list, preview columns.
func LayoutWidths(total int) (menu, list, preview int) {
	if total < 80 {
		total = 80
	}
	menu = total / 5
	if menu < 22 {
		menu = 22
	}
	preview = total / 3
	if preview < 28 {
		preview = 28
	}
	list = total - menu - preview - 2
	if list < 24 {
		list = 24
	}
	return menu, list, preview
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
