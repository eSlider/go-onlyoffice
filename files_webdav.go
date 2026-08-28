package onlyoffice

// WebDAV-oriented Files operations. These expose the Documents module through
// value types and cover everything needed to back a filesystem mapping:
// listing (including the virtual @root sections), folder/file CRUD, move/copy,
// and streaming upload/download. They are intentionally small and dependency
// free (only net/http), so callers are not forced to import heavier parts of
// the library.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DavFolder is a folder row from the Files module.
type DavFolder struct {
	ID           string
	Title        string
	ParentID     string
	RootType     int // 1=Common, 3=Trash, 5=My, 6=Share, 8=Projects, ...
	FilesCount   int
	FoldersCount int
	Access       int
	Shared       bool
	Updated      string
}

// DavFile is a file row from the Files module.
type DavFile struct {
	ID      string
	Title   string
	Size    int64
	Updated string
	ViewURL string
}

// DavListing is the contents of one folder.
type DavListing struct {
	Current DavFolder
	Files   []DavFile
	Folders []DavFolder
}

// ListDavFolder returns the contents of a folder by id, which may be a
// symbolic root such as "@my". For "@root" use ListDavSections.
func (c *Client) ListDavFolder(ctx context.Context, id string) (*DavListing, error) {
	raw, err := c.getJSON(ctx, "/api/2.0/files/"+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	// @root returns an array with a single blob; a normal folder returns an
	// object. Normalize both.
	if len(resp) > 0 && resp[0] == '[' {
		var arr []*DavListing
		if err := json.Unmarshal(resp, &arr); err != nil {
			return nil, err
		}
		if len(arr) == 0 {
			return &DavListing{}, nil
		}
		return arr[0], nil
	}
	var l DavListing
	if err := json.Unmarshal(resp, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// ListDavSections returns the virtual top-level sections shown by @root
// ("In projects", "My documents", "Shared with me", "Common", "Favorites",
// "Recent", "Trash"). Each is the `current` folder of one @root element.
func (c *Client) ListDavSections(ctx context.Context) ([]DavFolder, error) {
	raw, err := c.getJSON(ctx, "/api/2.0/files/@root")
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	var arr []struct {
		Current DavFolder `json:"current"`
	}
	if err := json.Unmarshal(resp, &arr); err != nil {
		// Tolerate a non-array (single listing) response.
		var single DavListing
		if err2 := json.Unmarshal(resp, &single); err2 != nil {
			return nil, err
		}
		return []DavFolder{single.Current}, nil
	}
	sections := make([]DavFolder, 0, len(arr))
	for i := range arr {
		sections = append(sections, arr[i].Current)
	}
	return sections, nil
}

// CreateDavFolder creates a folder titled title inside parentID.
func (c *Client) CreateDavFolder(ctx context.Context, parentID, title string) (*DavFolder, error) {
	raw, err := c.postJSON(ctx, "/api/2.0/files/folder/"+url.PathEscape(parentID),
		map[string]string{"title": title})
	if err != nil {
		return nil, err
	}
	var env struct {
		Response *DavFolder `json:"response"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Response == nil {
		return nil, fmt.Errorf("onlyoffice: empty create-folder response")
	}
	return env.Response, nil
}

// RenameDavFolder renames a folder.
func (c *Client) RenameDavFolder(ctx context.Context, id, title string) error {
	_, err := c.putJSON(ctx, "/api/2.0/files/folder/"+url.PathEscape(id),
		map[string]string{"title": title})
	return err
}

// RenameDavFile renames a file (title includes the extension).
func (c *Client) RenameDavFile(ctx context.Context, id, title string) error {
	_, err := c.putJSON(ctx, "/api/2.0/files/file/"+url.PathEscape(id),
		map[string]string{"title": title})
	return err
}

// MoveDavItems moves the given folders and/or files into destFolderID.
func (c *Client) MoveDavItems(ctx context.Context, folderIDs, fileIDs []string, destFolderID string) error {
	_, err := c.putJSON(ctx, "/api/2.0/files/fileops/move", map[string]any{
		"folderIds":    nums(folderIDs),
		"fileIds":      nums(fileIDs),
		"destFolderId": num(destFolderID),
		"resolveType":  "Skip",
		"holdResult":   true,
	})
	return err
}

// CopyDavItems copies the given folders and/or files into destFolderID.
func (c *Client) CopyDavItems(ctx context.Context, folderIDs, fileIDs []string, destFolderID string) error {
	_, err := c.putJSON(ctx, "/api/2.0/files/fileops/copy", map[string]any{
		"folderIds":           nums(folderIDs),
		"fileIds":             nums(fileIDs),
		"destFolderId":        num(destFolderID),
		"conflictResolveType": "Skip",
		"deleteAfter":         true,
	})
	return err
}

// DeleteDavItems deletes the given folders and/or files.
func (c *Client) DeleteDavItems(ctx context.Context, folderIDs, fileIDs []string) error {
	body := map[string]any{"DeleteAfter": true, "Immediately": true}
	for _, id := range folderIDs {
		if _, err := c.deleteJSON(ctx, "/api/2.0/files/folder/"+url.PathEscape(id), body); err != nil {
			return err
		}
	}
	for _, id := range fileIDs {
		if _, err := c.deleteJSON(ctx, "/api/2.0/files/file/"+url.PathEscape(id), body); err != nil {
			return err
		}
	}
	return nil
}

// UploadDavFile uploads src (fileName) into folderID, streaming from src.
func (c *Client) UploadDavFile(ctx context.Context, folderID, fileName string, src io.Reader) (*DavFile, error) {
	raw, err := c.uploadReader(ctx, "/api/2.0/files/"+url.PathEscape(folderID)+"/upload", "file", fileName, src)
	if err != nil {
		return nil, err
	}
	var env struct {
		Response *DavFile `json:"response"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Response == nil {
		return nil, fmt.Errorf("onlyoffice: empty upload response")
	}
	return env.Response, nil
}

// DownloadDavFile streams the file identified by id to w, returning bytes copied.
func (c *Client) DownloadDavFile(ctx context.Context, id string, w io.Writer) (int64, error) {
	file, err := c.GetFile(ctx, id)
	if err != nil {
		return 0, err
	}
	if file.ViewURL == nil || *file.ViewURL == "" {
		return 0, fmt.Errorf("onlyoffice: file %s has no viewUrl", id)
	}
	u := c.resolveAPIURL(*file.ViewURL)
	auth, err := c.authHeader()
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", auth)
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("onlyoffice: download: %d", resp.StatusCode)
	}
	return io.Copy(w, resp.Body)
}

// --- internal helpers -------------------------------------------------------

// deleteJSON performs an authenticated DELETE with an optional JSON body.
func (c *Client) deleteJSON(ctx context.Context, path string, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(buf)
	}
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL()+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DELETE %s: %d %s", path, resp.StatusCode, truncate(string(raw), 400))
	}
	return raw, nil
}

// uploadReader uploads a stream to path under the given form field name.
func (c *Client) uploadReader(ctx context.Context, path, fieldName, fileName string, src io.Reader) (json.RawMessage, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, src); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	auth, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upload %s: %d %s", path, resp.StatusCode, truncate(string(raw), 400))
	}
	return raw, nil
}

