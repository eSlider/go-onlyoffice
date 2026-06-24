package model

import "fmt"

// ClassifyCalendarRow decides whether an API row is a calendar or an event.
func ClassifyCalendarRow(raw map[string]any) (Kind, string) {
	if raw == nil {
		return KindCalendar, "Calendar"
	}
	if strRaw(raw, "start") != "" || strRaw(raw, "end") != "" {
		return KindEvent, "Event"
	}
	if strRaw(raw, "eventType") != "" {
		return KindEvent, "Event"
	}
	return KindCalendar, "Calendar"
}

// CalendarTypeLabel returns the type column text for a calendar list row.
func CalendarTypeLabel(it Item) string {
	if it.Raw != nil {
		if t := strRaw(it.Raw, "type"); t != "" {
			return t
		}
	}
	_, label := ClassifyCalendarRow(it.Raw)
	return label
}

// FormatCalendarDateTime formats start/end timestamps for the calendar table.
func FormatCalendarDateTime(v any) string {
	t, ok := parseDeadlineTime(v)
	if !ok {
		if s := fmt.Sprint(v); s != "" && s != "<nil>" {
			return s
		}
		return ""
	}
	return t.Format("Jan 2 15:04")
}
