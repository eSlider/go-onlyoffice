package model

import "fmt"

// FormFields holds editable/display fields loaded from API detail.
type FormFields struct {
	PrimaryLabel   string
	SecondaryLabel string
	Primary        string
	Secondary      string
	ReadOnly       bool
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
		return FormFields{
			PrimaryLabel: "Name", SecondaryLabel: "Email",
			Primary: strRaw(raw, "displayName"), Secondary: strRaw(raw, "email"),
			ReadOnly: true,
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
			ReadOnly: kind != KindTask && kind != KindProject,
		}
	}
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

// IsDocumentKind is true when the right pane should show file content, not a form.
func IsDocumentKind(kind Kind) bool {
	return kind == KindFile
}
