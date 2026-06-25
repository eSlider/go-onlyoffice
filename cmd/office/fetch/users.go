package fetch

import (
	"context"

	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// LoadUserChoices returns portal users for responsible pickers.
func (l *Loader) LoadUserChoices(ctx context.Context) ([]model.UserOption, error) {
	if l == nil || l.Client == nil {
		return nil, nil
	}
	users, err := l.Client.GetUsers()
	if err != nil {
		return nil, err
	}
	out := make([]model.UserOption, 0, len(users))
	for _, u := range users {
		if u == nil || u.ID == nil || *u.ID == "" {
			continue
		}
		name := ""
		if u.DisplayName != nil {
			name = *u.DisplayName
		}
		if name == "" && u.Email != nil {
			name = *u.Email
		}
		out = append(out, model.UserOption{ID: *u.ID, Name: name})
	}
	return out, nil
}
