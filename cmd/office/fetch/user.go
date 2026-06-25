package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

func (l *Loader) listUsers(ctx context.Context) ([]model.Item, error) {
	users, err := l.Client.GetUsers()
	if err != nil {
		return nil, err
	}
	items := make([]model.Item, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		raw := userToRaw(u)
		id := strMap(raw, "id")
		userName := strMap(raw, "userName")
		if userName == "" {
			userName = strMap(raw, "email")
		}
		items = append(items, model.Item{
			ID:    id,
			Title: userName,
			Kind:  model.KindUser,
			Raw:   raw,
		})
	}
	return items, nil
}

func userToRaw(u *onlyoffice.User) map[string]any {
	b, err := json.Marshal(u)
	if err != nil {
		return map[string]any{}
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return map[string]any{}
	}
	return raw
}

func strMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	if m[key] == nil {
		return ""
	}
	return fmt.Sprint(m[key])
}

func userProfileUpdateBody(isAdmin bool, modules []string) map[string]any {
	body := map[string]any{
		"isAdmin": isAdmin,
	}
	if !isAdmin {
		body["listAdminModules"] = modules
	}
	return body
}

// SaveUser persists user account settings from the detail form.
func (l *Loader) SaveUser(ctx context.Context, userID string, fields model.FormFields) error {
	if l == nil || l.Client == nil {
		return fmt.Errorf("fetch: client is nil")
	}
	raw, err := l.Client.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	wasEnabled := model.UserIsEnabled(raw)

	if fields.UserEnabled && !wasEnabled {
		if err := l.Client.ChangeUserStatus(ctx, userID, true); err != nil {
			return err
		}
	}

	isAdmin, modules := fields.UserACL.APIPayload()
	body := userProfileUpdateBody(isAdmin, modules)
	if _, err := l.updateUserProfile(ctx, userID, body, fields.UserEnabled); err != nil {
		return err
	}

	if !fields.UserEnabled && wasEnabled {
		if err := l.Client.ChangeUserStatus(ctx, userID, false); err != nil {
			return err
		}
	}

	if fields.UserPassword != "" {
		if err := l.Client.ChangeUserPassword(ctx, userID, fields.UserPassword); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loader) updateUserProfile(ctx context.Context, userID string, body map[string]any, wantEnabled bool) (map[string]any, error) {
	out, err := l.Client.UpdateUser(ctx, userID, body)
	if err == nil {
		return out, nil
	}
	if wantEnabled && isSuspendedUserError(err) {
		if actErr := l.Client.ChangeUserStatus(ctx, userID, true); actErr != nil {
			return nil, err
		}
		return l.Client.UpdateUser(ctx, userID, body)
	}
	return nil, err
}

func isSuspendedUserError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "suspended") || strings.Contains(msg, "terminated")
}
