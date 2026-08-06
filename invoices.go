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
	EntityID            int64 // 0 = omit; link to opportunity/case
	EntityType          int   // OnlyOffice EntityType; 0 = Opportunity
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
	if p.EntityID != 0 {
		body["entityId"] = p.EntityID
		body["entityType"] = p.EntityType // 0 = Opportunity on this portal
	}
	return c.postJSONObject(ctx, "/api/2.0/crm/invoice", body)
}

// UpdateInvoiceParams are fields for PUT /api/2.0/crm/invoice/{id}.
// Loads the current invoice and merges non-zero / set fields.
type UpdateInvoiceParams struct {
	EntityID         int64 // link to opportunity; 0 = leave unchanged
	EntityType       int
	ContactID        int64  // 0 = leave unchanged
	Description      string // invoice notes; empty + DescriptionSet=false keeps existing
	DescriptionSet   bool
	PurchaseOrder    string
	PurchaseOrderSet bool
	Terms            string
	TermsSet         bool
}

// UpdateInvoice PUTs a full invoice body (OnlyOffice requires complete payload).
//
// Linking an opportunity via EntityID on an existing invoice often returns HTTP 400
// on this portal — prefer CreateInvoice with EntityID set. See docs/crm-associations.md.
func (c *Client) UpdateInvoice(ctx context.Context, id string, p UpdateInvoiceParams) (map[string]any, error) {
	inv, err := c.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	contactID := int64(0)
	if m, ok := inv["contact"].(map[string]any); ok {
		contactID = flexInt(m["id"])
	}
	if p.ContactID != 0 {
		contactID = p.ContactID
	}
	cur := "EUR"
	if m, ok := inv["currency"].(map[string]any); ok {
		if a := stringField(m, "abbreviation"); a != "" {
			cur = a
		}
	}
	statusID := 1
	if m, ok := inv["status"].(map[string]any); ok {
		statusID = int(flexInt(m["id"]))
	}
	var lines []map[string]any
	switch raw := inv["invoiceLines"].(type) {
	case []any:
		for _, row := range raw {
			m, ok := row.(map[string]any)
			if !ok {
				continue
			}
			lines = append(lines, map[string]any{
				"id":            flexInt(m["id"]),
				"invoiceItemID": flexInt(m["invoiceItemID"]),
				"invoiceTax1ID": flexInt(m["invoiceTax1ID"]),
				"invoiceTax2ID": flexInt(m["invoiceTax2ID"]),
				"description":   stringField(m, "description"),
				"quantity":      floatField(m, "quantity"),
				"price":         floatField(m, "price"),
				"discount":      floatField(m, "discount"),
				"sortOrder":     int(flexInt(m["sortOrder"])),
			})
		}
	case []map[string]any:
		for _, m := range raw {
			lines = append(lines, map[string]any{
				"id":            flexInt(m["id"]),
				"invoiceItemID": flexInt(m["invoiceItemID"]),
				"invoiceTax1ID": flexInt(m["invoiceTax1ID"]),
				"invoiceTax2ID": flexInt(m["invoiceTax2ID"]),
				"description":   stringField(m, "description"),
				"quantity":      floatField(m, "quantity"),
				"price":         floatField(m, "price"),
				"discount":      floatField(m, "discount"),
				"sortOrder":     int(flexInt(m["sortOrder"])),
			})
		}
	}
	body := map[string]any{
		"id":                  flexInt(inv["id"]),
		"number":              stringField(inv, "number"),
		"issueDate":           stringField(inv, "issueDate"),
		"dueDate":             stringField(inv, "dueDate"),
		"contactId":           contactID,
		"language":            stringField(inv, "language"),
		"currency":            cur,
		"exchangeRate":        floatField(inv, "exchangeRate"),
		"purchaseOrderNumber": stringField(inv, "purchaseOrderNumber"),
		"terms":               stringField(inv, "terms"),
		"description":         stringField(inv, "description"),
		"templateType":        int(flexInt(inv["templateType"])),
		"status":              statusID,
		"invoiceLines":        lines,
	}
	if p.DescriptionSet {
		body["description"] = p.Description
	}
	if p.PurchaseOrderSet {
		body["purchaseOrderNumber"] = p.PurchaseOrder
	}
	if p.TermsSet {
		body["terms"] = p.Terms
	}
	if p.EntityID != 0 {
		body["entityId"] = p.EntityID
		body["entityType"] = p.EntityType
	} else if ent, ok := inv["entity"].(map[string]any); ok && ent != nil {
		body["entityId"] = flexInt(ent["entityId"])
		// API returns entityType as string ("opportunity"); create/update want int.
		switch v := ent["entityType"].(type) {
		case float64:
			body["entityType"] = int(v)
		case int:
			body["entityType"] = v
		case string:
			if strings.EqualFold(v, "opportunity") {
				body["entityType"] = 0
			}
		}
	}
	return c.putJSONObject(ctx, fmt.Sprintf("/api/2.0/crm/invoice/%s", url.PathEscape(id)), body)
}

// DeleteInvoice removes an invoice by id.
func (c *Client) DeleteInvoice(ctx context.Context, id string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/invoice/%s.json", url.PathEscape(id)))
}

// Invoice status ids used by OnlyOffice CRM on produktor.io.
const (
	InvoiceStatusDraft    = 1
	InvoiceStatusBilled   = 2
	InvoiceStatusRejected = 3
	InvoiceStatusPaid     = 4
)