func nums(ids []string) []json.Number {
	out := make([]json.Number, 0, len(ids))
	for _, id := range ids {
		if _, err := strconv.Atoi(id); err == nil {
			out = append(out, json.Number(id))
		}
	}
	return out
}

func num(id string) any {
	if _, err := strconv.Atoi(id); err == nil {
		return json.Number(id)
	}
	return id
}

// UnmarshalJSON decodes a folder from the portal envelope, including fields
// that the base FolderEntry omits (parentId, rootFolderType, access, ...).
func (f *DavFolder) UnmarshalJSON(b []byte) error {
	var raw struct {
		ID           *json.Number `json:"id"`
		Title        *string      `json:"title"`
		ParentID     *json.Number `json:"parentId"`
		RootType     *int         `json:"rootFolderType"`
		FilesCount   *int         `json:"filesCount"`
		FoldersCount *int         `json:"foldersCount"`
		Access       *int         `json:"access"`
		Shared       *bool        `json:"shared"`
		Updated      *string      `json:"updated"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw.ID != nil {
		f.ID = raw.ID.String()
	}
	if raw.Title != nil {
		f.Title = *raw.Title
	}
	if raw.ParentID != nil {
		f.ParentID = raw.ParentID.String()
	}
	if raw.RootType != nil {
		f.RootType = *raw.RootType
	}
	if raw.FilesCount != nil {
		f.FilesCount = *raw.FilesCount
	}
	if raw.FoldersCount != nil {
		f.FoldersCount = *raw.FoldersCount
	}
	if raw.Access != nil {
		f.Access = *raw.Access
	}
	if raw.Shared != nil {
		f.Shared = *raw.Shared
	}
	if raw.Updated != nil {
		f.Updated = *raw.Updated
	}
	return nil
}

// UnmarshalJSON decodes a file row, capturing size and timestamps.
func (f *DavFile) UnmarshalJSON(b []byte) error {
	var raw struct {
		ID        *json.Number `json:"id"`
		Title     *string      `json:"title"`
		PureSize  *int64       `json:"pureContentLength"`
		SizeStr   *string      `json:"contentLength"`
		Updated   *string      `json:"updated"`
		ViewURL   *string      `json:"viewUrl"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw.ID != nil {
		f.ID = raw.ID.String()
	}
	if raw.Title != nil {
		f.Title = *raw.Title
	}
	if raw.PureSize != nil {
		f.Size = *raw.PureSize
	} else if raw.SizeStr != nil {
		if n, err := strconv.ParseInt(strings.Fields(*raw.SizeStr)[0], 10, 64); err == nil {
			f.Size = n
		}
	}
	if raw.Updated != nil {
		f.Updated = *raw.Updated
	}
	if raw.ViewURL != nil {
		f.ViewURL = *raw.ViewURL
	}
	return nil
}

// ModTime parses the folder's updated timestamp.
func (f *DavFolder) ModTime() time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05.0000000-07:00", f.Updated)
	return t
}

// ModTime parses the file's updated timestamp.
func (f *DavFile) ModTime() time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05.0000000-07:00", f.Updated)
	return t
}
