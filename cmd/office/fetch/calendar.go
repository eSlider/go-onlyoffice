package fetch

import (
	"context"
	"fmt"
	"time"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// CalendarItemFields maps calendar API rows to list items.
var CalendarItemFields = FieldMap{IDKey: "objectId", TitleKey: "title"}

// CalendarItemsFromRows converts API calendar/event rows into list items.
func CalendarItemsFromRows(rows []map[string]any) []model.Item {
	out := make([]model.Item, 0, len(rows))
	for _, row := range rows {
		kind, typeLabel := model.ClassifyCalendarRow(row)
		title := str(row, "title")
		if title == "" {
			title = str(row, "name")
		}
		if title == "" {
			title = "(untitled)"
		}
		raw := make(map[string]any, len(row)+1)
		for k, v := range row {
			raw[k] = v
		}
		raw["type"] = typeLabel
		out = append(out, model.Item{
			ID:    idStr(row, "objectId"),
			Title: title,
			Kind:  kind,
			Raw:   raw,
		})
	}
	return out
}

// ListCalendar returns calendars and events for the upcoming date window.
func (l *Loader) ListCalendar(ctx context.Context) ([]model.Item, error) {
	if l == nil || l.Client == nil {
		return nil, fmt.Errorf("fetch: client is nil")
	}
	start := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	end := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	rows, err := l.Client.ListCalendars(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return CalendarItemsFromRows(rows), nil
}
