package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TaskStatusLabel returns a human-readable project/CRM task status.
func TaskStatusLabel(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		if label := taskStatusFromString(s); label != "" {
			return label
		}
	}
	n := intRawVal(map[string]any{"status": v}, "status")
	switch n {
	case 0:
		return "Not accepted"
	case 1:
		return "Open"
	case 2:
		return "Closed"
	case 3:
		return "Disabled"
	case 4:
		return "Unclassified"
	case 5:
		return "Not in milestone"
	default:
		if s := fmt.Sprint(v); s != "" && s != "<nil>" {
			return s
		}
		return ""
	}
}

func taskStatusFromString(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "open":
		return "Open"
	case "closed":
		return "Closed"
	case "notaccept", "not accept", "not accepted":
		return "Not accepted"
	case "disable", "disabled":
		return "Disabled"
	case "unclassified":
		return "Unclassified"
	case "notinmilestone", "not in milestone":
		return "Not in milestone"
	default:
		return ""
	}
}

// FormatRelativeDeadline formats a deadline as "in 2 hours" or "3 days ago".
func FormatRelativeDeadline(v any) string {
	return FormatRelativeDeadlineAt(v, time.Now())
}

// FormatRelativeDeadlineAt is FormatRelativeDeadline with a fixed clock (for tests).
func FormatRelativeDeadlineAt(v any, now time.Time) string {
	t, ok := parseDeadlineTime(v)
	if !ok {
		return ""
	}
	return relativeTimePhrase(now, t)
}

func relativeTimePhrase(now, target time.Time) string {
	d := target.Sub(now)
	future := d >= 0
	if !future {
		d = -d
	}
	phrase := func(n int, unit string) string {
		if n == 1 {
			unit = strings.TrimSuffix(unit, "s")
		}
		if future {
			return fmt.Sprintf("in %d %s", n, unit)
		}
		return fmt.Sprintf("%d %s ago", n, unit)
	}
	switch {
	case d < time.Minute:
		if future {
			return "in 1 minute"
		}
		return "just now"
	case d < time.Hour:
		return phrase(int(d.Round(time.Minute)/time.Minute), "minutes")
	case d < 24*time.Hour:
		return phrase(int(d.Round(time.Hour)/time.Hour), "hours")
	default:
		return phrase(int(d.Round(24*time.Hour)/(24*time.Hour)), "days")
	}
}

func parseDeadlineTime(v any) (time.Time, bool) {
	if v == nil {
		return time.Time{}, false
	}
	switch x := v.(type) {
	case time.Time:
		return x, true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return time.Time{}, false
		}
		formats := []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05.0000000-07:00",
			"2006-01-02T15:04:05-07:00",
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, layout := range formats {
			if t, err := time.Parse(layout, s); err == nil {
				return t, true
			}
		}
		return time.Time{}, false
	case float64:
		return time.Unix(int64(x), 0), true
	case int64:
		return time.Unix(x, 0), true
	case int:
		return time.Unix(int64(x), 0), true
	default:
		return parseDeadlineTime(fmt.Sprint(v))
	}
}

// TaskResponsibleLabel returns the assignee display name when present.
func TaskResponsibleLabel(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	if name := personName(raw["responsible"]); name != "" {
		return name
	}
	if list, ok := raw["responsibles"].([]any); ok {
		names := make([]string, 0, len(list))
		for _, entry := range list {
			if name := personName(entry); name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			return strings.Join(names, ", ")
		}
	}
	if s := strRaw(raw, "responsible"); s != "" && !looksLikeID(s) {
		return s
	}
	return ""
}

func personName(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	if name := strRaw(m, "displayName"); name != "" {
		if strings.EqualFold(name, "Profile has been removed") {
			return ""
		}
		return name
	}
	first := strRaw(m, "firstName")
	last := strRaw(m, "lastName")
	return strings.TrimSpace(first + " " + last)
}

func looksLikeID(s string) bool {
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return true
	}
	_, err := strconv.Atoi(s)
	return err == nil
}
