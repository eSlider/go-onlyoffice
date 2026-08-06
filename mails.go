package onlyoffice

// OnlyOffice Workspace Mail addon (/addons/mail) — list, read, and remove
// messages for the mailbox bound to the authenticated portal user.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
)

// Mail folder IDs in OnlyOffice Workspace (standard mailboxes).
const (
	MailFolderInbox  = 1
	MailFolderSent   = 2
	MailFolderDrafts = 3
	MailFolderTrash  = 4
	MailFolderSpam   = 5
)

// MailMessagesFilter selects messages from GET /api/2.0/mail/messages.
// The API returns at most 25 messages per request; ListMailMessages paginates
// automatically when Count exceeds that or StartIndex is non-zero.
type MailMessagesFilter struct {
	Folder     int // folder id (default inbox when zero)
	Count      int // max messages to return (0 → one API page)
	StartIndex int // skip this many messages before collecting Count
}

const mailMessagesPageSize = 25

// ListMailAccounts returns mailboxes linked to the current user.
func (c *Client) ListMailAccounts(ctx context.Context) ([]map[string]any, error) {
	return c.ResponseArray(ctx, "/api/2.0/mail/accounts")
}

// ListMailFolders returns folder counters (id, unread, total_count, …).
func (c *Client) ListMailFolders(ctx context.Context) ([]map[string]any, error) {
	return c.ResponseArray(ctx, "/api/2.0/mail/folders")
}

// ListMailMessages returns messages matching the filter.
func (c *Client) ListMailMessages(ctx context.Context, f MailMessagesFilter) ([]map[string]any, error) {
	want := f.Count
	if want <= 0 {
		want = mailMessagesPageSize
	}
	if want <= mailMessagesPageSize && f.StartIndex == 0 {
		return c.ResponseArray(ctx, mailMessagesPath(f, 1, want))
	}

	var out []map[string]any
	toSkip := f.StartIndex
	page := 1
	for len(out) < want {
		chunk, err := c.ResponseArray(ctx, mailMessagesPath(f, page, mailMessagesPageSize))
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}
		if toSkip > 0 {
			if toSkip >= len(chunk) {
				toSkip -= len(chunk)
				page++
				continue
			}
			chunk = chunk[toSkip:]
			toSkip = 0
		}
		pageLen := len(chunk)
		need := want - len(out)
		if len(chunk) > need {
			chunk = chunk[:need]
		}
		out = append(out, chunk...)
		if pageLen < mailMessagesPageSize {
			break
		}
		page++
	}
	return out, nil
}

// GetMailMessage returns one message by numeric id.
func (c *Client) GetMailMessage(ctx context.Context, messageID string) (map[string]any, error) {
	id := strings.TrimSpace(messageID)
	if id == "" {
		return nil, fmt.Errorf("GetMailMessage: message id is required")
	}
	return c.ResponseObject(ctx, "/api/2.0/mail/messages/"+url.PathEscape(id))
}

// RemoveMailMessages deletes messages by id (PUT /api/2.0/mail/messages/remove).
// The API response "response" field may be a number or object; success is HTTP 2xx.
func (c *Client) RemoveMailMessages(ctx context.Context, ids ...int) (map[string]any, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("RemoveMailMessages: at least one id is required")
	}
	raw, err := c.putJSON(ctx, "/api/2.0/mail/messages/remove", map[string]any{"ids": ids})
	if err != nil {
		return nil, err
	}
	out := map[string]any{"ok": true, "ids": ids}
	if resp, err := responseField(raw, "response"); err == nil {
		var n json.Number
		if json.Unmarshal(resp, &n) == nil {
			out["response"] = n.String()
		} else {
			var m map[string]any
			if json.Unmarshal(resp, &m) == nil {
				out["response"] = m
			} else {
				out["response"] = string(resp)
			}
		}
	}
	return out, nil
}

