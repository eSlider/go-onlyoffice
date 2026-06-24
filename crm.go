package onlyoffice

// Minimal CRM helpers: contacts, opportunities, cases, tasks, and history notes.
// These expose untyped maps for flexibility — they are primarily consumed by
// cmd/oo and the applications-sync workflow.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ListContacts returns a page of CRM contacts and the total count.
func (c *Client) ListContacts(ctx context.Context, count, startIndex int, search string) ([]map[string]any, int, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("startIndex", strconv.Itoa(startIndex))
	if search != "" {
		q.Set("filterValue", search)
	}
	raw, err := c.getJSON(ctx, "/api/2.0/crm/contact/filter.json?"+q.Encode())
	if err != nil {
		return nil, 0, err
	}
	var env struct {
		Response []map[string]any `json:"response"`
		Total    int              `json:"total"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, 0, err
	}
	total := env.Total
	if total == 0 && len(env.Response) > 0 {
		total = len(env.Response)
	}
	return env.Response, total, nil
}

// GetContact returns a single contact by id.
func (c *Client) GetContact(ctx context.Context, contactID string) (map[string]any, error) {
	return c.ResponseObject(ctx, fmt.Sprintf("/api/2.0/crm/contact/%s.json", url.PathEscape(contactID)))
}

// FindCompany searches for a company contact with an exact (case-insensitive)
// displayName match. Returns nil when not found.
func (c *Client) FindCompany(ctx context.Context, name string) (map[string]any, error) {
	needle := strings.ToLower(strings.TrimSpace(name))
	const page = 50
	for start := 0; ; start += page {
		items, total, err := c.ListContacts(ctx, page, start, name)
		if err != nil {
			return nil, err
		}
		for _, co := range items {
			if isCompany(co) && strings.ToLower(fmt.Sprint(co["displayName"])) == needle {
				return co, nil
			}
		}
		if start+page >= total || len(items) == 0 {
			break
		}
	}
	return nil, nil
}

// FindPerson searches for a person by first+last (case-insensitive).
func (c *Client) FindPerson(ctx context.Context, first, last string) (map[string]any, error) {
	firstNeedle := strings.ToLower(strings.TrimSpace(first))
	lastNeedle := strings.ToLower(strings.TrimSpace(last))
	const page = 50
	for start := 0; ; start += page {
		items, total, err := c.ListContacts(ctx, page, start, first+" "+last)
		if err != nil {
			return nil, err
		}
		for _, p := range items {
			if isCompany(p) {
				continue
			}
			if strings.ToLower(fmt.Sprint(p["firstName"])) == firstNeedle &&
				strings.ToLower(fmt.Sprint(p["lastName"])) == lastNeedle {
				return p, nil
			}
		}
		if start+page >= total || len(items) == 0 {
			break
		}
	}
	return nil, nil
}

func isCompany(m map[string]any) bool {
	v, ok := m["isCompany"].(bool)
	return ok && v
}

// CreateCompany creates a company contact with the given name.
func (c *Client) CreateCompany(ctx context.Context, name string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("companyName", name)
	return c.postFormObject(ctx, "/api/2.0/crm/contact/company.json", fields)
}

// CreatePerson creates a person contact; companyID == 0 means unlinked.
func (c *Client) CreatePerson(ctx context.Context, first, last string, companyID int, jobTitle, about string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("firstName", first)
	fields.Set("lastName", last)
	if companyID != 0 {
		fields.Set("companyId", strconv.Itoa(companyID))
	}
	if jobTitle != "" {
		fields.Set("jobTitle", jobTitle)
	}
	if about != "" {
		fields.Set("about", about)
	}
	return c.postFormObject(ctx, "/api/2.0/crm/contact/person.json", fields)
}

// AddContactInfo attaches an email/website/phone/etc. to a contact.
func (c *Client) AddContactInfo(ctx context.Context, contactID, infoType, dataValue, category string, isPrimary bool) (map[string]any, error) {
	if category == "" {
		category = "Work"
	}
	fields := url.Values{}
	fields.Set("infoType", infoType)
	fields.Set("data", dataValue)
	fields.Set("category", category)
	fields.Set("isPrimary", strconv.FormatBool(isPrimary))
	return c.postFormObject(ctx, fmt.Sprintf("/api/2.0/crm/contact/%s/data.json", url.PathEscape(contactID)), fields)
}

// DeleteContact removes a CRM contact by id.
func (c *Client) DeleteContact(ctx context.Context, contactID string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/contact/%s.json", url.PathEscape(contactID)))
}

// ListAllContacts paginates through every CRM contact.
func (c *Client) ListAllContacts(ctx context.Context) ([]map[string]any, error) {
	const page = 100
	var all []map[string]any
	for start := 0; ; start += page {
		chunk, total, err := c.ListContacts(ctx, page, start, "")
		if err != nil {
			return nil, err
		}
		all = append(all, chunk...)
		if start+page >= total || len(chunk) == 0 {
			break
		}
	}
	return all, nil
}

// MergeContacts merges secondary into primary (secondary is removed).
func (c *Client) MergeContacts(ctx context.Context, primaryID, secondaryID string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("fromContactId", secondaryID)
	fields.Set("toContactId", primaryID)
	out, err := c.putFormObject(ctx, "/api/2.0/crm/contact/merge.json", fields)
	if err == nil {
		return out, nil
	}
	// Some instances expect JSON body with alternate field names.
	body := map[string]any{
		"fromContactId": secondaryID,
		"toContactId":   primaryID,
	}
	return c.putJSONObject(ctx, "/api/2.0/crm/contact/merge.json", body)
}

// ListCompanyPersons returns persons linked to a company.
func (c *Client) ListCompanyPersons(ctx context.Context, companyID string) ([]map[string]any, error) {
	return c.ResponseArray(ctx, fmt.Sprintf("/api/2.0/crm/contact/company/%s/person.json", url.PathEscape(companyID)))
}

// DeleteContactInfo removes one info row from a contact.
func (c *Client) DeleteContactInfo(ctx context.Context, contactID, dataID string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/contact/%s/data/%s.json", url.PathEscape(contactID), url.PathEscape(dataID)))
}

// ContactInfoRows returns commonData/info rows from a contact map.
func ContactInfoRows(contact map[string]any) []map[string]any {
	for _, key := range []string{"commonData", "data", "contactData"} {
		if rows, ok := contact[key].([]any); ok {
			return mapsFromAnySlice(rows)
		}
		if rows, ok := contact[key].([]map[string]any); ok {
			return rows
		}
	}
	return nil
}

// HasContactInfo reports whether a contact already has the given type+value.
func HasContactInfo(contact map[string]any, infoType, value string) bool {
	key := ContactInfoKey(infoType, value)
	for _, row := range ContactInfoRows(contact) {
		v := fmt.Sprint(row["data"])
		if v == "" || v == "<nil>" {
			v = fmt.Sprint(row["value"])
		}
		if ContactInfoKey(fmt.Sprint(row["infoType"]), v) == key {
			return true
		}
	}
	return false
}

func mapsFromAnySlice(rows []any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if m, ok := row.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// ListOpportunities returns a page of deals/opportunities and the total count.
func (c *Client) ListOpportunities(ctx context.Context, count, startIndex int) ([]map[string]any, int, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("startIndex", strconv.Itoa(startIndex))
	raw, err := c.getJSON(ctx, "/api/2.0/crm/opportunity/filter.json?"+q.Encode())
	if err != nil {
		return nil, 0, err
	}
	var env struct {
		Response []map[string]any `json:"response"`
		Total    int              `json:"total"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, 0, err
	}
	return env.Response, env.Total, nil
}

