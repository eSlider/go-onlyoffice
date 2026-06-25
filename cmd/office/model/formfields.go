package model

import "fmt"

// FormFields holds editable/display fields loaded from API detail.
type FormFields struct {
	PrimaryLabel   string
	SecondaryLabel string
	Primary        string
	Secondary      string
	ReadOnly       bool
	HasStatus      bool
	Status         ProjectLifecycle
	HasTaskStatus  bool
	TaskStatus     TaskLifecycle
	HasResponsible bool
	ResponsibleID  string
	UserChoices    []UserOption
	ProjectTitle   string
	TimingSummary  string
	HasUserEdit    bool
	UserEnabled    bool
	UserACL        UserACLState
	GroupsText     string
	UserPassword   string
}

// KindHeading returns a short label for the detail pane header.
func KindHeading(kind Kind, id string) string {
	return fmt.Sprintf("%s %s", kindLabel(kind), id)
}

func kindLabel(kind Kind) string {
	switch kind {
	case KindProject:
		return "Project"
	case KindTask:
		return "Task"
	case KindCRMTask:
		return "CRM task"
	case KindContact:
		return "Contact"
	case KindOpportunity:
		return "Deal"
	case KindCase:
		return "Case"
	case KindMail:
		return "Mail"
	case KindEvent:
		return "Event"
	case KindCalendar:
		return "Calendar"
	case KindFile:
		return "Document"
	case KindUser:
		return "User"
	default:
		return string(kind)
	}
}

// FormFieldsFromRaw maps API detail JSON to form fields for the right pane.
func FormFieldsFromRaw(kind Kind, raw map[string]any) FormFields {
	switch kind {
	case KindMail:
		body := strRaw(raw, "body")
		if body == "" {
			body = strRaw(raw, "htmlBody")
		}
		return FormFields{
			PrimaryLabel: "Subject", SecondaryLabel: "Body",
			Primary: strRaw(raw, "subject"), Secondary: body,
			ReadOnly: true,
		}
	case KindUser:
		acl := UserACLFromRaw(raw)
		return FormFields{
			HasUserEdit: true,
			ReadOnly:    false,
			UserEnabled: UserIsEnabled(raw),
			UserACL:     acl,
			GroupsText:  UserGroupsText(raw),
		}
	case KindContact:
		name := strRaw(raw, "displayName")
		if name == "" {
			name = strRaw(raw, "title")
		}
		return FormFields{
			PrimaryLabel: "Name", SecondaryLabel: "About",
			Primary: name, Secondary: strRaw(raw, "about"),
			ReadOnly: true,
		}
	case KindEvent, KindCalendar:
		return FormFields{
			PrimaryLabel: "Title", SecondaryLabel: "Description",
			Primary: strRaw(raw, "title"), Secondary: strRaw(raw, "description"),
			ReadOnly: true,
		}
	case KindProject:
		title := strRaw(raw, "title")
		if title == "" {
			title = strRaw(raw, "name")
		}
		return FormFields{
			PrimaryLabel:   "Title",
			SecondaryLabel: "Description",
			Primary:        title,
			Secondary:      strRaw(raw, "description"),
			ReadOnly:       false,
			HasStatus:      true,
			Status:         ProjectStatusFromAny(raw["status"]),
			ResponsibleID:  ResponsibleIDFromRaw(raw),
		}
	case KindTask:
		return FormFields{
			PrimaryLabel:   "Title",
			SecondaryLabel: "Description",
			Primary:        strRaw(raw, "title"),
			Secondary:      strRaw(raw, "description"),
			ReadOnly:       false,
			HasTaskStatus:  true,
			TaskStatus:     TaskStatusFromAny(raw["status"]),
			HasResponsible: true,
			ResponsibleID:  TaskResponsibleIDFromRaw(raw),
			ProjectTitle:   TaskProjectTitle(raw),
			TimingSummary:  TaskTimingSummary(raw),
		}
	default:
		title := strRaw(raw, "title")
		if title == "" {
			title = strRaw(raw, "name")
		}
		if title == "" {
			title = strRaw(raw, "subject")
		}
		return FormFields{
			PrimaryLabel: "Title", SecondaryLabel: "Description",
			Primary: title, Secondary: strRaw(raw, "description"),
			ReadOnly: true,
		}
	}
}

// TaskResponsibleIDFromRaw returns the first assignee user id from task detail.
func TaskResponsibleIDFromRaw(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	if ids, ok := raw["responsibleIds"].([]any); ok && len(ids) > 0 {
		return strRaw(map[string]any{"id": ids[0]}, "id")
	}
	if list, ok := raw["responsibles"].([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			return strRaw(m, "id")
		}
	}
	return ""
}

// TaskProjectTitle returns the owning project title when present.
func TaskProjectTitle(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	if po, ok := raw["projectOwner"].(map[string]any); ok {
		if t := strRaw(po, "title"); t != "" {
			return t
		}
	}
	return strRaw(raw, "projectTitle")
}

// TaskTimingSummary formats start and deadline for the detail form.
func TaskTimingSummary(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	start := formatTaskDateLabel(raw["startDate"])
	deadline := formatTaskDateLabel(raw["deadline"])
	switch {
	case start != "" && deadline != "":
		return start + " → " + deadline
	case deadline != "":
		return "Due " + deadline
	case start != "":
		return "From " + start
	default:
		return ""
	}
}

func formatTaskDateLabel(v any) string {
	t, ok := parseDeadlineTime(v)
	if !ok {
		s := strRaw(map[string]any{"v": v}, "v")
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
	return t.Format("2006-01-02 15:04")
}

func strRaw(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	if m[key] == nil {
		return ""
	}
	return fmt.Sprint(m[key])
}

// ResponsibleIDFromRaw extracts the project responsible user id from API detail.
func ResponsibleIDFromRaw(raw map[string]any) string {
	if id := strRaw(raw, "responsibleId"); id != "" {
		return id
	}
	resp, ok := raw["responsible"].(map[string]any)
	if !ok {
		return ""
	}
	return strRaw(resp, "id")
}

// IsDocumentKind is true when the right pane should show rendered preview content, not an editable form.
func IsDocumentKind(kind Kind) bool {
	return kind == KindFile || kind == KindMail
}