// SaveMailDraftParams creates or updates a draft via PUT /api/2.0/mail/drafts/save.
// Id 0 creates a new draft. Body is HTML; the API field name is "body" (not htmlBody).
type SaveMailDraftParams struct {
	ID      int64  // 0 = create
	From    string // mailbox address, e.g. eslider@gmail.com
	To      string // comma-separated or single address
	Cc      string
	Bcc     string
	Subject string
	Body    string // HTML
}

// SaveMailDraft saves a draft message. Returns the saved message map (incl. id).
func (c *Client) SaveMailDraft(ctx context.Context, p SaveMailDraftParams) (map[string]any, error) {
	if strings.TrimSpace(p.To) == "" {
		return nil, fmt.Errorf("SaveMailDraft: to is required")
	}
	if strings.TrimSpace(p.From) == "" {
		from, err := c.defaultMailFrom(ctx)
		if err != nil {
			return nil, err
		}
		p.From = from
	}
	body := map[string]any{
		"id":      p.ID,
		"from":    p.From,
		"to":      p.To,
		"cc":      p.Cc,
		"bcc":     p.Bcc,
		"subject": p.Subject,
		"body":    p.Body,
	}
	return c.putJSONObject(ctx, "/api/2.0/mail/drafts/save", body)
}

func (c *Client) defaultMailFrom(ctx context.Context) (string, error) {
	accounts, err := c.ListMailAccounts(ctx)
	if err != nil {
		return "", err
	}
	var fallback string
	for _, a := range accounts {
		email := strings.TrimSpace(stringFromMap(a, "email"))
		if email == "" || email == "<nil>" {
			continue
		}
		if boolField(a, "enabled") || boolField(a, "isDefault") {
			return email, nil
		}
		if fallback == "" {
			fallback = email
		}
	}
	if fallback == "" {
		return "", fmt.Errorf("SaveMailDraft: from is required (no mailbox on account)")
	}
	return fallback, nil
}

// AttachMailDocument attaches an OnlyOffice Files document to a mail message/draft.
// fileID is the Documents/CRM file id (e.g. from InvoicePDFFile).
func (c *Client) AttachMailDocument(ctx context.Context, messageID string, fileID int64) (map[string]any, error) {
	id := strings.TrimSpace(messageID)
	if id == "" {
		return nil, fmt.Errorf("AttachMailDocument: message id is required")
	}
	if fileID <= 0 {
		return nil, fmt.Errorf("AttachMailDocument: file id is required")
	}
	fields := url.Values{}
	fields.Set("fileId", strconv.FormatInt(fileID, 10))
	return c.postFormObject(ctx, fmt.Sprintf("/api/2.0/mail/messages/%s/document", url.PathEscape(id)), fields)
}

// InvoicePDFFile regenerates/returns the invoice PDF file metadata (id, title, viewUrl).
func (c *Client) InvoicePDFFile(ctx context.Context, invoiceID string) (map[string]any, error) {
	id := strings.TrimSpace(invoiceID)
	if id == "" {
		return nil, fmt.Errorf("InvoicePDFFile: invoice id is required")
	}
	return c.ResponseObject(ctx, fmt.Sprintf("/api/2.0/crm/invoice/%s/pdf", url.PathEscape(id)))
}

// PlainTextToMailHTML turns plain text into simple HTML paragraphs for drafts.
// If s already looks like HTML, it is returned unchanged.
func PlainTextToMailHTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "<") && strings.Contains(s, ">") {
		return s
	}
	parts := strings.Split(s, "\n\n")
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		b.WriteString("<p>")
		b.WriteString(strings.ReplaceAll(p, "\n", "<br/>"))
		b.WriteString("</p>\n")
	}
	return b.String()
}

// MailHTMLWithBlankParagraphs inserts empty paragraphs between blocks
// (German Geschäftsbrief-style vertical spacing in OnlyOffice Mail).
func MailHTMLWithBlankParagraphs(html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return ""
	}
	return strings.ReplaceAll(html, "</p>\n<p>", "</p>\n<p>&nbsp;</p>\n<p>")
}

