package onlyoffice

import (
	"context"
	"fmt"
	"strconv"
)

// DedupeResult summarizes a cleanup pass.
type DedupeResult struct {
	Groups  int      `json:"groups"`
	Merged  int      `json:"merged"`
	Deleted int      `json:"deleted"`
	Renamed int      `json:"renamed"`
	Removed int      `json:"removed"`
	Errors  []string `json:"errors,omitempty"`
}

func (r *DedupeResult) addErr(err error) {
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
	}
}

type crmDedupeClient interface {
	ListAllContacts(ctx context.Context) ([]map[string]any, error)
	ListAllOpportunities(ctx context.Context) ([]map[string]any, error)
	GetContact(ctx context.Context, contactID string) (map[string]any, error)
	GetOpportunity(ctx context.Context, id string) (map[string]any, error)
	MergeContacts(ctx context.Context, primaryID, secondaryID string) (map[string]any, error)
	DeleteOpportunity(ctx context.Context, id string) (map[string]any, error)
	DeleteContactInfo(ctx context.Context, contactID, dataID string) (map[string]any, error)
	ListCompanyPersons(ctx context.Context, companyID string) ([]map[string]any, error)
	AddOpportunityMember(ctx context.Context, oppID, contactID string) (map[string]any, error)
	RemoveOpportunityMember(ctx context.Context, oppID, contactID string) (map[string]any, error)
	UpdateOpportunityTitle(ctx context.Context, id, newTitle string) (map[string]any, error)
}

func executeMergePlans(ctx context.Context, client crmDedupeClient, plans []MergePlan, res *DedupeResult) {
	for _, plan := range plans {
		res.Groups++
		for _, sec := range plan.Secondary {
			_, err := client.MergeContacts(ctx, strconv.FormatInt(plan.Primary, 10), strconv.FormatInt(sec, 10))
			if err != nil {
				res.addErr(fmt.Errorf("merge %d into %d: %w", sec, plan.Primary, err))
				continue
			}
			res.Merged++
		}
	}
}

// DedupeCompanies merges duplicate company contacts by normalized name.
func DedupeCompanies(ctx context.Context, client crmDedupeClient) (DedupeResult, error) {
	var res DedupeResult
	items, err := client.ListAllContacts(ctx)
	if err != nil {
		return res, err
	}
	plans := BuildMergePlans(GroupCompaniesByName(items))
	executeMergePlans(ctx, client, plans, &res)
	return res, nil
}

// DedupePersons merges duplicate person contacts by normalized first+last.
func DedupePersons(ctx context.Context, client crmDedupeClient) (DedupeResult, error) {
	var res DedupeResult
	items, err := client.ListAllContacts(ctx)
	if err != nil {
		return res, err
	}
	plans := BuildMergePlans(GroupPersonsByKey(items))
	executeMergePlans(ctx, client, plans, &res)
	return res, nil
}

// DedupeCompanyPersons merges same-name persons within each company.
func DedupeCompanyPersons(ctx context.Context, client crmDedupeClient) (DedupeResult, error) {
	var res DedupeResult
	items, err := client.ListAllContacts(ctx)
	if err != nil {
		return res, err
	}
	grouped := GroupCompanyPersons(items)
	for companyID, byName := range grouped {
		plans := BuildMergePlans(byName)
		if len(plans) == 0 {
			continue
		}
		_ = companyID
		executeMergePlans(ctx, client, plans, &res)
	}
	return res, nil
}

// DedupeContactInfo removes duplicate email/phone/etc rows on all contacts.
func DedupeContactInfo(ctx context.Context, client crmDedupeClient) (DedupeResult, error) {
	var res DedupeResult
	items, err := client.ListAllContacts(ctx)
	if err != nil {
		return res, err
	}
	for _, row := range items {
		cid := strconv.FormatInt(rowID(row), 10)
		contact, err := client.GetContact(ctx, cid)
		if err != nil {
			res.addErr(err)
			continue
		}
		rows := ContactInfoRows(contact)
		for _, dataID := range GroupContactInfoRows(rows) {
			_, err := client.DeleteContactInfo(ctx, cid, strconv.FormatInt(dataID, 10))
			if err != nil {
				res.addErr(err)
				continue
			}
			res.Removed++
		}
	}
	return res, nil
}

