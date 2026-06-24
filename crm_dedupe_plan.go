package onlyoffice

import (
	"fmt"
	"sort"
	"strings"
)

// PickCanonicalID returns the lowest non-zero id, or 0 when empty.
func PickCanonicalID(ids []int64) int64 {
	if len(ids) == 0 {
		return 0
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids[0]
}

func rowID(m map[string]any) int64 {
	return flexInt(m["id"])
}

func rowDisplayName(m map[string]any) string {
	if v := strings.TrimSpace(fmt.Sprint(m["displayName"])); v != "" && v != "<nil>" {
		return v
	}
	if v := strings.TrimSpace(fmt.Sprint(m["companyName"])); v != "" && v != "<nil>" {
		return v
	}
	first := strings.TrimSpace(fmt.Sprint(m["firstName"]))
	last := strings.TrimSpace(fmt.Sprint(m["lastName"]))
	return strings.TrimSpace(first + " " + last)
}

// GroupCompaniesByName buckets company contacts by normalized display name.
func GroupCompaniesByName(items []map[string]any) map[string][]map[string]any {
	out := make(map[string][]map[string]any)
	for _, row := range items {
		if !isCompany(row) {
			continue
		}
		key := CompanyGroupingKey(rowDisplayName(row))
		if key == "" {
			continue
		}
		out[key] = append(out[key], row)
	}
	return out
}

// GroupPersonsByKey buckets person contacts by normalized first+last.
func GroupPersonsByKey(items []map[string]any) map[string][]map[string]any {
	out := make(map[string][]map[string]any)
	for _, row := range items {
		if isCompany(row) {
			continue
		}
		key := NormalizePersonKey(fmt.Sprint(row["firstName"]), fmt.Sprint(row["lastName"]))
		if strings.TrimSpace(key) == "" {
			continue
		}
		out[key] = append(out[key], row)
	}
	return out
}

// DealTitleKey returns the grouping key for an opportunity title.
func DealTitleKey(title string, ignoreCompanySuffix bool) string {
	title = strings.TrimSpace(title)
	if ignoreCompanySuffix {
		return NormalizeOpportunityTitle(StripCompanySuffix(title))
	}
	if i := strings.LastIndex(title, " @ "); i >= 0 {
		pos := strings.TrimSpace(title[:i])
		co := StripSloganSuffix(strings.TrimSpace(title[i+len(" @ "):]))
		if pos == "" && co != "" {
			return NormalizeCompanyName(co)
		}
		if pos != "" && co != "" {
			return collapseKey(pos + " @ " + co)
		}
	}
	return NormalizeCompanyName(StripSloganSuffix(title))
}

// GroupOpportunitiesByTitle buckets deals by title key.
func GroupOpportunitiesByTitle(items []map[string]any, ignoreCompanySuffix bool) map[string][]map[string]any {
	out := make(map[string][]map[string]any)
	for _, row := range items {
		title := fmt.Sprint(row["title"])
		key := DealTitleKey(title, ignoreCompanySuffix)
		if key == "" {
			continue
		}
		out[key] = append(out[key], row)
	}
	return out
}

// MergePlan lists secondary ids to merge into primary.
type MergePlan struct {
	Primary   int64
	Secondary []int64
}

// BuildMergePlans creates merge plans from duplicate groups (lowest id wins).
func BuildMergePlans(groups map[string][]map[string]any) []MergePlan {
	var plans []MergePlan
	for _, rows := range groups {
		if len(rows) < 2 {
			continue
		}
		ids := make([]int64, len(rows))
		for i, row := range rows {
			ids[i] = rowID(row)
		}
		primary := PickCanonicalID(ids)
		var secondary []int64
		for _, id := range ids {
			if id != primary {
				secondary = append(secondary, id)
			}
		}
		plans = append(plans, MergePlan{Primary: primary, Secondary: secondary})
	}
	return plans
}

// DedupeMemberIDs returns duplicate member ids to remove (keep first occurrence).
func DedupeMemberIDs(ids []int64) []int64 {
	seen := make(map[int64]int)
	var remove []int64
	for _, id := range ids {
		seen[id]++
		if seen[id] > 1 {
			remove = append(remove, id)
		}
	}
	return remove
}

// DedupeMembersByDisplayName returns member ids to remove when the same
// displayName appears with different ids (keep lowest id per name).
func DedupeMembersByDisplayName(members []map[string]any) []int64 {
	type slot struct {
		id int64
	}
	byName := make(map[string][]slot)
	for _, m := range members {
		key := MemberDisplayKey(rowDisplayName(m))
		if key == "" {
			continue
		}
		byName[key] = append(byName[key], slot{id: rowID(m)})
	}
	var remove []int64
	for _, slots := range byName {
		if len(slots) < 2 {
			continue
		}
		ids := make([]int64, len(slots))
		for i, s := range slots {
			ids[i] = s.id
		}
		keep := PickCanonicalID(ids)
		for _, s := range slots {
			if s.id != keep {
				remove = append(remove, s.id)
			}
		}
	}
	return remove
}

// GroupContactInfoRows returns info row ids to delete (duplicates by type+value).
func GroupContactInfoRows(rows []map[string]any) []int64 {
	type rowSlot struct {
		id        int64
		isPrimary bool
	}
	byKey := make(map[string][]rowSlot)
	for _, row := range rows {
		infoType := fmt.Sprint(row["infoType"])
		value := fmt.Sprint(row["data"])
		if value == "" || value == "<nil>" {
			value = fmt.Sprint(row["value"])
		}
		key := ContactInfoKey(infoType, value)
		if key == "|" || value == "" || value == "<nil>" {
			continue
		}
		primary, _ := row["isPrimary"].(bool)
		byKey[key] = append(byKey[key], rowSlot{id: rowID(row), isPrimary: primary})
	}
	var remove []int64
	for _, slots := range byKey {
		if len(slots) < 2 {
			continue
		}
		keep := int64(0)
		for _, s := range slots {
			if s.isPrimary {
				keep = s.id
				break
			}
		}
		if keep == 0 {
			ids := make([]int64, len(slots))
			for i, s := range slots {
				ids[i] = s.id
			}
			keep = PickCanonicalID(ids)
		}
		for _, s := range slots {
			if s.id != keep {
				remove = append(remove, s.id)
			}
		}
	}
	return remove
}

// GroupCompanyPersons groups persons by company id and person name key.
func GroupCompanyPersons(persons []map[string]any) map[int64]map[string][]map[string]any {
	out := make(map[int64]map[string][]map[string]any)
	for _, row := range persons {
		if isCompany(row) {
			continue
		}
		cid := flexInt(row["companyId"])
		if cid == 0 {
			continue
		}
		key := NormalizePersonKey(fmt.Sprint(row["firstName"]), fmt.Sprint(row["lastName"]))
		if strings.TrimSpace(key) == "" {
			continue
		}
		if out[cid] == nil {
			out[cid] = make(map[string][]map[string]any)
		}
		out[cid][key] = append(out[cid][key], row)
	}
	return out
}
