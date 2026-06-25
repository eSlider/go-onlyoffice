package model

import "time"

const (
	EmployeeStatusActive     = 1
	EmployeeStatusTerminated = 2
	EmployeeStatusPending    = 4
)

// UserIsEnabled reports whether the portal user account is active.
func UserIsEnabled(raw map[string]any) bool {
	if raw == nil {
		return true
	}
	if t, ok := raw["terminated"].(bool); ok && t {
		return false
	}
	switch intRawVal(raw, "status") {
	case EmployeeStatusTerminated:
		return false
	case EmployeeStatusPending:
		return false
	default:
		return true
	}
}

// UserStatusLabel returns a short list-cell label for account state.
func UserStatusLabel(raw map[string]any) string {
	if !UserIsEnabled(raw) {
		return "Disabled"
	}
	if intRawVal(raw, "status") == EmployeeStatusPending {
		return "Pending"
	}
	return "Active"
}

// UserGroupsText formats group membership for the detail form.
func UserGroupsText(raw map[string]any) string {
	if raw == nil {
		return "—"
	}
	list, ok := raw["groups"].([]any)
	if !ok || len(list) == 0 {
		return "—"
	}
	names := make([]string, 0, len(list))
	for _, g := range list {
		m, ok := g.(map[string]any)
		if !ok {
			continue
		}
		name := strRaw(m, "name")
		if name == "" {
			name = strRaw(m, "title")
		}
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "—"
	}
	return stringsJoin(names, ", ")
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

// FormatUserRegistration formats registration or work-from date for list cells.
func FormatUserRegistration(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	for _, key := range []string{"registrationDate", "workFrom"} {
		if s := formatUserDate(raw[key]); s != "" {
			return s
		}
	}
	return ""
}

func formatUserDate(v any) string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return ""
		}
		if parsed, err := time.Parse("2006-01-02T15:04:05.0000000-07:00", t); err == nil {
			return parsed.Format("2006-01-02")
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed.Format("2006-01-02")
		}
		if len(t) >= 10 {
			return t[:10]
		}
		return t
	default:
		s := strRaw(map[string]any{"v": v}, "v")
		if s == "" || s == "<nil>" {
			return ""
		}
		return formatUserDate(s)
	}
}
