package fetch

import (
	"bytes"
	"context"
	"fmt"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/office/model"
	"github.com/eslider/go-onlyoffice/cmd/office/preview"
)

// PreviewMarkdown loads item data and returns markdown for the preview pane.
func (l *Loader) PreviewMarkdown(ctx context.Context, item model.Item) (string, error) {
	if l == nil || l.Client == nil {
		return "", fmt.Errorf("fetch: client is nil")
	}
	if item.Kind == model.KindFile {
		return l.filePreviewMarkdown(ctx, item)
	}
	raw, err := l.Detail(ctx, item)
	if err != nil {
		return "", err
	}
	return preview.EntityMarkdown(string(item.Kind), raw), nil
}

func (l *Loader) filePreviewMarkdown(ctx context.Context, item model.Item) (string, error) {
	if item.ID == "" {
		return "", fmt.Errorf("file id missing")
	}
	name := item.Title
	if meta, err := l.Client.GetFile(ctx, item.ID); err == nil && meta != nil {
		if t := onlyoffice.FileEntryTitle(meta); t != "" {
			name = t
		}
	}
	var buf bytes.Buffer
	if _, err := l.Client.DownloadFile(ctx, item.ID, &buf); err != nil {
		return "", err
	}
	return preview.FileBytesToMarkdown(name, buf.Bytes())
}
