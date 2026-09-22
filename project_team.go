package onlyoffice

// Project team (portal users) CRUD. Team is distinct from CRM contacts,
// which are linked through project_contacts.go.

import (
	"context"
	"fmt"
)

// ListProjectTeam returns portal users on the project team.
// GET /api/2.0/project/{projectid}/team
func (c *Client) ListProjectTeam(ctx context.Context, projectID int) ([]map[string]any, error) {
	return c.GetProjectTeam(ctx, projectID)
}

// AddProjectTeamUser adds a portal user to the project team.
// POST /api/2.0/project/{projectid}/team with {"userId": "..."}.
// Returns the resulting team list.
func (c *Client) AddProjectTeamUser(ctx context.Context, projectID int, userID string) ([]map[string]any, error) {
	return c.postJSONArray(ctx, fmt.Sprintf("/api/2.0/project/%d/team", projectID),
		map[string]any{"userId": userID})
}

// RemoveProjectTeamUser removes a portal user from the project team.
// DELETE /api/2.0/project/{projectid}/team with {"userId": "..."}.
// Returns the resulting team list.
func (c *Client) RemoveProjectTeamUser(ctx context.Context, projectID int, userID string) ([]map[string]any, error) {
	return c.deleteJSONArray(ctx, fmt.Sprintf("/api/2.0/project/%d/team", projectID),
		map[string]any{"userId": userID})
}

// SetProjectTeam replaces the project team with the given user IDs.
// PUT /api/2.0/project/{projectid}/team with participants + notify.
// This is the canonical way to register several members at once.
func (c *Client) SetProjectTeam(ctx context.Context, projectID int, participants []string, notify bool) ([]map[string]any, error) {
	return c.putJSONArray(ctx, fmt.Sprintf("/api/2.0/project/%d/team", projectID),
		map[string]any{"projectId": projectID, "participants": participants, "notify": notify})
}
