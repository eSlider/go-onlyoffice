package onlyoffice

// Deep links to OnlyOffice portal objects.
//
// Where this is used: third-party-facing document packs (e.g. the internal
// office.example.com "example-city apartment" project) embed per-file links of the
// form /Products/Files/DocEditor.aspx?fileid=<id> in a cover document and in
// chat messages. Those links must be generated consistently so they match the
// file ids returned by `oo projects files list` / `oo link`.
//
// Caveat proven on the internal portal: file ids are server-assigned and a
// re-upload/delete yields a NEW id, so an already-shared link can go stale.
// Use `oo projects files replace-in` (keep the id clean) and, if a legacy link
// must keep working, an nginx alias can 302 the old fileid to the new one.

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