// DedupeOpportunities merges duplicate deals by title; relinks members first.
func DedupeOpportunities(ctx context.Context, client crmDedupeClient, ignoreCompanySuffix bool) (DedupeResult, error) {
	var res DedupeResult
	items, err := client.ListAllOpportunities(ctx)
	if err != nil {
		return res, err
	}
	groups := GroupOpportunitiesByTitle(items, ignoreCompanySuffix)
	for _, rows := range groups {
		if len(rows) < 2 {
			continue
		}
		res.Groups++
		ids := make([]int64, len(rows))
		for i, row := range rows {
			ids[i] = rowID(row)
		}
		primary := PickCanonicalID(ids)
		primaryID := strconv.FormatInt(primary, 10)
		for _, row := range rows {
			sec := rowID(row)
			if sec == primary {
				continue
			}
			secID := strconv.FormatInt(sec, 10)
			opp, err := client.GetOpportunity(ctx, secID)
			if err != nil {
				res.addErr(err)
				continue
			}
			for _, m := range OpportunityMembers(opp) {
				mid := strconv.FormatInt(rowID(m), 10)
				_, _ = client.AddOpportunityMember(ctx, primaryID, mid)
			}
			if _, err := client.DeleteOpportunity(ctx, secID); err != nil {
				res.addErr(err)
				continue
			}
			res.Deleted++
		}
	}
	return res, nil
}

// DedupeOpportunityMembers removes duplicate members on each deal (by id and displayName).
func DedupeOpportunityMembers(ctx context.Context, client crmDedupeClient) (DedupeResult, error) {
	var res DedupeResult
	items, err := client.ListAllOpportunities(ctx)
	if err != nil {
		return res, err
	}
	for _, row := range items {
		oppID := strconv.FormatInt(rowID(row), 10)
		opp, err := client.GetOpportunity(ctx, oppID)
		if err != nil {
			res.addErr(err)
			continue
		}
		members := OpportunityMembers(opp)
		if len(members) == 0 {
			continue
		}
		var ids []int64
		for _, m := range members {
			ids = append(ids, rowID(m))
		}
		remove := append(DedupeMemberIDs(ids), DedupeMembersByDisplayName(members)...)
		seen := make(map[int64]bool)
		for _, contactID := range remove {
			if contactID == 0 || seen[contactID] {
				continue
			}
			seen[contactID] = true
			if _, err := client.RemoveOpportunityMember(ctx, oppID, strconv.FormatInt(contactID, 10)); err != nil {
				res.addErr(err)
				continue
			}
			res.Removed++
		}
	}
	return res, nil
}

// FixOpportunityTitles renames deals with malformed titles.
func FixOpportunityTitles(ctx context.Context, client crmDedupeClient) (DedupeResult, error) {
	var res DedupeResult
	items, err := client.ListAllOpportunities(ctx)
	if err != nil {
		return res, err
	}
	for _, row := range items {
		old := fmt.Sprint(row["title"])
		newTitle := FixDealTitle(old)
		if newTitle == old || newTitle == "" {
			continue
		}
		oppID := strconv.FormatInt(rowID(row), 10)
		if _, err := client.UpdateOpportunityTitle(ctx, oppID, newTitle); err != nil {
			res.addErr(err)
			continue
		}
		res.Renamed++
	}
	return res, nil
}

// CleanupCRM runs all dedupe passes in dependency order.
func CleanupCRM(ctx context.Context, client crmDedupeClient, ignoreCompanySuffix bool) (map[string]DedupeResult, error) {
	out := make(map[string]DedupeResult)
	steps := []struct {
		name string
		fn   func(context.Context, crmDedupeClient) (DedupeResult, error)
	}{
		{"companies", func(ctx context.Context, c crmDedupeClient) (DedupeResult, error) {
			return DedupeCompanies(ctx, c)
		}},
		{"persons", func(ctx context.Context, c crmDedupeClient) (DedupeResult, error) {
			return DedupePersons(ctx, c)
		}},
		{"company-persons", func(ctx context.Context, c crmDedupeClient) (DedupeResult, error) {
			return DedupeCompanyPersons(ctx, c)
		}},
		{"contact-info", func(ctx context.Context, c crmDedupeClient) (DedupeResult, error) {
			return DedupeContactInfo(ctx, c)
		}},
		{"opportunity-members", func(ctx context.Context, c crmDedupeClient) (DedupeResult, error) {
			return DedupeOpportunityMembers(ctx, c)
		}},
	}
	for _, step := range steps {
		r, err := step.fn(ctx, client)
		out[step.name] = r
		if err != nil {
			return out, err
		}
	}
	r, err := DedupeOpportunities(ctx, client, ignoreCompanySuffix)
	out["opportunities"] = r
	if err != nil {
		return out, err
	}
	r, err = FixOpportunityTitles(ctx, client)
	out["fix-titles"] = r
	if err != nil {
		return out, err
	}
	return out, nil
}

var _ crmDedupeClient = (*Client)(nil)
