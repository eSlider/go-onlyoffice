package model

import (
	"strings"
)

// FilterItems returns items whose searchable text contains query (case-insensitive).
func FilterItems(items []Item, query string) []Item {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		out := make([]Item, len(items))
		copy(out, items)
		return out
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if strings.Contains(strings.ToLower(ItemSearchText(it)), q) {
			out = append(out, it)
		}
	}
	return out
}

// ItemSearchText joins item fields used for list filtering.
func ItemSearchText(it Item) string {
	var parts []string
	parts = append(parts, it.ID, it.Title, it.Subtitle, string(it.Kind))
	if it.Raw != nil {
		for _, v := range it.Raw {
			parts = append(parts, formatAny(v))
		}
	}
	return strings.Join(parts, " ")
}