// GetOpportunity returns a single opportunity (deal) by id.
func (c *Client) GetOpportunity(ctx context.Context, id string) (map[string]any, error) {
	return c.ResponseObject(ctx, fmt.Sprintf("/api/2.0/crm/opportunity/%s.json", url.PathEscape(id)))
}

// CreateOpportunity creates a new deal. An empty responsibleID falls back to
// the authenticated user's id, and bidCurrency defaults to "EUR".
func (c *Client) CreateOpportunity(ctx context.Context, title string, stageID int, responsibleID, bidCurrency, description string, bidValue float64) (map[string]any, error) {
	if responsibleID == "" {
		s, err := c.SelfUserID(ctx)
		if err != nil {
			return nil, err
		}
		responsibleID = s
	}
	if bidCurrency == "" {
		bidCurrency = "EUR"
	}
	fields := url.Values{}
	fields.Set("title", title)
	fields.Set("stageId", strconv.Itoa(stageID))
	fields.Set("responsibleId", responsibleID)
	fields.Set("bidCurrencyAbbr", bidCurrency)
	if bidValue != 0 {
		fields.Set("bidValue", strconv.FormatFloat(bidValue, 'g', -1, 64))
	}
	if description != "" {
		fields.Set("description", description)
	}
	return c.postFormObject(ctx, "/api/2.0/crm/opportunity.json", fields)
}

// AddOpportunityMember links a contact to an opportunity.
func (c *Client) AddOpportunityMember(ctx context.Context, oppID, contactID string) (map[string]any, error) {
	return c.postFormObject(ctx,
		fmt.Sprintf("/api/2.0/crm/opportunity/%s/contact/%s.json", url.PathEscape(oppID), url.PathEscape(contactID)),
		url.Values{})
}

