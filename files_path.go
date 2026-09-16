package onlyoffice

// Human-readable folder paths for search results (F9). The OnlyOffice ES
// index stores only ancestor folder ids; titles live in the Documents tree, so
// resolving a path costs one GET /api/2.0/files/{id} per distinct folder,
// cached on the client. Folders that cannot be listed (e.g. a section root)
// fall back to their id, so a path is always produced.

import (
	"context"
	"strings"
)

// FolderTitle returns the title of a Documents folder id, cached on the client.
// An empty id yields an empty title. Unknown/unlistable ids (section roots)
// return ("", nil) so callers can fall back to the id.
func (c *Client) FolderTitle(ctx context.Context, folderID string) (string, error) {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return "", nil
	}
	c.folderTitlesMu.Lock()
	if c.folderTitles != nil {
		if t, ok := c.folderTitles[folderID]; ok {
			c.folderTitlesMu.Unlock()
			return t, nil
		}
	}
	c.folderTitlesMu.Unlock()

	title := ""
	out, err := c.ListFolder(ctx, folderID)
	if err == nil {
		if cur, ok := out["current"].(map[string]any); ok {
			if s, ok := cur["title"].(string); ok {
				title = strings.TrimSpace(s)
			}
		}
	}

	c.folderTitlesMu.Lock()
	if c.folderTitles == nil {
		c.folderTitles = map[string]string{}
	}
	c.folderTitles[folderID] = title
	c.folderTitlesMu.Unlock()
	return title, nil
}

// FolderPath resolves an ancestor folder id chain (root → leaf, as the ES
// backend reports it) into folder titles, falling back to the id when a title
// cannot be read. The result never fails on a single lookup: only the whole
// call honours ctx cancellation.
func (c *Client) FolderPath(ctx context.Context, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			break
		}
		title, err := c.FolderTitle(ctx, id)
		if err != nil || title == "" {
			title = id
		}
		out = append(out, title)
	}
	return out
}

// UniquePath builds a stable, human-readable, unique path for a result: the
// resolved folder chain plus the file title. "." separates nothing — the
// segments are joined with "/", matching the Documents breadcrumb the web UI
// shows.
func (c *Client) UniquePath(ctx context.Context, folderPath []string, title string) string {
	parts := c.FolderPath(ctx, folderPath)
	if t := strings.TrimSpace(title); t != "" {
		parts = append(parts, t)
	}
	return strings.Join(parts, "/")
}
