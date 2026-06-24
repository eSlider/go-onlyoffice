package fetch

import (
	"context"
	"fmt"
	"strconv"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
)

// UpdateTask saves title and description for a project task.
func (l *Loader) UpdateTask(ctx context.Context, taskID, title, description string) error {
	if l == nil || l.Client == nil {
		return fmt.Errorf("fetch: client is nil")
	}
	id, err := strconv.Atoi(taskID)
	if err != nil {
		return fmt.Errorf("task id %q: %w", taskID, err)
	}
	_, err = l.Client.UpdateProjectTask(onlyoffice.ProjectTaskUpdateRequest{
		ID:          id,
		Title:       title,
		Description: description,
	})
	return err
}

// TaskFields loads title and description for a project task item.
func (l *Loader) TaskFields(ctx context.Context, item model.Item) (title, description string, err error) {
	fields, err := l.DetailForm(ctx, item)
	if err != nil {
		return "", "", err
	}
	return fields.Primary, fields.Secondary, nil
}
