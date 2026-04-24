package onlyoffice

// User / People endpoints and associated entity types.

import (
	"context"
	"encoding/json"
	"time"
)

// User represents an OnlyOffice portal user. Fields reflect the full
// /api/2.0/people/filter.json response; most are optional and returned
// only in user-detail responses.
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

// Contact is a typed contact entry attached to a User.
type Contact struct {
	Type  *string `json:"type,omitempty"`
	Value *string `json:"value,omitempty"`
}

// Group is a portal user group.
type Group struct {
	ID      *string `json:"id,omitempty"`
	Name    *string `json:"name,omitempty"`
	Manager any     `json:"manager,omitempty"`
}

// GetUsers lists all portal users.
func (c *Client) GetUsers() (list []*User, err error) {
	return list, c.Query(Request{Uri: "/api/2.0/people/filter.json"},
		&struct {
			MetaResponse `json:",inline"`
			Response     *[]*User `json:"response"`
		}{Response: &list})
}

// SelfUserID returns the ID of the authenticated user (people/@self), cached.
func (c *Client) SelfUserID(ctx context.Context) (string, error) {
	if c.selfID != "" {
		return c.selfID, nil
	}
	raw, err := c.getJSON(ctx, "/api/2.0/people/@self.json")
	if err != nil {
		return "", err
	}
	var env struct {
		Response struct {
			ID string `json:"id"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	c.selfID = env.Response.ID
	return c.selfID, nil
}
