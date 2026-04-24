package onlyoffice

// Project / Milestone typed API.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Project struct
type Project struct {
	ID            *int            `json:"id"`
	Title         *string         `json:"title"`
	Security      map[string]bool `json:"security,omitempty"`
	ProjectFolder *json.Number    `json:"projectFolder,omitempty"`
	Description   *string         `json:"description"`
	Status        *int            `json:"status"`

	ResponsibleID *string `json:"responsibleId,omitempty"`
	Responsible   *User   `json:"responsible,omitempty"`

	IsPrivate *bool `json:"isPrivate"`

	TaskCount         *int    `json:"taskCount,omitempty"`
	TaskCountTotal    *int    `json:"taskCountTotal,omitempty"`
	MilestoneCount    *int    `json:"milestoneCount,omitempty"`
	DiscussionCount   *int    `json:"discussionCount,omitempty"`
	ParticipantCount  *int    `json:"participantCount,omitempty"`
	TimeTrackingTotal *string `json:"timeTrackingTotal,omitempty"`
	DocumentsCount    *int    `json:"documentsCount,omitempty"`

	IsFollow *bool `json:"isFollow,omitempty"`

	Created     *time.Time `json:"created"`
	CreatedBy   *User      `json:"createdBy,omitempty"`
	CreatedByID *string    `json:"createdById"`
	Updated     *time.Time `json:"updated"`
	UpdatedByID *string    `json:"updatedById"`

	Permissions *Permissions `json:",inline,omitempty"`
}

// String returns the project Title, or "" if the Title is nil.
func (p Project) String() string {
	if p.Title == nil {
		return ""
	}
	return *p.Title
}

// Projects is a slice with helpers for title lookup.
type Projects []*Project

// Get returns the first project whose title equals title, or nil.
func (p Projects) Get(title string) *Project {
	for _, prj := range p {
		if prj != nil && prj.Title != nil && *prj.Title == title {
			return prj
		}
	}
	return nil
}

// Milestone is a project milestone.
type Milestone struct {
	ID          *int64     `json:"id,omitempty"`
	Description *string    `json:"description,omitempty"`
	Title       *string    `json:"title,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`

	IsKey    *bool `json:"isKey,omitempty"`
	IsNotify *bool `json:"isNotify,omitempty"`

	ProjectOwner *ProjectOwner `json:"projectOwner,omitempty"`
	Responsible  *User         `json:"responsible,omitempty"`

	ActiveTaskCount *int64 `json:"activeTaskCount,omitempty"`
	ClosedTaskCount *int64 `json:"closedTaskCount,omitempty"`
	Status          *int64 `json:"status,omitempty"`

	Created   *time.Time `json:"created,omitempty"`
	CreatedBy *User      `json:"createdBy,omitempty"`

	Updated *time.Time `json:"updated,omitempty"`

	*Permissions `json:",inline,omitempty"`
}

// ProjectOwner is a compact project reference used by Milestone/Task.
type ProjectOwner struct {
	ID        *int    `json:"id,omitempty"`
	Title     *string `json:"title,omitempty"`
	Status    *int    `json:"status,omitempty"`
	IsPrivate *bool   `json:"isPrivate,omitempty"`
}

// NewProjectRequest is the payload for CreateProject.
type NewProjectRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ResponsibleID string `json:"responsibleId"`
}

// ProjectUpdateRequest is the payload for UpdateProject. Only non-empty
// fields are transmitted (enforced by omitempty).
type ProjectUpdateRequest struct {
	ID            int    `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	ResponsibleID string `json:"responsibleId,omitempty"`
}

// GetProjects returns all projects, including private ones the caller can see.
func (c *Client) GetProjects() (list Projects, err error) {
	return list, c.Query(Request{Uri: `/api/2.0/project/filter.json?simple=true`},
		&struct {
			MetaResponse `json:",inline"`
			Response     *Projects
		}{Response: &list})
}

// GetProjectByID returns a single project as an untyped map. The typed
// counterpart is not currently provided; callers can iterate GetProjects and
// match by title or write their own typed wrapper.
//
// When projectID is empty the configured default is used.
func (c *Client) GetProjectByID(ctx context.Context, projectID string) (map[string]any, error) {
	if projectID == "" {
		projectID = c.defaults.ProjectID
	}
	return c.ResponseObject(ctx, fmt.Sprintf("/api/2.0/project/%s.json", url.PathEscape(projectID)))
}

// GetProjectMilestones returns milestones for the given project.
// https://api1.onlyoffice.com/portals/method/project/post/api/2.0/project/%7bid%7d/milestone
func (c *Client) GetProjectMilestones(project *Project) ([]*Milestone, error) {
	var list []*Milestone
	err := c.Query(Request{Uri: fmt.Sprintf(`/api/2.0/project/%d/milestone`, *project.ID)},
		&struct {
			MetaResponse `json:",inline"`
			Response     *[]*Milestone
		}{Response: &list})
	return list, err
}

// CreateProject creates a new project.
//   - if ResponsibleID is empty, the first user matching the client's User
//     email is picked; failing that, the first portal user.
func (c *Client) CreateProject(np NewProjectRequest) (*Project, error) {
	if np.ResponsibleID == "" {
		users, err := c.GetUsers()
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			if u.Email != nil && *u.Email == c.credentials.User {
				np.ResponsibleID = *u.ID
				break
			}
		}
		if np.ResponsibleID == "" && len(users) > 0 && users[0].ID != nil {
			np.ResponsibleID = *users[0].ID
		}
	}

	prj := new(Project)
	return prj, c.Query(Request{
		Uri:    "/api/2.0/project.json",
		Method: "POST",
		Body:   np,
	}, &struct {
		MetaResponse `json:",inline"`
		Response     *Project `json:"response"`
	}{
		Response: prj,
	})
}

// DeleteProject deletes a project by numeric ID.
func (c *Client) DeleteProject(id int) (*Project, error) {
	p := &Project{}
	return p, c.Query(
		Request{
			Uri:    fmt.Sprintf("/api/2.0/project/%d.json", id),
			Method: "DELETE",
		},
		&struct {
			Response *Project `json:"response"`
		}{p})
}

// UpdateProject updates project fields.
func (c *Client) UpdateProject(req ProjectUpdateRequest) (*Project, error) {
	p := &Project{}
	return p, c.Query(Request{
		Uri:    fmt.Sprintf("/api/2.0/project/%d.json", req.ID),
		Method: "PUT",
		Body:   req,
	},
		&struct {
			Response *Project `json:"response"`
		}{p})
}
