package onlyoffice

// CRM Invoices and invoice catalog items (billing).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// InvoiceLine is one line on a create-invoice request.
type InvoiceLine struct {
	InvoiceItemID int64   `json:"invoiceItemID"`
	InvoiceTax1ID int64   `json:"invoiceTax1ID"`
	InvoiceTax2ID int64   `json:"invoiceTax2ID"`
	Description   string  `json:"description"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	Discount      float64 `json:"discount"`
	SortOrder     int     `json:"sortOrder"`
}

// CreateInvoiceParams holds fields for POST /api/2.0/crm/invoice.
type CreateInvoiceParams struct {
	Number              string
	IssueDate           string // ISO-8601
	DueDate             string // ISO-8601
	ContactID           int64
	ConsigneeID         int64 // 0 = omit
	Language            string
	Currency            string
	ExchangeRate        float64
	PurchaseOrderNumber string
	Terms               string
	Description         string
	TemplateType        int
	Lines               []InvoiceLine
}

// ListInvoices returns a page of CRM invoices and the total count.
func (c *Client) ListInvoices(ctx context.Context, count, startIndex int) ([]map[string]any, int, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("startIndex", strconv.Itoa(startIndex))
	raw, err := c.getJSON(ctx, "/api/2.0/crm/invoice/filter.json?"+q.Encode())
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

// GetInvoice returns a single invoice by id (includes invoiceLines).
func (c *Client) GetInvoice(ctx context.Context, id string) (map[string]any, error) {
	return c.ResponseObject(ctx, fmt.Sprintf("/api/2.0/crm/invoice/%s.json", url.PathEscape(id)))
}

// CreateInvoice creates a draft invoice with the given lines.
func (c *Client) CreateInvoice(ctx context.Context, p CreateInvoiceParams) (map[string]any, error) {
	if p.ContactID == 0 {
		return nil, fmt.Errorf("contactId is required")
	}
	if p.Number == "" {
		return nil, fmt.Errorf("number is required")
	}
	if len(p.Lines) == 0 {
		return nil, fmt.Errorf("at least one invoice line is required")
	}
	if p.Language == "" {
		p.Language = "de-DE"
	}
	if p.Currency == "" {
		p.Currency = "EUR"
	}
	if p.ExchangeRate == 0 {
		p.ExchangeRate = 1
	}
	body := map[string]any{
		"number":              p.Number,
		"issueDate":           p.IssueDate,
		"dueDate":             p.DueDate,
		"contactId":           p.ContactID,
		"language":            p.Language,
		"currency":            p.Currency,
		"exchangeRate":        p.ExchangeRate,
		"purchaseOrderNumber": p.PurchaseOrderNumber,
		"terms":               p.Terms,
		"description":         p.Description,
		"templateType":        p.TemplateType,
		"invoiceLines":        p.Lines,
	}
	if p.ConsigneeID != 0 {
		body["consigneeId"] = p.ConsigneeID
	}
	return c.postJSONObject(ctx, "/api/2.0/crm/invoice", body)
}

// DeleteInvoice removes an invoice by id.
func (c *Client) DeleteInvoice(ctx context.Context, id string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/invoice/%s.json", url.PathEscape(id)))
}

// ListInvoiceItems returns catalog invoice items.
func (c *Client) ListInvoiceItems(ctx context.Context, count, startIndex int) ([]map[string]any, int, error) {
	q := url.Values{}
	q.Set("count", strconv.Itoa(count))
	q.Set("startIndex", strconv.Itoa(startIndex))
	raw, err := c.getJSON(ctx, "/api/2.0/crm/invoiceitem/filter.json?"+q.Encode())
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

// CreateInvoiceItem creates a reusable catalog line item.
func (c *Client) CreateInvoiceItem(ctx context.Context, title, description string, price float64, currency string) (map[string]any, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if currency == "" {
		currency = "EUR"
	}
	fields := url.Values{}
	fields.Set("title", title)
	fields.Set("description", description)
	fields.Set("price", strconv.FormatFloat(price, 'f', 2, 64))
	fields.Set("currency", currency)
	fields.Set("stockKeepingUnit", "")
	fields.Set("trackInventory", "false")
	fields.Set("quantity", "0")
	fields.Set("invoiceTax1ID", "0")
	fields.Set("invoiceTax2ID", "0")
	return c.postFormObject(ctx, "/api/2.0/crm/invoiceitem", fields)
}

// DeleteInvoiceItem removes a catalog invoice item.
func (c *Client) DeleteInvoiceItem(ctx context.Context, id string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/invoiceitem/%s.json", url.PathEscape(id)))
}

// AddContactAddress attaches a postal address to a contact.
// category: Home|Postal|Office|Billing|Other|Work (or numeric string).
func (c *Client) AddContactAddress(ctx context.Context, contactID, street, city, state, zip, country, category string, isPrimary bool) (map[string]any, error) {
	if category == "" {
		category = "Billing"
	}
	fields := url.Values{}
	fields.Set("street", street)
	fields.Set("city", city)
	fields.Set("state", state)
	fields.Set("zip", zip)
	fields.Set("country", country)
	fields.Set("category", category)
	fields.Set("isPrimary", strconv.FormatBool(isPrimary))
	return c.postFormObject(ctx, fmt.Sprintf("/api/2.0/crm/contact/%s/address", url.PathEscape(contactID)), fields)
}

// UpdateCompany updates company name and optional about text.
func (c *Client) UpdateCompany(ctx context.Context, companyID, name, about string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("companyName", name)
	if about != "" {
		fields.Set("about", about)
	}
	return c.putFormObject(ctx, fmt.Sprintf("/api/2.0/crm/contact/company/%s.json", url.PathEscape(companyID)), fields)
}

// UpdateOpportunityParams are optional fields for UpdateOpportunity.
// Zero/empty values mean "keep existing" except BidValueSet.
type UpdateOpportunityParams struct {
	Title       string
	Description string
	StageID     int64
	BidValue    float64
	BidValueSet bool
}

// UpdateOpportunity loads a deal and PUTs updated fields.
func (c *Client) UpdateOpportunity(ctx context.Context, id string, p UpdateOpportunityParams) (map[string]any, error) {
	opp, err := c.GetOpportunity(ctx, id)
	if err != nil {
		return nil, err
	}
	title := p.Title
	if title == "" {
		title = stringField(opp, "title")
	}
	body := opportunityUpdateBody(opp, title)
	if p.Description != "" {
		body["description"] = p.Description
	}
	if p.StageID != 0 {
		body["stageid"] = p.StageID
	}
	if p.BidValueSet {
		body["bidValue"] = p.BidValue
	}
	return c.putJSONObject(ctx, fmt.Sprintf("/api/2.0/crm/opportunity/%s.json", url.PathEscape(id)), body)
}

// InvoiceContactName extracts displayName from nested contact on an invoice row.
func InvoiceContactName(inv map[string]any) string {
	if m, ok := inv["contact"].(map[string]any); ok {
		return stringField(m, "displayName")
	}
	return ""
}

// InvoiceStatusTitle extracts status.title from an invoice row.
func InvoiceStatusTitle(inv map[string]any) string {
	if m, ok := inv["status"].(map[string]any); ok {
		return stringField(m, "title")
	}
	return ""
}

// FlattenInvoiceRow copies nested fields for table output.
func FlattenInvoiceRow(inv map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range inv {
		out[k] = v
	}
	out["contactName"] = InvoiceContactName(inv)
	out["statusTitle"] = InvoiceStatusTitle(inv)
	if cur, ok := inv["currency"].(map[string]any); ok {
		out["currency"] = stringField(cur, "abbreviation")
	}
	return out
}

// FindInvoiceItemByTitle returns the first catalog item with an exact title match.
func FindInvoiceItemByTitle(items []map[string]any, title string) map[string]any {
	want := strings.TrimSpace(title)
	for _, it := range items {
		if strings.TrimSpace(stringField(it, "title")) == want {
			return it
		}
	}
	return nil
}
