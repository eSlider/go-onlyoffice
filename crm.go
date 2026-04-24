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
	items, _, err := c.ListContacts(ctx, 50, 0, name)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(name))
	for _, co := range items {
		if isCompany(co) && strings.ToLower(fmt.Sprint(co["displayName"])) == needle {
			return co, nil
		}
	}
	return nil, nil
}

// FindPerson searches for a person by first+last (case-insensitive).
func (c *Client) FindPerson(ctx context.Context, first, last string) (map[string]any, error) {
	items, _, err := c.ListContacts(ctx, 50, 0, first+" "+last)
	if err != nil {
		return nil, err
	}
	first = strings.ToLower(first)
	last = strings.ToLower(last)
	for _, p := range items {
		if isCompany(p) {
			continue
		}
		if strings.ToLower(fmt.Sprint(p["firstName"])) == first &&
			strings.ToLower(fmt.Sprint(p["lastName"])) == last {
			return p, nil
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
