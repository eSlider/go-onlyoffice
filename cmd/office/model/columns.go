package model

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// TableColumn is one column in the center data table.
type TableColumn struct {
	Key   string
	Title string
	Width int
}

var subjectExtraKeys = map[Subject][]string{
	SubjectTasks:         {"status", "deadline", "responsible"},
	SubjectCalendars:     {"description"},
	SubjectEvents:        {"start", "end"},
	SubjectContacts:      {"primaryEmail", "displayName"},
	SubjectPersons:       {"primaryEmail", "displayName"},
	SubjectCompanies:     {"primaryEmail", "displayName"},
	SubjectOpportunities: {"stage", "bidValue", "bidCurrency"},
	SubjectCases:         {"status"},
	SubjectCRMTasks:      {"status", "deadline"},
	SubjectMailInbox:     {"from", "date"},
	SubjectMailSent:      {"to", "date"},
	SubjectMailDrafts:    {"to", "date"},
	SubjectMailTrash:     {"from", "date"},
	SubjectMailSpam:      {"from", "date"},
}

// BuildColumns derives table columns from the list subject and item payloads.
func BuildColumns(subject Subject, items []Item) []TableColumn {
	if subject == SubjectProjects {
		return buildProjectColumns(items)
	}
	if subject == SubjectUsers {
		return buildUserColumns(items)
	}
	if subject == SubjectTasks {
		return buildTaskColumns(items)
	}
	if subject == SubjectCalendar {
		return buildCalendarColumns(items)
	}
	cols := []TableColumn{
		{Key: "_sel", Title: "✓", Width: 3},
		{Key: "id", Title: "ID", Width: 10},
		{Key: "title", Title: "Title", Width: 28},
	}
	if hasSubtitle(items) {
		cols = append(cols, TableColumn{Key: "subtitle", Title: "Subtitle", Width: 22})
	}

	seen := map[string]bool{"_sel": true, "id": true, "title": true, "subtitle": true}
	for _, key := range subjectExtraKeys[subject] {
		if seen[key] || !columnHasData(items, key) {
			continue
		}
		seen[key] = true
		cols = append(cols, TableColumn{Key: key, Title: titleLabel(key), Width: defaultWidth(key)})
	}

	discovered := discoverRawKeys(items, seen, 8)
	for _, key := range discovered {
		cols = append(cols, TableColumn{Key: key, Title: titleLabel(key), Width: defaultWidth(key)})
	}

	sizeColumns(cols, items)
	return cols
}

func buildCalendarColumns(items []Item) []TableColumn {
	cols := []TableColumn{
		{Key: "_sel", Title: "✓", Width: 3},
		{Key: "id", Title: "ID", Width: 10},
		{Key: "type", Title: "Type", Width: 10},
		{Key: "title", Title: "Title", Width: 28},
	}
	for _, key := range []string{"start", "end"} {
		if columnHasData(items, key) {
			cols = append(cols, TableColumn{Key: key, Title: titleLabel(key), Width: defaultWidth(key)})
		}
	}
	sizeColumns(cols, items)
	return cols
}

func buildTaskColumns(items []Item) []TableColumn {
	cols := []TableColumn{
		{Key: "_sel", Title: "✓", Width: 3},
		{Key: "id", Title: "ID", Width: 8},
		{Key: "title", Title: "Title", Width: 28},
		{Key: "status", Title: "Status", Width: 10},
	}
	for _, key := range []string{"deadline", "responsible"} {
		if columnHasData(items, key) {
			cols = append(cols, TableColumn{Key: key, Title: titleLabel(key), Width: defaultWidth(key)})
		}
	}
	sizeColumns(cols, items)
	return cols
}

func buildUserColumns(items []Item) []TableColumn {
	_ = items
	return []TableColumn{
		{Key: "_sel", Title: "✓", Width: 3},
		{Key: "userName", Title: "User", Width: 14},
		{Key: "registration", Title: "Registered", Width: 12},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "email", Title: "Email", Width: 20},
	}
}

func buildProjectColumns(items []Item) []TableColumn {
	_ = items
	return []TableColumn{
		{Key: "_sel", Title: "✓", Width: 3},
		{Key: "id", Title: "ID", Width: 8},
		{Key: "status", Title: "●", Width: 10},
		{Key: "title", Title: "Title", Width: 16},
		{Key: "tasks", Title: "Tasks", Width: 11},
		{Key: "documents", Title: "Docs", Width: 7},
		{Key: "users", Title: "Users", Width: 7},
	}
}

