package onlyoffice

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ListProjectContacts returns CRM persons/companies linked to a project.
// GET /api/2.0/crm/contact/project/{projectid}
func (c *Client) ListProjectContacts(ctx context.Context, projectID int) ([]map[string]any, error) {
	return c.ResponseArray(ctx, fmt.Sprintf("/api/2.0/crm/contact/project/%d", projectID))
}

// AddProjectContact links a CRM contact (person or company) to a project.
// POST /api/2.0/project/{projectid}/contact  form: contactId=
func (c *Client) AddProjectContact(ctx context.Context, projectID, contactID int) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("contactId", strconv.Itoa(contactID))
	return c.postFormObject(ctx, fmt.Sprintf("/api/2.0/project/%d/contact", projectID), fields)
}

// RemoveProjectContact unlinks a CRM contact from a project.
// DELETE /api/2.0/project/{projectid}/contact?contactId=
func (c *Client) RemoveProjectContact(ctx context.Context, projectID, contactID int) error {
	_, err := c.deleteReq(ctx, fmt.Sprintf("/api/2.0/project/%d/contact?contactId=%d", projectID, contactID))
	return err
}

// GetProjectTeam returns portal users on the project team.
// GET /api/2.0/project/{projectid}/team
func (c *Client) GetProjectTeam(ctx context.Context, projectID int) ([]map[string]any, error) {
	return c.ResponseArray(ctx, fmt.Sprintf("/api/2.0/project/%d/team", projectID))
}
