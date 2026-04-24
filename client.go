// Package onlyoffice provides a Go client for the OnlyOffice Workspace
// (formerly ONLYOFFICE) REST API. It is organised as a flat single package
// intentionally: a single *Client exposes Projects, Tasks, Calendar, CRM and
// Files operations as receiver methods. Domain split is by file, not by
// subpackage, to keep call sites uniform (c.ListContacts, c.GetTasks,
// c.AddEvent, c.UploadOpportunityFile all live on the same handle).
//
// The CLI binary `oo` (see cmd/oo) is a thin cobra wrapper on top of this
// library and mirrors the Python cv/bin/office reference tooling.
package onlyoffice

import (
	"net/http"
	"os"
	"strings"
)

// Client of OnlyOffice API uses credentials to get a token and query the API
// by every request.
//
// Construct with NewClient; optionally set fallbacks via SetDefaults. The
// zero value is NOT usable — credentials are required. A single client is
// safe for sequential use from one goroutine; for concurrent use, callers
// should wrap with their own synchronization or create one client per
// goroutine.
type Client struct {
	client      *http.Client
	credentials *Credentials
	token       *Token

	defaults  Defaults // optional fallbacks for calendar/project IDs
	selfID    string   // cached /api/2.0/people/@self id
	noteCatID int      // cached CRM history category id for "note"
}

// NewClient returns a new Client backed by http.DefaultClient.
func NewClient(c Credentials) *Client {
	return &Client{
		client:      http.DefaultClient,
		credentials: &c,
	}
}

// Credentials of OnlyOffice User. The Url field is NOT sent with the auth
// payload — it only determines the host.
type Credentials struct {
	Url      string `json:"-"`
	User     string `json:"userName"`
	Password string `json:"password"`
}

// Defaults holds optional fallbacks used by package-level helpers when callers
// pass an empty identifier (calendar or project). Set via (*Client).SetDefaults
// or read from env via GetEnvironmentDefaults.
type Defaults struct {
	CalendarID string
	ProjectID  string
}

// SetDefaults configures optional identifiers used as fallbacks by methods
// such as AddEvent (when calendarID == "") or AddTask (when projectID == "").
func (c *Client) SetDefaults(d Defaults) { c.defaults = d }

// GetEnvironmentCredentials reads OnlyOffice credentials from environment.
//
// Primary variables (documented):
//   - ONLYOFFICE_URL
//   - ONLYOFFICE_USER
//   - ONLYOFFICE_PASS
//
// Additional aliases accepted for interoperability with sibling tools:
//   - ONLYOFFICE_HOST        (alias for ONLYOFFICE_URL)
//   - ONLYOFFICE_NAME        (alias for ONLYOFFICE_USER)
//   - ONLYOFFICE_PASSWORD    (alias for ONLYOFFICE_PASS)
func GetEnvironmentCredentials() Credentials {
	url := firstNonEmpty(os.Getenv("ONLYOFFICE_URL"), os.Getenv("ONLYOFFICE_HOST"))
	url = strings.TrimRight(url, "/")
	return Credentials{
		Url:      url,
		User:     firstNonEmpty(os.Getenv("ONLYOFFICE_USER"), os.Getenv("ONLYOFFICE_NAME")),
		Password: firstNonEmpty(os.Getenv("ONLYOFFICE_PASS"), os.Getenv("ONLYOFFICE_PASSWORD")),
	}
}

// GetEnvironmentDefaults reads optional library defaults from environment:
//
//   - ONLYOFFICE_CALENDAR_ID (default: "1")
//   - ONLYOFFICE_PROJECT_ID  (alias: ONLYOFFICE_CALENDAR_PROJECT_ID; default: "33")
func GetEnvironmentDefaults() Defaults {
	return Defaults{
		CalendarID: firstNonEmpty(os.Getenv("ONLYOFFICE_CALENDAR_ID"), "1"),
		ProjectID: firstNonEmpty(
			os.Getenv("ONLYOFFICE_PROJECT_ID"),
			os.Getenv("ONLYOFFICE_CALENDAR_PROJECT_ID"),
			"33",
		),
	}
}

// firstNonEmpty returns the first non-empty trimmed value, or "" if none.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
