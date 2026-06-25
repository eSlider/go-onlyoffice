package fetch

import (
	"context"
	"fmt"
	"strconv"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// UpdateTask saves editable task fields via the typed JSON API.
func (l *Loader) UpdateTask(ctx context.Context, taskID string, fields model.FormFields) error {
	return l.updateTask(ctx, taskID, fields, false)
}

// CloseTask saves fields and sets status to closed.
func (l *Loader) CloseTask(ctx context.Context, taskID string, fields model.FormFields) error {
	fields.TaskStatus = model.TaskLifecycleClosed
	return l.updateTask(ctx, taskID, fields, true)
}

func (l *Loader) updateTask(ctx context.Context, taskID string, fields model.FormFields, closing bool) error {
	if l == nil || l.Client == nil {
		return fmt.Errorf("fetch: client is nil")
	}
	id, err := strconv.Atoi(taskID)
	if err != nil {
		return fmt.Errorf("task id %q: %w", taskID, err)
	}
	status := onlyoffice.ProjectTaskStatus(fields.TaskStatus)
	if closing {
		status = onlyoffice.ProjectTaskStatusClosed
	}
	req := onlyoffice.ProjectTaskUpdateRequest{
		ID:          id,
		Title:       fields.Primary,
		Description: fields.Secondary,
		Status:      status,
	}
	if fields.ResponsibleID != "" {
		req.Responsible = []string{fields.ResponsibleID}
	} else if !closing {
		raw, derr := l.Detail(ctx, model.Item{ID: taskID, Kind: model.KindTask})
		if derr == nil {
			if rid := model.TaskResponsibleIDFromRaw(raw); rid != "" {
				req.Responsible = []string{rid}
			}
		}
	}
	_, err = l.Client.UpdateProjectTask(req)
	return err
}