// SetInvoiceStatus sets CRM invoice status for one or more invoice ids
// (PUT /api/2.0/crm/invoice/status/{statusId} with invoiceids).
// Note: Billed→Draft often does not stick; recreate Draft instead (see docs/crm-associations.md).
func (c *Client) SetInvoiceStatus(ctx context.Context, statusID int, invoiceIDs ...int64) (map[string]any, error) {
	if statusID <= 0 {
		return nil, fmt.Errorf("status id is required")
	}
	if len(invoiceIDs) == 0 {
		return nil, fmt.Errorf("at least one invoice id is required")
	}
	ids := make([]string, 0, len(invoiceIDs))
	for _, id := range invoiceIDs {
		ids = append(ids, strconv.FormatInt(id, 10))
	}
	fields := url.Values{}
	fields.Set("invoiceids", strings.Join(ids, ","))
	return c.putFormObject(ctx, fmt.Sprintf("/api/2.0/crm/invoice/status/%d", statusID), fields)
}

// InvoicePDFFile returns invoice PDF file metadata (id, title, viewUrl).
// Without force, OnlyOffice may return a cached fileID with stale layout.
func (c *Client) InvoicePDFFile(ctx context.Context, invoiceID string) (map[string]any, error) {
	id := strings.TrimSpace(invoiceID)
	if id == "" {
		return nil, fmt.Errorf("InvoicePDFFile: invoice id is required")
	}
	return c.ResponseObject(ctx, fmt.Sprintf("/api/2.0/crm/invoice/%s/pdf", url.PathEscape(id)))
}

// ForceRegenerateInvoicePDF clears the cached PDF (Draft touch) then requests a new file.
// No-op touch when the invoice is not editable (e.g. Billed) — falls back to GET /pdf.
func (c *Client) ForceRegenerateInvoicePDF(ctx context.Context, invoiceID string) (map[string]any, error) {
	id := strings.TrimSpace(invoiceID)
	if id == "" {
		return nil, fmt.Errorf("ForceRegenerateInvoicePDF: invoice id is required")
	}
	inv, err := c.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	canEdit := true
	if v, ok := inv["canEdit"].(bool); ok {
		canEdit = v
	}
	if canEdit {
		desc := stringField(inv, "description")
		// Append/remove a trailing space so PUT clears fileID without visible change.
		touch := desc + " "
		if strings.HasSuffix(desc, " ") {
			touch = strings.TrimSuffix(desc, " ")
		}
		if _, err := c.UpdateInvoice(ctx, id, UpdateInvoiceParams{
			Description:    touch,
			DescriptionSet: true,
		}); err != nil {
			return nil, fmt.Errorf("ForceRegenerateInvoicePDF: clear cache: %w", err)
		}
	}
	return c.InvoicePDFFile(ctx, id)
}

// ListCRMContactFiles lists Documents attached on a CRM contact card (#files).
func (c *Client) ListCRMContactFiles(ctx context.Context, contactID string) ([]map[string]any, error) {
	return c.ResponseArray(ctx, fmt.Sprintf("/api/2.0/crm/contact/%s/files.json", url.PathEscape(contactID)))
}

// ListOpportunityFiles lists Documents attached on a CRM opportunity.
func (c *Client) ListOpportunityFiles(ctx context.Context, opportunityID string) ([]map[string]any, error) {
	return c.ResponseArray(ctx, fmt.Sprintf("/api/2.0/crm/opportunity/%s/files.json", url.PathEscape(opportunityID)))
}

// PurgeStaleInvoicePDFs deletes older P-*.pdf copies on the invoice contact (and linked
// opportunity) while keeping the invoice's current fileID. Returns deleted file ids.
func (c *Client) PurgeStaleInvoicePDFs(ctx context.Context, invoiceID string) ([]int, error) {
	inv, err := c.GetInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	keep := flexInt(inv["fileID"])
	number := strings.TrimSpace(stringField(inv, "number"))
	base := number
	for strings.HasSuffix(base, "b") || strings.HasSuffix(base, "B") {
		base = base[:len(base)-1]
	}
	if base == "" {
		base = "P-"
	}

	seen := map[int]struct{}{}
	var candidates []int
	addFiles := func(files []map[string]any) {
		for _, f := range files {
			title := stringField(f, "title")
			id := int(flexInt(f["id"]))
			if id == 0 || id == int(keep) {
				continue
			}
			if !strings.HasPrefix(title, "P-") {
				continue
			}
			if !strings.HasPrefix(title, base) {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			candidates = append(candidates, id)
		}
	}

	if m, ok := inv["contact"].(map[string]any); ok {
		cid := strconv.FormatInt(flexInt(m["id"]), 10)
		if cid != "0" {
			files, err := c.ListCRMContactFiles(ctx, cid)
			if err != nil {
				return nil, err
			}
			addFiles(files)
		}
	}
	if ent, ok := inv["entity"].(map[string]any); ok && ent != nil {
		if strings.EqualFold(fmt.Sprint(ent["entityType"]), "opportunity") || flexInt(ent["entityType"]) == 0 {
			oid := strconv.FormatInt(flexInt(ent["entityId"]), 10)
			if oid != "0" {
				files, err := c.ListOpportunityFiles(ctx, oid)
				if err != nil {
					return nil, err
				}
				addFiles(files)
			}
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}
	if err := c.DeleteFiles(ctx, candidates); err != nil {
		return nil, err
	}
	// CRM may still list deleted files briefly; also try CRM unlink.
	for _, fid := range candidates {
		_, _ = c.deleteObject(ctx, fmt.Sprintf("/api/2.0/crm/files/%d.json", fid))
	}
	return candidates, nil
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
