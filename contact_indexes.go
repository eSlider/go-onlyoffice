package onlyoffice

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// History entities that OnlyOffice CRM actually accepts for history notes.
// There is NO person/contact history in this API version: POST /api/2.0/crm/history.json
// returns 400 "Value does not fall within the expected range." for entityType
// contact/person/people/client/member. Verified against a live instance (#74).
const (
	HistoryEntityOpportunity = "opportunity"
	HistoryEntityCase        = "case"
)

// IsCompany reports whether a CRM contact row is a company (vs a person).
// The field arrives as JSON bool; be liberal about what we accept.
func IsCompany(person map[string]any) bool {
	b, _ := person["isCompany"].(bool)
	return b
}

// ContactID returns the CRM id of a contact row as a plain string.
func ContactID(row map[string]any) string {
	return fmt.Sprint(row["id"])
}

// BuildContactEmailIndex scans all persons once and maps lowercase email →
// contact id. Use this instead of calling FindPersonByEmail per address:
// the index is O(N) over the whole CRM, the per-address lookup is O(N×M).
func (c *Client) BuildContactEmailIndex(ctx context.Context) (map[string]string, error) {
	all, err := c.ListAllContacts(ctx)
	if err != nil {
		return nil, err
	}
	index := make(map[string]string, len(all)*2)
	for _, person := range all {
		if IsCompany(person) {
			continue
		}
		id := ContactID(person)
		for _, row := range ContactInfoRows(person) {
			if NormalizeContactInfoType(fmt.Sprint(row["infoType"])) != "email" {
				continue
			}
			email := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["data"])))
			if email != "" && email != "<nil>" {
				index[email] = id
			}
		}
	}
	return index, nil
}

// BuildPersonOpportunityIndex maps every opportunity member's contact id to a
// deterministic representative opportunity: the one with the lowest numeric id.
// OnlyOffice has no person-level history, so notes for a person go on their
// deal — this index answers "which deal" in one pass.
func (c *Client) BuildPersonOpportunityIndex(ctx context.Context) (map[string]string, error) {
	opps, err := c.ListAllOpportunities(ctx)
	if err != nil {
		return nil, err
	}
	index := map[string]string{}
	for _, opp := range opps {
		oppID := ContactID(opp)
		for _, member := range OpportunityMembers(opp) {
			pid := ContactID(member)
			if cur, ok := index[pid]; !ok || NumericIDLess(oppID, cur) {
				index[pid] = oppID
			}
		}
	}
	return index, nil
}

// NumericIDLess compares two string ids numerically when possible, falling
// back to lexicographic order so results stay deterministic either way.
func NumericIDLess(a, b string) bool {
	na, errA := strconv.Atoi(strings.TrimSpace(a))
	nb, errB := strconv.Atoi(strings.TrimSpace(b))
	if errA == nil && errB == nil && na != nb {
		return na < nb
	}
	return a < b
}

// SortIDs orders id strings deterministically (numeric first, then lexical).
func SortIDs(ids []string) {
	sort.Strings(ids)
	sort.SliceStable(ids, func(i, j int) bool { return NumericIDLess(ids[i], ids[j]) })
}