// ListDealStages returns the configured opportunity stages.
func (c *Client) ListDealStages(ctx context.Context) ([]map[string]any, error) {
	return c.ResponseArray(ctx, "/api/2.0/crm/opportunity/stage.json")
}

// DeleteOpportunity removes a deal by id.
func (c *Client) DeleteOpportunity(ctx context.Context, id string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/opportunity/%s.json", url.PathEscape(id)))
}

// ListAllOpportunities paginates through every opportunity.
func (c *Client) ListAllOpportunities(ctx context.Context) ([]map[string]any, error) {
	const page = 100
	var all []map[string]any
	for start := 0; ; start += page {
		chunk, total, err := c.ListOpportunities(ctx, page, start)
		if err != nil {
			return nil, err
		}
		all = append(all, chunk...)
		if start+page >= total || len(chunk) == 0 {
			break
		}
	}
	return all, nil
}

// OpportunityMembers extracts the members slice from a GetOpportunity response.
func OpportunityMembers(opp map[string]any) []map[string]any {
	raw, ok := opp["members"].([]any)
	if !ok {
		if rows, ok := opp["members"].([]map[string]any); ok {
			return rows
		}
		return nil
	}
	return mapsFromAnySlice(raw)
}

// ListOpportunityMembers returns contacts linked to an opportunity.
func (c *Client) ListOpportunityMembers(ctx context.Context, oppID string) ([]map[string]any, error) {
	opp, err := c.GetOpportunity(ctx, oppID)
	if err != nil {
		return nil, err
	}
	if members := OpportunityMembers(opp); len(members) > 0 {
		return members, nil
	}
	return c.ResponseArray(ctx, fmt.Sprintf("/api/2.0/crm/opportunity/%s/contact.json", url.PathEscape(oppID)))
}

// RemoveOpportunityMember detaches a contact from an opportunity.
func (c *Client) RemoveOpportunityMember(ctx context.Context, oppID, contactID string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/opportunity/%s/contact/%s.json", url.PathEscape(oppID), url.PathEscape(contactID)))
}

// IsOpportunityMember reports whether contactID is already on the opportunity.
func (c *Client) IsOpportunityMember(ctx context.Context, oppID, contactID string) (bool, error) {
	members, err := c.ListOpportunityMembers(ctx, oppID)
	if err != nil {
		return false, err
	}
	want := flexInt(contactID)
	for _, m := range members {
		if flexInt(m["id"]) == want {
			return true, nil
		}
	}
	return false, nil
}

// UpdateOpportunityTitle renames a deal; loads full record and PUTs it back.
func (c *Client) UpdateOpportunityTitle(ctx context.Context, id, newTitle string) (map[string]any, error) {
	opp, err := c.GetOpportunity(ctx, id)
	if err != nil {
		return nil, err
	}
	body := opportunityUpdateBody(opp, newTitle)
	return c.putJSONObject(ctx, fmt.Sprintf("/api/2.0/crm/opportunity/%s.json", url.PathEscape(id)), body)
}

func opportunityUpdateBody(opp map[string]any, title string) map[string]any {
	body := map[string]any{
		"opportunityid": flexInt(opp["id"]),
		"title":         title,
		"description":   stringField(opp, "description"),
		"isPrivate":     boolField(opp, "isPrivate"),
		"isNotify":      false,
	}
	if stage, ok := opp["stage"].(map[string]any); ok {
		body["stageid"] = flexInt(stage["id"])
	}
	if resp, ok := opp["responsible"].(map[string]any); ok {
		body["responsibleid"] = fmt.Sprint(resp["id"])
	}
	if cur, ok := opp["bidCurrency"].(map[string]any); ok {
		body["bidCurrencyAbbr"] = stringField(cur, "abbreviation")
	} else {
		body["bidCurrencyAbbr"] = "EUR"
	}
	body["bidValue"] = floatField(opp, "bidValue")
	body["bidType"] = 0
	body["perPeriodValue"] = 0
	body["successProbability"] = 1
	var memberIDs []int64
	seen := make(map[int64]bool)
	for _, m := range OpportunityMembers(opp) {
		id := flexInt(m["id"])
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		memberIDs = append(memberIDs, id)
	}
	if len(memberIDs) > 0 {
		body["members"] = memberIDs
		body["contactid"] = memberIDs[0]
	}
	if al, ok := opp["accessList"].([]any); ok && len(al) > 0 {
		body["accessList"] = al
	} else {
		body["accessList"] = []any{}
	}
	return body
}

func stringField(m map[string]any, key string) string {
	v := fmt.Sprint(m[key])
	if v == "<nil>" {
		return ""
	}
	return v
}

func boolField(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func floatField(m map[string]any, key string) float64 {
	switch x := m[key].(type) {
	case float64:
		return x
	case int:
		return float64(x)
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(x), 64)
		return f
	}
}

