package model

import (
	"fmt"
	"strings"
)

// UserACLDef is one administrator permission row in the user editor.
type UserACLDef struct {
	Key   string
	Label string
}

// UserACLDefs lists portal ACL rows (Full access + per-module grants).
var UserACLDefs = []UserACLDef{
	{Key: "full", Label: "Full access"},
	{Key: "documents", Label: "Documents"},
	{Key: "projects", Label: "Projects"},
	{Key: "crm", Label: "CRM"},
	{Key: "community", Label: "Community"},
	{Key: "people", Label: "People"},
	{Key: "sample", Label: "Sample"},
	{Key: "mail", Label: "Mail"},
}

// UserACLState is the editable ACL snapshot for one user.
type UserACLState struct {
	FullAccess bool
	Modules    map[string]bool
}

// UserACLFromRaw parses API user detail into ACL editor state.
func UserACLFromRaw(raw map[string]any) UserACLState {
	modules := make(map[string]bool, len(UserACLDefs))
	for _, def := range UserACLDefs[1:] {
		modules[def.Key] = false
	}
	if list, ok := raw["listAdminModules"].([]any); ok {
		for _, v := range list {
			key := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
			if key != "" {
				modules[key] = true
			}
		}
	}
	full := boolRaw(raw, "isAdmin")
	if full {
		for k := range modules {
			modules[k] = true
		}
	}
	return UserACLState{FullAccess: full, Modules: modules}
}

// APIPayload converts editor state to OnlyOffice update fields.
func (s UserACLState) APIPayload() (isAdmin bool, modules []string) {
	if s.FullAccess {
		return true, nil
	}
	for _, def := range UserACLDefs[1:] {
		if s.Modules[def.Key] {
			modules = append(modules, def.Key)
		}
	}
	return false, modules
}

// ACLModuleOn reports whether a module row should show as granted.
func (s UserACLState) ACLModuleOn(key string) bool {
	if key == "full" {
		return s.FullAccess
	}
	if s.FullAccess {
		return true
	}
	return s.Modules[key]
}

func boolRaw(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	switch v := m[key].(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	default:
		return false
	}
}