// CellText returns the display string for one table cell.
func CellText(it Item, key string) string {
	switch key {
	case "_sel":
		if it.Selected {
			return "●"
		}
		return "○"
	case "id":
		return it.ID
	case "title":
		return it.Title
	case "type":
		return CalendarTypeLabel(it)
	case "start", "end":
		if it.Raw == nil {
			return ""
		}
		if formatted := FormatCalendarDateTime(it.Raw[key]); formatted != "" {
			return formatted
		}
		return formatAny(it.Raw[key])
	case "status":
		switch it.Kind {
		case KindProject:
			return ProjectStatusCell(it.Raw)
		case KindUser:
			return UserStatusLabel(it.Raw)
		case KindTask, KindCRMTask:
			if it.Raw == nil {
				return ""
			}
			return TaskStatusLabel(it.Raw["status"])
		default:
			if it.Raw == nil {
				return ""
			}
			return formatAny(it.Raw["status"])
		}
	case "deadline":
		if it.Raw == nil {
			return ""
		}
		if rel := FormatRelativeDeadline(it.Raw["deadline"]); rel != "" {
			return rel
		}
		return formatAny(it.Raw["deadline"])
	case "responsible":
		if it.Kind == KindTask || it.Kind == KindCRMTask {
			return TaskResponsibleLabel(it.Raw)
		}
		if it.Raw == nil {
			return ""
		}
		return formatAny(it.Raw["responsible"])
	case "subtitle":
		return it.Subtitle
	case "userName":
		if it.Raw != nil {
			if u := strRaw(it.Raw, "userName"); u != "" {
				return u
			}
		}
		return it.Title
	case "registration":
		return FormatUserRegistration(it.Raw)
	case "email":
		if it.Raw != nil {
			return strRaw(it.Raw, "email")
		}
		return it.Subtitle
	case "tasks":
		return formatProjectTasks(it.Raw)
	case "documents":
		return intRaw(it.Raw, "documentsCount")
	case "users":
		return intRaw(it.Raw, "participantCount")
	case "kind":
		return string(it.Kind)
	default:
		if it.Raw == nil {
			return ""
		}
		return formatAny(it.Raw[key])
	}
}

func formatProjectTasks(raw map[string]any) string {
	if raw == nil {
		return "0/0"
	}
	open := intRawVal(raw, "taskCount")
	total := intRawVal(raw, "taskCountTotal")
	closed := total - open
	if closed < 0 {
		closed = 0
	}
	return fmt.Sprintf("%d/%d", open, closed)
}

func intRaw(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	v := intRawVal(raw, key)
	if v == 0 {
		if _, ok := raw[key]; !ok {
			return ""
		}
	}
	return fmt.Sprintf("%d", v)
}

func intRawVal(raw map[string]any, key string) int {
	if raw == nil {
		return 0
	}
	switch v := raw[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		if raw[key] == nil {
			return 0
		}
		n, _ := strconv.Atoi(fmt.Sprint(raw[key]))
		return n
	}
}

func hasSubtitle(items []Item) bool {
	for _, it := range items {
		if it.Subtitle != "" {
			return true
		}
	}
	return false
}

func columnHasData(items []Item, key string) bool {
	for _, it := range items {
		if CellText(it, key) != "" {
			return true
		}
	}
	return false
}

func discoverRawKeys(items []Item, seen map[string]bool, limit int) []string {
	counts := map[string]int{}
	for _, it := range items {
		if it.Raw == nil {
			continue
		}
		for k := range it.Raw {
			if seen[k] || k == "id" || k == "title" {
				continue
			}
			counts[k]++
		}
	}
	type kv struct {
		k string
		n int
	}
	var ranked []kv
	for k, n := range counts {
		ranked = append(ranked, kv{k, n})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].n == ranked[j].n {
			return ranked[i].k < ranked[j].k
		}
		return ranked[i].n > ranked[j].n
	})
	out := make([]string, 0, limit)
	for _, r := range ranked {
		if len(out) >= limit {
			break
		}
		out = append(out, r.k)
	}
	return out
}

func titleLabel(key string) string {
	if key == "" {
		return ""
	}
	parts := strings.Split(key, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func defaultWidth(key string) int {
	switch key {
	case "id", "status", "stage":
		return 10
	case "start", "end", "deadline", "date":
		return 20
	case "from", "to", "email", "primaryEmail":
		return 22
	case "description", "displayName":
		return 24
	default:
		return 14
	}
}

func sizeColumns(cols []TableColumn, items []Item) {
	for i := range cols {
		maxW := runeLen(cols[i].Title)
		for _, it := range items {
			n := runeLen(CellText(it, cols[i].Key))
			if n > maxW {
				maxW = n
			}
		}
		maxW += 2
		if maxW < 4 {
			maxW = 4
		}
		if maxW > 36 {
			maxW = 36
		}
		cols[i].Width = maxW
	}
}

func runeLen(s string) int {
	return len([]rune(s))
}

func formatAny(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "yes"
		}
		return "no"
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	default:
		return fmt.Sprint(v)
	}
}
