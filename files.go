package onlyoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// UploadOpportunityFile uploads a single file to a CRM opportunity.
// Returns the decoded "response" object from OnlyOffice.
func (c *Client) UploadOpportunityFile(ctx context.Context, opportunityID, filePath string) (map[string]any, error) {
	path := fmt.Sprintf("/api/2.0/crm/opportunity/%s/files/upload.json", url.PathEscape(opportunityID))
	raw, err := c.uploadMultipart(ctx, path, "file", filePath)
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
