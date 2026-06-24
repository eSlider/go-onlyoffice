package fetch

import (
	"fmt"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// FieldMap names the JSON keys used when building list items from API rows.
type FieldMap struct {
	IDKey       string
	TitleKey    string
	SubtitleKey string
}

// TaskItemFields is the default field map for project tasks.
var TaskItemFields = FieldMap{IDKey: "id", TitleKey: "title", SubtitleKey: "status"}

// ProjectItemFields is the default field map for projects.
var ProjectItemFields = FieldMap{IDKey: "id", TitleKey: "title", SubtitleKey: "status"}

// ContactItemFields is the default field map for CRM contacts.
var ContactItemFields = FieldMap{IDKey: "id", TitleKey: "displayName", SubtitleKey: "primaryEmail"}

// MailItemFields is the default field map for mail messages.
var MailItemFields = FieldMap{IDKey: "id", TitleKey: "subject", SubtitleKey: "from"}

// ItemsFromMaps converts OnlyOffice list rows into TUI items.
func ItemsFromMaps(rows []map[string]any, kind model.Kind, fields FieldMap) []model.Item {
	out := make([]model.Item, len(rows))
	for i, row := range rows {
		title := str(row, fields.TitleKey)
		if title == "" {
			title = str(row, "title")
		}
		if title == "" {
			title = str(row, "name")
		}
		raw := row
		out[i] = model.Item{
			ID:       idStr(row, fields.IDKey),
			Title:    title,
			Subtitle: str(row, fields.SubtitleKey),
			Kind:     kind,
			Raw:      raw,
		}
	}
	return out
}

func idStr(m map[string]any, key string) string {
	if key == "" {
		key = "id"
	}
	switch v := m[key].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprint(m[key])
	}
}

func str(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	if m[key] == nil {
		return ""
	}
	return fmt.Sprint(m[key])
}