// ListCases returns a page of CRM cases and the total count.
func (c *Client) ListCases(ctx context.Context, count, startIndex int) ([]map[string]any, int, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("startIndex", strconv.Itoa(startIndex))
	raw, err := c.getJSON(ctx, "/api/2.0/crm/case/filter.json?"+q.Encode())
	if err != nil {
		return nil, 0, err
	}
	var env struct {
		Response []map[string]any `json:"response"`
		Total    int              `json:"total"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, 0, err
	}
	return env.Response, env.Total, nil
}

// CreateCase creates a new CRM case.
func (c *Client) CreateCase(ctx context.Context, title string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("title", title)
	return c.postFormObject(ctx, "/api/2.0/crm/case.json", fields)
}

// AddCaseMember links a contact to a CRM case.
func (c *Client) AddCaseMember(ctx context.Context, caseID, contactID string) (map[string]any, error) {
	return c.postFormObject(ctx,
		fmt.Sprintf("/api/2.0/crm/case/%s/contact/%s.json", url.PathEscape(caseID), url.PathEscape(contactID)),
		url.Values{})
}

// DeleteCase removes a case by id.
func (c *Client) DeleteCase(ctx context.Context, id string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/case/%s.json", url.PathEscape(id)))
}

// ListCRMTasks returns a page of CRM tasks (separate from Project tasks).
func (c *Client) ListCRMTasks(ctx context.Context, count, startIndex int) ([]map[string]any, int, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("startIndex", strconv.Itoa(startIndex))
	raw, err := c.getJSON(ctx, "/api/2.0/crm/task/filter.json?"+q.Encode())
	if err != nil {
		return nil, 0, err
	}
	var env struct {
		Response []map[string]any `json:"response"`
		Total    int              `json:"total"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, 0, err
	}
	return env.Response, env.Total, nil
}

// CreateCRMTask creates a CRM task (reminder) attached to an entity.
func (c *Client) CreateCRMTask(ctx context.Context, title, deadline string, categoryID, contactID int, entityType string, entityID int, description string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("title", title)
	fields.Set("deadline", deadline)
	fields.Set("categoryId", strconv.Itoa(categoryID))
	if contactID != 0 {
		fields.Set("contactId", strconv.Itoa(contactID))
	}
	if entityType != "" {
		fields.Set("entityType", entityType)
	}
	if entityID != 0 {
		fields.Set("entityId", strconv.Itoa(entityID))
	}
	if description != "" {
		fields.Set("description", description)
	}
	return c.postFormObject(ctx, "/api/2.0/crm/task.json", fields)
}

// DeleteCRMTask removes a CRM task by id.
func (c *Client) DeleteCRMTask(ctx context.Context, id string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/task/%s.json", url.PathEscape(id)))
}

// ListTaskCategories returns CRM task categories.
func (c *Client) ListTaskCategories(ctx context.Context) ([]map[string]any, error) {
	return c.ResponseArray(ctx, "/api/2.0/crm/task/category.json")
}

// AddHistoryNote attaches a history note to a CRM entity. When categoryID is 0,
// the cached "note" category id is looked up from /api/2.0/crm/history/category.json.
func (c *Client) AddHistoryNote(ctx context.Context, entityType string, entityID int, content string, categoryID int) (map[string]any, error) {
	if categoryID == 0 {
		id, err := c.historyNoteCategoryID(ctx)
		if err != nil {
			return nil, err
		}
		categoryID = id
	}
	fields := url.Values{}
	fields.Set("entityType", entityType)
	fields.Set("entityId", strconv.Itoa(entityID))
	fields.Set("content", content)
	fields.Set("categoryId", strconv.Itoa(categoryID))
	return c.postFormObject(ctx, "/api/2.0/crm/history.json", fields)
}

func (c *Client) historyNoteCategoryID(ctx context.Context) (int, error) {
	if c.noteCatID != 0 {
		return c.noteCatID, nil
	}
	list, err := c.ResponseArray(ctx, "/api/2.0/crm/history/category.json")
	if err != nil {
		return 0, err
	}
	for _, row := range list {
		if strings.EqualFold(fmt.Sprint(row["title"]), "note") {
			c.noteCatID = int(flexInt(row["id"]))
			return c.noteCatID, nil
		}
	}
	if len(list) > 0 {
		c.noteCatID = int(flexInt(list[0]["id"]))
		return c.noteCatID, nil
	}
	return 0, nil
}

// flexInt coerces OnlyOffice numeric fields that JSON unmarshal may surface as
// float64, int, json.Number, or string into a plain int64.
func flexInt(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int:
		return int64(x)
	case int64:
		return x
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(x), 10, 64)
		return n
	}
}