// Int64FromMap coerces OnlyOffice numeric id fields (float64/int/string) to int64.
func Int64FromMap(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(m[key]), 10, 64)
		return n
	}
}

// ResolveMailFolder maps a CLI folder name or numeric string to a folder id.
// Empty input defaults to inbox (1).
func ResolveMailFolder(name string) (int, error) {
	s := strings.TrimSpace(strings.ToLower(name))
	if s == "" {
		return MailFolderInbox, nil
	}
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n, nil
	}
	switch s {
	case "inbox":
		return MailFolderInbox, nil
	case "sent":
		return MailFolderSent, nil
	case "drafts", "draft":
		return MailFolderDrafts, nil
	case "trash":
		return MailFolderTrash, nil
	case "spam":
		return MailFolderSpam, nil
	default:
		return 0, fmt.Errorf("unknown mail folder %q (use inbox|sent|drafts|trash|spam or numeric id)", name)
	}
}

func mailMessagesPath(f MailMessagesFilter, page, count int) string {
	q := url.Values{}
	folder := f.Folder
	if folder <= 0 {
		folder = MailFolderInbox
	}
	q.Set("folder", strconv.Itoa(folder))
	if count > 0 {
		q.Set("count", strconv.Itoa(count))
	}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	return "/api/2.0/mail/messages?" + q.Encode()
}

// ParseMailAddress splits a RFC 5322 mailbox string into display name and email.
// Examples:
//   - `"LinkedIn" <a@b.com>` → name LinkedIn, address a@b.com
//   - `eslider@gmail.com` → address only
func ParseMailAddress(raw string) (name, address string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	a, err := mail.ParseAddress(raw)
	if err != nil {
		return "", raw
	}
	return a.Name, a.Address
}

// MailMessagesAsTableRows converts list results for printTable.
func MailMessagesAsTableRows(msgs []map[string]any) []map[string]any {
	rows := make([]map[string]any, len(msgs))
	for i, m := range msgs {
		fromName, fromAddress := ParseMailAddress(stringFromMap(m, "from"))
		rows[i] = map[string]any{
			"id":          idFromMap(m, "id"),
			"subject":     stringFromMap(m, "subject"),
			"fromName":    fromName,
			"fromAddress": fromAddress,
			"date":        stringFromMap(m, "date"),
			"folder":      idFromMap(m, "folder"),
			"size":        idFromMap(m, "size"),
			"isNew":       boolFromMap(m, "isNew"),
		}
	}
	return rows
}

// MailAccountsAsTableRows converts account list results for printTable.
func MailAccountsAsTableRows(accounts []map[string]any) []map[string]any {
	rows := make([]map[string]any, len(accounts))
	for i, a := range accounts {
		rows[i] = map[string]any{
			"mailboxId": idFromMap(a, "mailboxId"),
			"email":     stringFromMap(a, "email"),
			"enabled":   boolFromMap(a, "enabled"),
			"isDefault": boolFromMap(a, "isDefault"),
		}
	}
	return rows
}

// MailFoldersAsTableRows converts folder list results for printTable.
func MailFoldersAsTableRows(folders []map[string]any) []map[string]any {
	rows := make([]map[string]any, len(folders))
	for i, f := range folders {
		rows[i] = map[string]any{
			"id":            idFromMap(f, "id"),
			"unread":        idFromMap(f, "unread"),
			"total_count":   idFromMap(f, "total_count"),
			"time_modified": stringFromMap(f, "time_modified"),
		}
	}
	return rows
}

func idFromMap(m map[string]any, key string) string {
	switch v := m[key].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	default:
		return fmt.Sprint(m[key])
	}
}

func stringFromMap(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return fmt.Sprint(m[key])
}

func boolFromMap(m map[string]any, key string) string {
	switch v := m[key].(type) {
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprint(m[key])
	}
}
