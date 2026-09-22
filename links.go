package onlyoffice

// Deep links to OnlyOffice portal objects.

import (
	"fmt"
	"net/url"
	"strings"
)

// FileEditorURL builds the OnlyOffice DocEditor deep link for a portal file id:
//
//	https://<portal>/Products/Files/DocEditor.aspx?fileid=<id>
//
// portalBase may include a trailing slash; fileID is trimmed and URL-escaped.
func FileEditorURL(portalBase, fileID string) string {
	base := strings.TrimRight(strings.TrimSpace(portalBase), "/")
	return fmt.Sprintf("%s/Products/Files/DocEditor.aspx?fileid=%s",
		base, url.QueryEscape(strings.TrimSpace(fileID)))
}

// FileEditorURL is the client-bound convenience wrapper (uses the portal URL
// the client was built with).
func (c *Client) FileEditorURL(fileID string) string {
	return FileEditorURL(c.baseURL(), fileID)
}

// FolderURL builds a Documents-folder deep link for a folder id.
func FolderURL(portalBase, folderID string) string {
	base := strings.TrimRight(strings.TrimSpace(portalBase), "/")
	return fmt.Sprintf("%s/Products/Files/Default.aspx#folder=%s", base, url.QueryEscape(strings.TrimSpace(folderID)))
}

// FolderURL is the client-bound convenience wrapper.
func (c *Client) FolderURL(folderID string) string {
	return FolderURL(c.baseURL(), folderID)
}
