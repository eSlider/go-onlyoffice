package onlyoffice

// Task-related helpers that complement the typed Project/Task API in
// onlyoffice.go. These return untyped maps and use form-encoded endpoints,
// matching the Python cv/bin/office reference and the OnlyOffice web UI.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetProjectByID returns a single project as an untyped map.
// (The typed counterpart GetProject is not provided; callers that want typed
// structs should use Projects + iteration or add their own typed wrappers.)
func (c *Client) GetProjectByID(ctx context.Context, projectID string) (map[string]any, error) {
	if projectID == "" {
		projectID = c.defaults.ProjectID
	}
	raw, err := c.getJSON(ctx, fmt.Sprintf("/api/2.0/project/%s.json", url.PathEscape(projectID)))
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

// ListTasks lists tasks in a single project. When projectID is empty the
// configured default is used. status accepts "open"/"closed" or a numeric code.
func (c *Client) ListTasks(ctx context.Context, projectID, status string) ([]map[string]any, error) {
	if projectID == "" {
		projectID = c.defaults.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("ListTasks: projectID is required (pass explicitly or set via SetDefaults)")
	}
	path := fmt.Sprintf("/api/2.0/project/%s/task.json", url.PathEscape(projectID))
	if status != "" {
		code := map[string]string{"open": "0", "closed": "1"}[status]
		if code == "" {
			code = status
		}
		path += "?status=" + url.QueryEscape(code)
	}
	return c.ResponseArray(ctx, path)
}

// ListAllTasks lists tasks across projects for the authenticated user.
func (c *Client) ListAllTasks(ctx context.Context, status string) ([]map[string]any, error) {
	path := "/api/2.0/project/task/@self.json"
	if status != "" {
		code := map[string]string{"open": "0", "closed": "1"}[status]
		if code == "" {
			code = status
		}
		path += "?status=" + url.QueryEscape(code)
	}
	return c.ResponseArray(ctx, path)
}

// GetTaskByID returns a task as an untyped map, including its subtasks.
func (c *Client) GetTaskByID(ctx context.Context, taskID string) (map[string]any, error) {
	raw, err := c.getJSON(ctx, fmt.Sprintf("/api/2.0/project/task/%s.json", url.PathEscape(taskID)))
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

// AddTask creates a task via the form-encoded endpoint (no milestone/start).
// Prefer the typed CreateProjectTask for new code; AddTask is kept for parity
// with the Python reference tooling.
func (c *Client) AddTask(ctx context.Context, projectID, title, description string, priority int, deadline string) (map[string]any, error) {
	if projectID == "" {
		projectID = c.defaults.ProjectID
	}
	fields := url.Values{}
	fields.Set("title", title)
	fields.Set("description", description)
	fields.Set("priority", fmt.Sprintf("%d", priority))
	if deadline != "" {
		fields.Set("deadline", deadline)
	}
	raw, err := c.postForm(ctx, fmt.Sprintf("/api/2.0/project/%s/task.json", url.PathEscape(projectID)), fields)
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

// AddSubtask creates a subtask under parentTaskID.
func (c *Client) AddSubtask(ctx context.Context, parentTaskID, title string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("title", title)
	raw, err := c.postForm(ctx, fmt.Sprintf("/api/2.0/project/task/%s.json", url.PathEscape(parentTaskID)), fields)
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

// UpdateTaskStatus changes task status. status accepts "open"/"closed" (mapped
// to 1/2) or a raw numeric code passed through.
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID, status string) (map[string]any, error) {
	code := map[string]string{"open": "1", "closed": "2"}[status]
	if code == "" {
		code = status
	}
	fields := url.Values{}
	fields.Set("status", code)
	raw, err := c.putForm(ctx, fmt.Sprintf("/api/2.0/project/task/%s/status.json", url.PathEscape(taskID)), fields)
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	var out map[string]any
	_ = json.Unmarshal(resp, &out)
	return out, nil
}

// DeleteTask removes a project task by ID.
func (c *Client) DeleteTask(ctx context.Context, taskID string) (map[string]any, error) {
	raw, err := c.deleteReq(ctx, fmt.Sprintf("/api/2.0/project/task/%s.json", url.PathEscape(taskID)))
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	var out map[string]any
	_ = json.Unmarshal(resp, &out)
	return out, nil
}
