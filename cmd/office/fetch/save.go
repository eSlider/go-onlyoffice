package fetch

import (
	"context"
	"fmt"
	"strconv"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// SaveItem persists editable form fields for supported entity kinds.
func (l *Loader) SaveItem(ctx context.Context, item model.Item, fields model.FormFields) error {
	if l == nil || l.Client == nil {
		return fmt.Errorf("fetch: client is nil")
	}
	switch item.Kind {
	case model.KindTask:
		return l.UpdateTask(ctx, item.ID, fields)
	case model.KindProject:
		return l.saveProject(ctx, item.ID, fields)
	case model.KindUser:
		return l.SaveUser(ctx, item.ID, fields)
	default:
		return fmt.Errorf("save not supported for %s", item.Kind)
	}
}

func (l *Loader) saveProject(ctx context.Context, projectID string, fields model.FormFields) error {
	id, err := strconv.Atoi(projectID)
	if err != nil {
		return err
	}
	if fields.ResponsibleID == "" {
		raw, derr := l.Detail(ctx, model.Item{ID: projectID, Kind: model.KindProject})
		if derr == nil {
			fields.ResponsibleID = model.ResponsibleIDFromRaw(raw)
		}
	}
	req := onlyoffice.ProjectUpdateRequest{
		ID:          id,
		Title:       fields.Primary,
		Description: fields.Secondary,
	}
	if fields.ResponsibleID != "" {
		req.ResponsibleID = fields.ResponsibleID
	}
	if _, err := l.Client.UpdateProject(req); err != nil {
		return err
	}
	if fields.HasStatus {
		if _, err := l.Client.UpdateProjectStatus(id, string(fields.Status)); err != nil {
			return err
		}
	}
	return nil
}

// DetailForm loads form field values for the detail pane.
func (l *Loader) DetailForm(ctx context.Context, item model.Item) (model.FormFields, error) {
	raw, err := l.Detail(ctx, item)
	if err != nil {
		return model.FormFields{}, err
	}
	fields := model.FormFieldsFromRaw(item.Kind, raw)
	if item.Kind == model.KindTask {
		choices, uerr := l.LoadUserChoices(ctx)
		if uerr != nil {
			return model.FormFields{}, uerr
		}
		fields.UserChoices = choices
	}
	return fields, nil
}
