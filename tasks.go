package onlyoffice

// Project task API — both typed (Task + CreateProjectTask / UpdateProjectTask
// / GetTasks) and untyped form-endpoint helpers (AddTask, AddSubtask,
// UpdateTaskStatus, DeleteTask, ListTasks, …). The typed path mirrors the
// JSON responses; the form-encoded path mirrors the Python cv/bin/office
// reference and the OnlyOffice web UI.

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Task is the JSON-mapped project task as returned by /project/task/*.
type Task struct {
	ID           *int          `json:"id,omitempty"`
	Title        *string       `json:"title,omitempty"`
	StartDate    *time.Time    `json:"startDate,omitempty"`
	Deadline     *time.Time    `json:"deadline,omitempty"`
	Description  *string       `json:"description,omitempty"`
	Priority     *int          `json:"priority,omitempty"`
	ProjectOwner *ProjectOwner `json:"projectOwner,omitempty"`

	Subtasks []any `json:"subtasks,omitempty"`

	Status *ProjectTaskStatus `json:"status,omitempty"`

	Created     *time.Time `json:"created,omitempty"`
	CreatedBy   *User      `json:"createdBy,omitempty"`
	CreatedByID *string    `json:"createdById,omitempty"` // UUID

	Updated     *time.Time `json:"updated,omitempty"`
	UpdatedBy   *User      `json:"updatedBy,omitempty"`
	UpdatedById *string    `json:"updatedById,omitempty"` // UUID

	Responsibles   []*User  `json:"responsibles,omitempty"`
	ResponsibleIDS []string `json:"responsibleIds,omitempty"` // UUID list

	CanEdit            *bool `json:"canEdit,omitempty"`
	CanCreateSubtask   *bool `json:"canCreateSubtask,omitempty"`
	CanCreateTimeSpend *bool `json:"canCreateTimeSpend,omitempty"`
	CanDelete          *bool `json:"canDelete,omitempty"`
	CanReadFiles       *bool `json:"canReadFiles,omitempty"`

	MilestoneID *int64     `json:"milestoneId,omitempty"`
	Milestone   *Milestone `json:"milestone,omitempty"`
}

// TaskPriority values: High = 1, Normal = 0, Low = -1.
type TaskPriority int

const (
	TaskPriorityHigh   TaskPriority = 1
	TaskPriorityNormal TaskPriority = 0
	TaskPriorityLow    TaskPriority = -1
)

// ProjectTaskStatus encodes OnlyOffice task status codes.
type ProjectTaskStatus int

const (
	ProjectTaskStatusNotAccept      ProjectTaskStatus = 0
	ProjectTaskStatusOpen           ProjectTaskStatus = 1
	ProjectTaskStatusClosed         ProjectTaskStatus = 2
	ProjectTaskStatusDisable        ProjectTaskStatus = 3
	ProjectTaskStatusUnclassified   ProjectTaskStatus = 4
	ProjectTaskStatusNotInMilestone ProjectTaskStatus = 5
)

// GiteaIssue2OnlyOfficeMappingRegExp extracts a "URL:" footer pointing at a
// Gitea issue, used by external sync tooling (inventar-sync et al.).
var GiteaIssue2OnlyOfficeMappingRegExp = regexp.MustCompile(`URL:(.*)$`)

// GetGiteaIssueLink returns the first Gitea URL embedded in the task
// description via the "URL:<url>" convention, or "" if absent.
func (t *Task) GetGiteaIssueLink() string {
	if t.Description == nil {
		return ""
	}
	m := GiteaIssue2OnlyOfficeMappingRegExp.FindStringSubmatch(*t.Description)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// NewProjectTaskRequest creates a new task via the typed JSON API.
type NewProjectTaskRequest struct {
	Title       string `url:"title"`
	Description string `url:"description"`
	Notify      bool   `url:"notify"`
	MilestoneId int    `url:"milestoneId"`
	Priority    int    `url:"priority"`
	ProjectId   int    `url:"projectId"`
	StartDate   Time   `url:"startDate"`
	Deadline    Time   `url:"deadline"`
	Status      ProjectTaskStatus
}

// ProjectTaskUpdateRequest updates an existing task.
// https://api1.onlyoffice.com/portals/method/project/put/api/2.0/project/task/%7btaskid%7d
type ProjectTaskUpdateRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Priority    *int   `json:"priority,omitempty"`
	StartDate   *Time  `json:"startDate,omitempty"`
	Deadline    *Time  `json:"deadline,omitempty"`

	ProjectID   *int64            `json:"projectID,omitempty"`
	MilestoneId *int64            `json:"milestoneid,omitempty"`
	Responsible []string          `json:"responsibles,omitempty"` // UUID list
	Notify      bool              `json:"notify,omitempty"`
	Status      ProjectTaskStatus `json:"status,omitempty"`
}

// ProjectGetTasksRequest is the filter payload for GetTasks.
// See https://api.onlyoffice.com/workspace/api-backend/usage-api/project/tasks/get-tasks-by-status/
type ProjectGetTasksRequest struct {
	ProjectId  int    `url:"projectId"`
	Count      int    `url:"count"`
	StartIndex int    `url:"startIndex"`
	SortBy     string `url:"sortBy"`
	SortOrder  string `url:"sortOrder"`
	Simple     bool   `url:"simple"`
}

// NewProjectGetTasksRequest builds a simple "all tasks, sorted by title"
// request pre-populated with sane defaults (count=1000).
func NewProjectGetTasksRequest(projectId int) ProjectGetTasksRequest {
	return ProjectGetTasksRequest{
		ProjectId:  projectId,
		Count:      1000,
		StartIndex: 0,
		SortBy:     "title",
		SortOrder:  "ascending",
		Simple:     true,
	}
}

// CreateProjectTask creates a project task via the typed JSON API.
func (c *Client) CreateProjectTask(req NewProjectTaskRequest) (*Task, error) {
	task := &Task{}
	return task, c.Query(Request{
		Uri:    fmt.Sprintf("/api/2.0/project/%d/task.json", req.ProjectId),
		Method: "POST",
		Body:   req,
	}, &struct {
		Response *Task `json:"response"`
	}{task})
}

// UpdateProjectTask updates task fields via the typed JSON API.
func (c *Client) UpdateProjectTask(req ProjectTaskUpdateRequest) (*Task, error) {
	task := &Task{}
	return task, c.Query(
		Request{
			Uri:    fmt.Sprintf("/api/2.0/project/task/%d.json", req.ID),
			Method: "PUT",
			Body:   req,
		}, &struct {
			Response *Task `json:"response"`
		}{task})
}

// GetTasks returns a list of tasks for a project matching the given filter.
func (c *Client) GetTasks(req ProjectGetTasksRequest) (tasks []*Task, err error) {
	return tasks, c.Query(
		Request{
			Uri:    "/api/2.0/project/task/filter.json",
			Params: req,
		},
		&struct {
			Response *[]*Task `json:"response"`
		}{&tasks})
}

// -----------------------------------------------------------------------------
// Untyped form-endpoint helpers (Python cv/bin/office parity, web UI parity)
// -----------------------------------------------------------------------------

// ListTasks lists tasks in a single project. When projectID is empty the
// configured default is used. status accepts "open"/"closed" or a numeric
// code.
func (c *Client) ListTasks(ctx context.Context, projectID, status string) ([]map[string]any, error) {
	if projectID == "" {
		projectID = c.defaults.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("ListTasks: projectID is required (pass explicitly or set via SetDefaults)")
	}
	path := fmt.Sprintf("/api/2.0/project/%s/task.json", url.PathEscape(projectID))
	if code := taskStatusCode(status); code != "" {
		path += "?status=" + url.QueryEscape(code)
	}
	return c.ResponseArray(ctx, path)
}

// ListAllTasks lists tasks across projects for the authenticated user.
func (c *Client) ListAllTasks(ctx context.Context, status string) ([]map[string]any, error) {
	path := "/api/2.0/project/task/@self.json"
	if code := taskStatusCode(status); code != "" {
		path += "?status=" + url.QueryEscape(code)
	}
	return c.ResponseArray(ctx, path)
}

// taskStatusCode maps "open"/"closed" to their numeric codes, passes any
// other non-empty value through verbatim, and returns "" for an empty input.
func taskStatusCode(status string) string {
	switch status {
	case "":
		return ""
	case "open":
		return "0"
	case "closed":
		return "1"
	default:
		return status
	}
}

// GetTaskByID returns a task as an untyped map, including its subtasks.
func (c *Client) GetTaskByID(ctx context.Context, taskID string) (map[string]any, error) {
	return c.ResponseObject(ctx, fmt.Sprintf("/api/2.0/project/task/%s.json", url.PathEscape(taskID)))
}

// AddTask creates a task via the form-encoded endpoint (no milestone/start).
// Prefer the typed CreateProjectTask for new code; AddTask is kept for
// parity with the Python reference tooling.
func (c *Client) AddTask(ctx context.Context, projectID, title, description string, priority int, deadline string) (map[string]any, error) {
	if projectID == "" {
		projectID = c.defaults.ProjectID
	}
	fields := url.Values{}
	fields.Set("title", title)
	fields.Set("description", description)
	fields.Set("priority", strconv.Itoa(priority))
	if deadline != "" {
		fields.Set("deadline", deadline)
	}
	return c.postFormObject(ctx, fmt.Sprintf("/api/2.0/project/%s/task.json", url.PathEscape(projectID)), fields)
}

// AddSubtask creates a subtask under parentTaskID.
func (c *Client) AddSubtask(ctx context.Context, parentTaskID, title string) (map[string]any, error) {
	fields := url.Values{}
	fields.Set("title", title)
	return c.postFormObject(ctx, fmt.Sprintf("/api/2.0/project/task/%s.json", url.PathEscape(parentTaskID)), fields)
}

// UpdateTaskStatus changes task status. status accepts "open"/"closed"
// (mapped to 1/2) or a raw numeric code passed through.
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID, status string) (map[string]any, error) {
	code := status
	switch status {
	case "open":
		code = "1"
	case "closed":
		code = "2"
	}
	fields := url.Values{}
	fields.Set("status", code)
	return c.putFormObject(ctx, fmt.Sprintf("/api/2.0/project/task/%s/status.json", url.PathEscape(taskID)), fields)
}

// DeleteTask removes a project task by ID.
func (c *Client) DeleteTask(ctx context.Context, taskID string) (map[string]any, error) {
	return c.deleteObject(ctx, fmt.Sprintf("/api/2.0/project/task/%s.json", url.PathEscape(taskID)))
}
