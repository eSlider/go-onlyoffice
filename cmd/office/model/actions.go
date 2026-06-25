package model

// ActionID identifies an operation on a list item.
type ActionID string

const (
	ActionView     ActionID = "view"
	ActionSave     ActionID = "save"
	ActionClose    ActionID = "close"
	ActionDelete   ActionID = "delete"
	ActionRefresh  ActionID = "refresh"
	ActionDownload ActionID = "download"
)

// ItemAction is one CRUD-style operation offered for an item.
type ItemAction struct {
	ID     ActionID
	Label  string
	Danger bool
}

// ActionsFor returns available operations for an item kind (detail pane action bar).
func ActionsFor(kind Kind) []ItemAction {
	switch kind {
	case KindProject:
		return []ItemAction{
			{ID: ActionSave, Label: "Save"},
			{ID: ActionDelete, Label: "Delete", Danger: true},
		}
	case KindTask:
		return []ItemAction{
			{ID: ActionSave, Label: "Save"},
			{ID: ActionClose, Label: "Close"},
		}
	case KindCRMTask:
		return []ItemAction{
			{ID: ActionDelete, Label: "Delete", Danger: true},
		}
	case KindContact:
		return []ItemAction{
			{ID: ActionDelete, Label: "Delete contact", Danger: true},
		}
	case KindOpportunity:
		return []ItemAction{
			{ID: ActionDelete, Label: "Delete deal", Danger: true},
		}
	case KindCase:
		return []ItemAction{
			{ID: ActionDelete, Label: "Delete case", Danger: true},
		}
	case KindMail:
		return []ItemAction{
			{ID: ActionDelete, Label: "Delete message", Danger: true},
		}
	case KindFile:
		return []ItemAction{
			{ID: ActionDownload, Label: "Download"},
			{ID: ActionDelete, Label: "Delete file", Danger: true},
		}
	case KindUser:
		return []ItemAction{
			{ID: ActionSave, Label: "Save"},
		}
	case KindEvent, KindCalendar:
		return nil
	default:
		return nil
	}
}
