package onlyoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListCalendars returns calendars/events for the given date span (YYYY-MM-DD).
// Empty start/end fall back to a wide default window (2020-01-01..2030-12-31),
// matching the Python reference implementation.
func (c *Client) ListCalendars(ctx context.Context, start, end string) ([]map[string]any, error) {
	if start == "" {
		start = "2020-01-01"
	}
	if end == "" {
		end = "2030-12-31"
	}
	path := fmt.Sprintf("/api/2.0/calendar/calendars/%s/%s.json", url.PathEscape(start), url.PathEscape(end))
	return c.ResponseArray(ctx, path)
}

// ListEvents is an alias for ListCalendars; retained for the Python-port CLI.
func (c *Client) ListEvents(ctx context.Context, start, end string) ([]map[string]any, error) {
	return c.ListCalendars(ctx, start, end)
}

// AddEvent creates a simple one-shot calendar event. When calendarID is empty
// the configured default (SetDefaults or GetEnvironmentDefaults) is used.
// Start/end timestamps are forwarded verbatim (OnlyOffice accepts ISO 8601).
//
// The underlying API returns an array wrapper — this helper unwraps the first
// element for convenience.
func (c *Client) AddEvent(ctx context.Context, calendarID, title, start, end, description string, allDay bool) (map[string]any, error) {
	if calendarID == "" {
		calendarID = c.defaults.CalendarID
	}
	if calendarID == "" {
		return nil, fmt.Errorf("AddEvent: calendarID is required (pass explicitly or set via SetDefaults)")
	}
	fields := url.Values{}
	fields.Set("name", title)
	fields.Set("description", description)
	fields.Set("startDate", start)
	fields.Set("endDate", end)
	fields.Set("repeatType", "")
	fields.Set("alertType", "0")
	fields.Set("isAllDayLong", strconv.FormatBool(allDay))
	raw, err := c.postForm(ctx, fmt.Sprintf("/api/2.0/calendar/%s/event.json", url.PathEscape(calendarID)), fields)
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	var arr []map[string]any
	if err := json.Unmarshal(resp, &arr); err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("AddEvent: empty response")
	}
	return arr[0], nil
}

// DeleteEvent removes a single calendar event by its event ID.
func (c *Client) DeleteEvent(ctx context.Context, eventID string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/calendar/events/%s.json", url.PathEscape(eventID)))
}
