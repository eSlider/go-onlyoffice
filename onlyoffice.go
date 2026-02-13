package onlyoffice

// OnlyOffice client package

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-querystring/query"
	"io"
	"net/http"
	"os"
	regexp "regexp"
	"strings"
	"time"
)

// NewClient API
func NewClient(c Credentials) *Client {
	return &Client{
		client:      http.DefaultClient,
		credentials: &c,
	}
}

// GetEnvironmentCredentials when using environment variables
func GetEnvironmentCredentials() Credentials {
	return Credentials{
		Url:      os.Getenv("ONLYOFFICE_URL"),
		User:     os.Getenv("ONLYOFFICE_USER"),
		Password: os.Getenv("ONLYOFFICE_PASS"),
	}

}

// Credentials of OnlyOffice User
type Credentials struct {
	Url      string `json:"-"`
	User     string `json:"userName"`
	Password string `json:"password"`
}

// ToJson for Credentials
func (c Credentials) ToJson() []byte {
	b, err := json.Marshal(c)
	if err != nil {
		return nil
	}
	return b
}

// Client of OnlyOffice API uses credentials to get a token and query the API by every request
type Client struct {
	client      *http.Client // HTTP client
	credentials *Credentials // OnlyOffice credentials
	token       *Token       // Authentication token
}

// MetaResponse Response
type MetaResponse struct {
	Count      int `json:"count"`
	Total      int `json:"total"`
	Status     int `json:"status"`
	StatusCode int `json:"statusCode"`
}

type Permissions struct {
	CanEdit   *bool `json:"canEdit,omitempty"`   // Is user can edit
	CanDelete *bool `json:"canDelete,omitempty"` // Is user can delete
}

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

