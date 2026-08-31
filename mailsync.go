package onlyoffice

// High-level mail folder walk for ETL consumers (2dph brain mail-ingest,
// cv tools). This is the "integration layer" half of reusing the canonical
// client instead of private per-project OOClient copies: the caller gets a
// single hydrated stream instead of hand-rolling list → get → download
// against the raw API.

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// MailSyncAttachment is one attachment of a hydrated mail message.
type MailSyncAttachment struct {
	ID   string // id accepted by Client.DownloadMailAttachment
	Name string
	Size int64
	Body []byte // non-nil only when MailSyncOptions.FetchBodies is set
}

// MailSyncMessage is a hydrated mail message for sync pipelines.
type MailSyncMessage struct {
	ID             int64
	Folder         int
	Subject        string
	From           string // raw RFC 5322 header value ("Name" <addr>)
	Date           time.Time
	IsNew          bool
	HasAttachments bool
	Attachments    []MailSyncAttachment
}

// MailSyncOptions controls FetchMailFolder.
type MailSyncOptions struct {
	Limit       int  // max messages to hydrate; 0 = whole folder
	StartIndex  int  // skip this many messages before collecting
	FetchBodies bool // eagerly download attachment bytes
}

// FetchMailFolder walks a mail folder page by page and hydrates every
// message: list → get → (optionally) download attachments. It is the single
// entry point sync pipelines need on top of the mail API.
//
// Messages are returned in API order (newest first). The folder walk stops
// at the first empty or short page.
func (c *Client) FetchMailFolder(ctx context.Context, folderID int, opts MailSyncOptions) ([]MailSyncMessage, error) {
	if folderID <= 0 {
		folderID = MailFolderInbox
	}
	var out []MailSyncMessage
	skipped := 0
	for page := 1; ; page++ {
		batch, err := c.ResponseArray(ctx,
			mailMessagesPath(MailMessagesFilter{Folder: folderID}, page, mailMessagesPageSize))
		if err != nil {
			return nil, fmt.Errorf("FetchMailFolder: %w", err)
		}
		if len(batch) == 0 {
			break
		}
		for _, raw := range batch {
			if skipped < opts.StartIndex {
				skipped++
				continue
			}
			msg, err := c.hydrateMailMessage(ctx, raw, opts)
			if err != nil {
				return nil, err
			}
			out = append(out, *msg)
			if opts.Limit > 0 && len(out) >= opts.Limit {
				return out, nil
			}
		}
		if len(batch) < mailMessagesPageSize {
			break
		}
	}
	return out, nil
}

// hydrateMailMessage converts one raw API message into a MailSyncMessage,
// fetching the full record when the list item does not carry the attachment
// metadata, and downloading bodies when requested.
func (c *Client) hydrateMailMessage(ctx context.Context, m map[string]any, opts MailSyncOptions) (*MailSyncMessage, error) {
	msg := &MailSyncMessage{
		ID:      Int64FromMap(m, "id"),
		Folder:  int(Int64FromMap(m, "folder")),
		Subject: stringFromMap(m, "subject"),
		From:    stringFromMap(m, "from"),
		IsNew:   boolFromMap(m, "isNew") == "true",
	}
	msg.Date = parseMailTime(stringFromMap(m, "date"))

	atts, _ := m["attachments"].([]any)
	hasFlag := boolFromMap(m, "hasAttachments") == "true"
	if hasFlag && len(atts) == 0 {
		// List items may omit the attachment array; pull the full record.
		full, err := c.GetMailMessage(ctx, strconv.FormatInt(msg.ID, 10))
		if err != nil {
			return nil, fmt.Errorf("FetchMailFolder: hydrate message %d: %w", msg.ID, err)
		}
		atts, _ = full["attachments"].([]any)
	}
	for _, a := range atts {
		am, ok := a.(map[string]any)
		if !ok {
			continue
		}
		att := MailSyncAttachment{
			ID:   mailAttachmentID(am),
			Name: stringFromMap(am, "fileName"),
			Size: Int64FromMap(am, "size"),
		}
		if att.Name == "" {
			att.Name = stringFromMap(am, "name")
		}
		if att.ID != "" {
			msg.Attachments = append(msg.Attachments, att)
		}
	}
	msg.HasAttachments = hasFlag || len(msg.Attachments) > 0

	if opts.FetchBodies {
		for i := range msg.Attachments {
			body, err := c.DownloadMailAttachment(ctx, msg.Attachments[i].ID)
			if err != nil {
				return nil, fmt.Errorf("FetchMailFolder: message %d attachment %q: %w",
					msg.ID, msg.Attachments[i].Name, err)
			}
			msg.Attachments[i].Body = body
		}
	}
	return msg, nil
}

// mailAttachmentID extracts the download id from an attachment object.
// OnlyOffice variants use "id", "fileId" or "attachmentId".
func mailAttachmentID(am map[string]any) string {
	for _, key := range []string{"id", "fileId", "attachmentId"} {
		switch v := am[key].(type) {
		case string:
			if s := v; s != "" {
				return s
			}
		case float64:
			if n := int64(v); n != 0 {
				return strconv.FormatInt(n, 10)
			}
		case int64:
			if v != 0 {
				return strconv.FormatInt(v, 10)
			}
		}
	}
	return ""
}

// parseMailTime accepts the OnlyOffice timestamp shapes seen in the wild:
// RFC3339 (with any fractional digits) and second-precision local form.
func parseMailTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t
	}
	return time.Time{}
}
