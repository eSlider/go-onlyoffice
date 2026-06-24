package model

// ActionID identifies an operation on a list item.
type ActionID string

const (
	ActionView     ActionID = "view"
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

// ActionsFor returns available operations for an item kind.
func ActionsFor(kind Kind) []ItemAction {
	switch kind {
	case KindProject:
		return []ItemAction{
			{ID: ActionView, Label: "View details"},
			{ID: ActionDelete, Label: "Delete project", Danger: true},
		}
	case KindTask, KindCRMTask:
		return []ItemAction{
			{ID: ActionView, Label: "View details"},
			{ID: ActionDelete, Label: "Delete task", Danger: true},
		}
	case KindContact:
		return []ItemAction{
			{ID: ActionView, Label: "View details"},
			{ID: ActionDelete, Label: "Delete contact", Danger: true},
		}
	case KindOpportunity:
		return []ItemAction{
			{ID: ActionView, Label: "View details"},
			{ID: ActionDelete, Label: "Delete deal", Danger: true},
		}
	case KindCase:
		return []ItemAction{
			{ID: ActionView, Label: "View details"},
			{ID: ActionDelete, Label: "Delete case", Danger: true},
		}
	case KindMail:
		return []ItemAction{
			{ID: ActionView, Label: "Read message"},
			{ID: ActionDelete, Label: "Delete message", Danger: true},
		}
	case KindFile:
		return []ItemAction{
			{ID: ActionView, Label: "Preview file"},
			{ID: ActionDownload, Label: "Download"},
			{ID: ActionDelete, Label: "Delete file", Danger: true},
		}
	case KindEvent, KindCalendar, KindUser:
		return []ItemAction{{ID: ActionView, Label: "View details"}}
	default:
		return []ItemAction{{ID: ActionView, Label: "View details"}}
	}
}