type User struct {
	ID               *string    `json:"id,omitempty"`
	UserName         *string    `json:"userName,omitempty"`
	IsVisitor        *bool      `json:"isVisitor,omitempty"`
	FirstName        *string    `json:"firstName,omitempty"`
	LastName         *string    `json:"lastName,omitempty"`
	Email            *string    `json:"email,omitempty"`
	Status           *int       `json:"status,omitempty"`
	ActivationStatus *int       `json:"activationStatus,omitempty"`
	Terminated       any        `json:"terminated,omitempty"`
	Department       *string    `json:"department,omitempty"`
	WorkFrom         *time.Time `json:"workFrom,omitempty"`
	DisplayName      *string    `json:"displayName,omitempty"`
	AvatarMedium     *string    `json:"avatarMedium,omitempty"`
	Avatar           *string    `json:"avatar,omitempty"`
	IsAdmin          *bool      `json:"isAdmin,omitempty"`
	IsLDAP           *bool      `json:"isLDAP,omitempty"`
	ListAdminModules []string   `json:"listAdminModules,omitempty"`
	IsOwner          *bool      `json:"isOwner,omitempty"`
	CultureName      *string    `json:"cultureName,omitempty"`
	IsSSO            *bool      `json:"isSSO,omitempty"`
	AvatarSmall      *string    `json:"avatarSmall,omitempty"`
	QuotaLimit       *int       `json:"quotaLimit,omitempty"`
	UsedSpace        *int       `json:"usedSpace,omitempty"`
	DocsSpace        *int       `json:"docsSpace,omitempty"`
	MailSpace        *int       `json:"mailSpace,omitempty"`
	TalkSpace        *int       `json:"talkSpace,omitempty"`
	ProfileURL       *string    `json:"profileUrl,omitempty"`
	Title            *string    `json:"title,omitempty"`
	Sex              *string    `json:"sex,omitempty"`
	Lead             *string    `json:"lead,omitempty"`
	Birthday         *time.Time `json:"birthday,omitempty"`
	Location         *string    `json:"location,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	Contacts         []Contact  `json:"contacts,omitempty"`
	Groups           []Group    `json:"groups,omitempty"`
}

type Contact struct {
	Type  *string `json:"type,omitempty"`
	Value *string `json:"value,omitempty"`
}

type Group struct {
	ID      *string     `json:"id,omitempty"`
	Name    *string     `json:"name,omitempty"`
	Manager interface{} `json:"manager,omitempty"`
}

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

var GiteaIssue2OnlyOfficeMappingRegExp = regexp.MustCompile(`URL:(.*)$`)
var TestReg = `

URL:https://git.markets-platform.com/TradePlatform/email-templates/issues/1`

// GetGiteaIssueLink from task description
func (t *Task) GetGiteaIssueLink() string {

	// Get issue from description
	if t.Description != nil {
		var match = GiteaIssue2OnlyOfficeMappingRegExp.FindStringSubmatch(*t.Description)
		if len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

// TaskPriority High = 1, Normal = 0, Low = -1
type TaskPriority int

const (
	TaskPriorityHigh   TaskPriority = 1 // High
	TaskPriorityNormal TaskPriority = 0 // Normal
	TaskPriorityLow    TaskPriority = -1
)

type ProjectOwner struct {
	ID        *int    `json:"id,omitempty"`
	Title     *string `json:"title,omitempty"`
	Status    *int    `json:"status,omitempty"`
	IsPrivate *bool   `json:"isPrivate,omitempty"`
}

// String for Project to return title
func (p Project) String() string {
	return fmt.Sprintf(*p.Title)
}

// Token OnlyOffice
type Token struct {
	Value   string `json:"token"`
	Expires Time   `json:"expires"`
}

// Time for OnlyOffice
type Time time.Time

// To String for Time
func (t Time) String() string {
	return time.Time(t).Format("2006-01-02T15:04:05")
}

func (t Time) Before(u Time) bool {
	return time.Time(t).Before(time.Time(u))
}

func (t Time) After(u Time) bool {
	return time.Time(t).After(time.Time(u))
}

// UnmarshalJSON for Time
func (r *Time) UnmarshalJSON(data []byte) error {
	// Trim quotes
	data = data[1 : len(data)-1]
	t, err := time.Parse("2006-01-02T15:04:05.0000000-07:00", string(data))
	if err != nil {
		return err
	}
	*r = Time(t)
	return nil
}

// Request for OnlyOffice API
type Request struct {
	Uri    string  // URI is the path to the API endpoint e.g. /api/2.0/project.json
	Method string  // Method is the HTTP method e.g. GET, POST, PUT, DELETE
	Params any     // Params is the query parameters
	Body   any     // Body could be a struct, map, []byte, or string. It will be marshaled to JSON
	Token  *string // Token is the Authorization header, if not set then it will get a token
	NoAuth bool    // NoAuth for skipping automatic authentication
	Debug  bool
}

// GetMethod for Request
func (r Request) GetMethod() string {
	if r.Method == "" {
		return "GET"
	}
	return r.Method
}

// Query the OnlyOffice API
//   - If request.Method is not set then it will default to GET
//   - If request.Code is not nil then it will be marshaled to JSON
//   - If request.Token is nil then it will get a token
//   - If request.Token is not nil then it will be used as Authorization header
//   - If request.NoAuth is true then it will skip automatic authentication
func (c *Client) Query(request Request, result interface{}) (err error) {
	var url = fmt.Sprintf("%s%s", c.credentials.Url, request.Uri)
	var rdr io.Reader = nil
	var jsonRequestBody string

	// Add query parameters if available
	if request.Params != nil {
		v, err := query.Values(request.Params)
		if err != nil {
			return err
		}
		url = fmt.Sprintf("%s?%s", url, v.Encode())
	}

	// Create request
	if request.Body != nil {
		// Check request body if type is not []byte or string the marshal it to JSON
		switch request.Body.(type) {
		case []byte:
			jsonRequestBody = string(request.Body.([]byte))
			rdr = strings.NewReader(string(request.Body.([]byte)))
		case string:
			jsonRequestBody = request.Body.(string)
			rdr = strings.NewReader(request.Body.(string))
		default:
			// Marshal to JSON
			b, err := json.Marshal(request.Body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %v", err)
			}
			jsonRequestBody = string(b)
			rdr = strings.NewReader(string(b))
		}
	}

	req, err := http.NewRequest(request.GetMethod(), url, rdr)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Pragma", "no-cache")

	// Get token if not set?
	if !request.NoAuth {
		// Get token if not set or expired
		if c.token == nil || time.Time(c.token.Expires).Before(time.Now()) {
			c.token, err = c.Auth(c.credentials)
			if err != nil {
				return fmt.Errorf("failed to authenticate: %v", err)
			}
		}
	}

	// Set token if available
	if request.Token != nil {
		req.Header.Set("Authorization", *request.Token)
	} else {
		// Set token from a client if available
		if c.token != nil {
			req.Header.Set("Authorization", c.token.Value)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if result == nil {
		return nil
	}
	if request.Debug {
		// Rewind reader
		_ = jsonRequestBody
		var buf = new(strings.Builder)
		io.Copy(buf, resp.Body)
		js := buf.String()
		return json.Unmarshal([]byte(js), result)
	} else {
		// Unmarshal response using reader
		return json.NewDecoder(resp.Body).Decode(result)
	}

}

// Auth to authenticate by getting a token using credentials
func (c *Client) Auth(creds *Credentials) (t *Token, err error) {
	t = &Token{}
	return t, c.Query(Request{
		Uri:    "/api/2.0/authentication.json",
		Method: "POST",
		Body:   creds,
		NoAuth: true,
	}, &struct { // Unmarshal response into a struct
		MetaResponse `json:",inline"`
		Response     *Token `json:"response"`
	}{
		Response: t,
	})
}

// Projects list
type Projects []*Project

// Get Project by title
func (p Projects) Get(title string) *Project {
	for _, prj := range p {
		if *prj.Title == title {
			return prj
		}
	}
	return nil
}

// GetProjects
//   - Get all projects, not excluding private projects
func (c *Client) GetProjects() (list Projects, err error) {
	return list, c.Query(Request{Uri: `/api/2.0/project/filter.json?simple=true`},
		&struct {
			MetaResponse `json:",inline"`
			Response     *Projects
		}{Response: &list})
}

// GetProjectMilestones Get project milestones
// https://api1.onlyoffice.com/portals/method/project/post/api/2.0/project/%7bid%7d/milestone
func (c *Client) GetProjectMilestones(project *Project) ([]*Milestone, error) {
	var list []*Milestone

	err := c.Query(Request{Uri: fmt.Sprintf(`/api/2.0/project/%d/milestone`, *project.ID), Debug: false},
		&struct {
			MetaResponse `json:",inline"`
			//Response     *[]*Milestone
			Response *[]*Milestone
		}{Response: &list})
	return list, err
}

// GetUsers List users
func (c *Client) GetUsers() (list []*User, err error) {
	return list, c.Query(Request{Uri: "/api/2.0/people/filter.json"},
		&struct {
			MetaResponse `json:",inline"`
			Response     *[]*User `json:"response"`
		}{Response: &list})
}

// NewProjectRequest
type NewProjectRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ResponsibleID string `json:"responsibleId"`
}

// ProjectUpdateRequest
type ProjectUpdateRequest struct {
	ID            int    `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	ResponsibleID string `json:"responsibleId,omitempty"`
}

// NewProjectTaskRequest
type NewProjectTaskRequest struct {
	Title       string `url:"title"`
	Description string `url:"description"`
	Notify      bool   `url:"notify"`
	MilestoneId int    `url:"milestoneId"`
	Priority    int    `url:"priority"`
	ProjectId   int    `url:"projectId"`
	StartDate   Time   `url:"startDate"` // 2024-04-16T00:00:00
	Deadline    Time   `url:"deadline"`  // 2024-04-20T00:00:00
	Status      ProjectTaskStatus
}

type ProjectTaskStatus int

const (
	ProjectTaskStatusNotAccept      ProjectTaskStatus = 0
	ProjectTaskStatusOpen           ProjectTaskStatus = 1
	ProjectTaskStatusClosed         ProjectTaskStatus = 2
	ProjectTaskStatusDisable        ProjectTaskStatus = 3
	ProjectTaskStatusUnclassified   ProjectTaskStatus = 4
	ProjectTaskStatusNotInMilestone ProjectTaskStatus = 5
)

// ProjectTaskUpdateRequest - Update a task
// See: https://api1.onlyoffice.com/portals/method/project/put/api/2.0/project/task/%7btaskid%7d
type ProjectTaskUpdateRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Priority    *int   `json:"priority,omitempty"` // optional, New task priority	{"0"} High = 1, Normal = 0, Low = -1
	StartDate   *Time  `json:"startDate,omitempty"`
	Deadline    *Time  `json:"deadline,omitempty"`
	//Deadline    *Time             `json:"closed,omitempty"`
	ProjectID   *int64            `json:"projectID,omitempty"` // New task project ID (optional)
	MilestoneId *int64            `json:"milestoneid,omitempty"`
	Responsible []string          `json:"responsibles,omitempty"` // New list of task responsibles	{"9924256A-739C-462b-AF15-E652A3B1B6EB"}
	Notify      bool              `json:"notify,omitempty"`
	Status      ProjectTaskStatus `json:"status,omitempty"`
	//Progress     int64    `json:"progress,omitempty"` // New task progress
}

// MarshalJSON to quote Time as string
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(
		`"%s"`,
		time.Time(t).Format("2006-01-02T15:04:05"),
	)), nil
}

// CreateProject with a new project request
//   - if responsibleID is empty then it will use the first user as responsible
//   - if responsibleID is empty and user is not found then it will use the current user
func (c *Client) CreateProject(np NewProjectRequest) (prj *Project, err error) {
	// We need a responsible user
	if np.ResponsibleID == "" {
		var users []*User
		users, err = c.GetUsers()
		if err != nil {
			return
		}

		// Find user by email
		for _, user := range users {
			if *user.Email == c.credentials.User {
				np.ResponsibleID = *user.ID
				break
			}
		}

		// Has no user?
		if np.ResponsibleID == "" {
			// Set first user as responsible
			np.ResponsibleID = *users[0].ID
		}
	}

	prj = new(Project)
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

func (c *Client) CreateProjectTask(req NewProjectTaskRequest) (task *Task, err error) {
	task = &Task{}
	return task, c.Query(Request{
		Uri:    fmt.Sprintf("/api/2.0/project/%d/task.json", req.ProjectId),
		Method: "POST",
		Body:   req,
	}, &struct {
		//MetaResponse `json:",inline"`
		Response *Task `json:"response"`
	}{task})
}

func (c *Client) DeleteProject(id int) (p *Project, err error) {
	p = &Project{}
	return p, c.Query(
		Request{
			Uri:    fmt.Sprintf("/api/2.0/project/%d.json", id),
			Method: "DELETE"},
		&struct {
			//MetaResponse `json:",inline"`
			Response *Project `json:"response"`
		}{p})
}

func (c *Client) UpdateProject(req ProjectUpdateRequest) (p *Project, err error) {
	p = &Project{}
	return p, c.Query(Request{
		Uri:    fmt.Sprintf("/api/2.0/project/%d.json", req.ID),
		Method: "PUT",
		Body:   req,
	},
		&struct {
			//MetaResponse `json:",inline"`
			Response *Project `json:"response"`
		}{p})
}

func (c *Client) UpdateProjectTask(req ProjectTaskUpdateRequest) (task *Task, err error) {
	task = &Task{}
	return task, c.Query(
		Request{
			Uri:    fmt.Sprintf("/api/2.0/project/task/%d.json", req.ID),
			Method: "PUT",
			Body:   req,
			Debug:  true,
		}, &struct {
			//MetaResponse `json:",inline"`
			Response *Task `json:"response"`
		}{task})
}

// ProjectGetTasksRequest
// See: https://api.onlyoffice.com/workspace/api-backend/usage-api/project/tasks/get-tasks-by-status/
type ProjectGetTasksRequest struct {
	ProjectId  int    `url:"projectId"`
	Count      int    `url:"count"`
	StartIndex int    `url:"startIndex"`
	SortBy     string `url:"sortBy"`
	SortOrder  string `url:"sortOrder"`
	Simple     bool   `url:"simple"`
}

// NewProjectGetTasksRequest to get a simple task list
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

// GetTasks returns a list of all the tasks from a project with the ID specified in the request.
func (c *Client) GetTasks(req ProjectGetTasksRequest) (tasks []*Task, err error) {
	return tasks, c.Query(
		Request{
			Uri:    "/api/2.0/project/task/filter.json",
			Params: req,
			Debug:  true,
		},
		&struct {
			//MetaResponse `json:",inline"`
			Response *[]*Task `json:"response"`
		}{&tasks})
}
