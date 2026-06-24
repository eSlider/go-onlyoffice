package fetch

import (
	"context"
	"fmt"
	"strconv"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// SaveItem persists editable form fields for supported entity kinds.
func (l *Loader) SaveItem(ctx context.Context, item model.Item, title, description string) error {
	if l == nil || l.Client == nil {
		return fmt.Errorf("fetch: client is nil")
	}
	switch item.Kind {
	case model.KindTask:
		return l.UpdateTask(ctx, item.ID, title, description)
	case model.KindProject:
		id, err := strconv.Atoi(item.ID)
		if err != nil {
			return err
		}
		_, err = l.Client.UpdateProject(onlyoffice.ProjectUpdateRequest{
			ID:          id,
			Title:       title,
			Description: description,
		})
		return err
	default:
		return fmt.Errorf("save not supported for %s", item.Kind)
	}
}

// DetailForm loads form field values for the detail pane.
func (l *Loader) DetailForm(ctx context.Context, item model.Item) (model.FormFields, error) {
	raw, err := l.Detail(ctx, item)
	if err != nil {
		return model.FormFields{}, err
	}
	return model.FormFieldsFromRaw(item.Kind, raw), nil
}
