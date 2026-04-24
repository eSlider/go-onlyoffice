package onlyoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// FileEntry is a file row from the OnlyOffice Files module or project/task
// file listings (field names follow the Workspace API JSON).
type FileEntry struct {
	ID            *json.Number `json:"id,omitempty"`
	Title         *string      `json:"title,omitempty"`
	FileExst      *string      `json:"fileExst,omitempty"`
	ContentLength *string      `json:"contentLength,omitempty"`
	FileType      *int         `json:"fileType,omitempty"`
	ViewURL       *string      `json:"viewUrl,omitempty"`
	WebURL        *string      `json:"webUrl,omitempty"`
	FolderID      *json.Number `json:"folderId,omitempty"`
	Updated       *time.Time   `json:"updated,omitempty"`
	CreatedBy     *User        `json:"createdBy,omitempty"`
}

// FolderEntry is a folder row from project files listing.
type FolderEntry struct {
	ID           *json.Number `json:"id,omitempty"`
	Title        *string      `json:"title,omitempty"`
	FilesCount   *int         `json:"filesCount,omitempty"`
	FoldersCount *int         `json:"foldersCount,omitempty"`
}

// ProjectFilesResponse is the "response" object from GET
// /api/2.0/project/{id}/files — files and folders attached to the project.
type ProjectFilesResponse struct {
	Folders []*FolderEntry `json:"folders"`
	Files   []*FileEntry   `json:"files"`
}

// UploadOpportunityFile uploads a single file to a CRM opportunity.
// Returns the decoded "response" object from OnlyOffice.
func (c *Client) UploadOpportunityFile(ctx context.Context, opportunityID, filePath string) (map[string]any, error) {
	p := fmt.Sprintf("/api/2.0/crm/opportunity/%s/files/upload.json", url.PathEscape(opportunityID))
	raw, err := c.uploadMultipart(ctx, p, "file", filePath)
	if err != nil {
		return nil, err
	}
	return unmarshalResponseObject(raw)
}

// GetProjectFiles returns files and folders linked to the project.
func (c *Client) GetProjectFiles(ctx context.Context, projectID string) (*ProjectFilesResponse, error) {
	if projectID == "" {
		projectID = c.defaults.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("project id is required")
	}
	p := fmt.Sprintf("/api/2.0/project/%s/files.json", url.PathEscape(projectID))
	raw, err := c.getJSON(ctx, p)
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 || string(resp) == "null" {
		return &ProjectFilesResponse{}, nil
	}
	var out ProjectFilesResponse
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTaskFiles returns files attached to a project task.
func (c *Client) GetTaskFiles(ctx context.Context, taskID string) ([]*FileEntry, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	p := fmt.Sprintf("/api/2.0/project/task/%s/files.json", url.PathEscape(taskID))
	raw, err := c.getJSON(ctx, p)
	if err != nil {
		return nil, err
	}
	resp, err := responseField(raw, "response")
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 || string(resp) == "null" {
		return nil, nil
	}
	var list []*FileEntry
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// UploadTaskFile uploads a file into the task's project Documents folder, then
// attaches the new file id to the task. OnlyOffice POST .../task/{id}/files
// expects existing file IDs, not a multipart body.
func (c *Client) UploadTaskFile(ctx context.Context, taskID, localPath string) (*FileEntry, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	task, err := c.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	pid := projectIDFromTaskMap(task)
	if pid == "" {
		return nil, fmt.Errorf("task %s: cannot resolve project id for upload", taskID)
	}
	entry, err := c.UploadProjectFile(ctx, pid, localPath)
	if err != nil {
		return nil, err
	}
	nid := int(FileEntryNumericID(entry))
	if nid == 0 {
		return nil, fmt.Errorf("upload returned no file id")
	}
	if err := c.AttachFilesToTask(ctx, taskID, nid); err != nil {
		return nil, err
	}
	return entry, nil
}

// AttachFilesToTask links existing Documents-module files to a task.
func (c *Client) AttachFilesToTask(ctx context.Context, taskID string, fileIDs ...int) error {
	if taskID == "" || len(fileIDs) == 0 {
		return fmt.Errorf("task id and at least one file id are required")
	}
	v := url.Values{}
	for _, id := range fileIDs {
		v.Add("files", strconv.Itoa(id))
	}
	p := fmt.Sprintf("/api/2.0/project/task/%s/files.json", url.PathEscape(taskID))
	if _, err := c.postForm(ctx, p, v); err != nil {
		p2 := fmt.Sprintf("/api/2.0/project/task/%s/files", url.PathEscape(taskID))
		if _, err2 := c.postForm(ctx, p2, v); err2 != nil {
			return fmt.Errorf("attach files to task: %w (retry: %v)", err, err2)
		}
	}
	return nil
}

func projectIDFromTaskMap(m map[string]any) string {
	if m == nil {
		return ""
	}
	if po, ok := m["projectOwner"].(map[string]any); ok {
		if id, ok := po["id"]; ok {
			switch x := id.(type) {
			case float64:
				return strconv.FormatInt(int64(x), 10)
			case int:
				return strconv.Itoa(x)
			case string:
				return x
			}
		}
	}
	return ""
}

// DetachTaskFile removes a file attachment from the task (file remains in Documents).
func (c *Client) DetachTaskFile(ctx context.Context, taskID, fileID string) error {
	if taskID == "" || fileID == "" {
		return fmt.Errorf("task id and file id are required")
	}
	q := url.Values{}
	q.Set("fileid", fileID)
	p := fmt.Sprintf("/api/2.0/project/task/%s/files.json?%s", url.PathEscape(taskID), q.Encode())
	if _, err := c.deleteReq(ctx, p); err != nil {
		p2 := fmt.Sprintf("/api/2.0/project/task/%s/files?%s", url.PathEscape(taskID), q.Encode())
		if _, err2 := c.deleteReq(ctx, p2); err2 != nil {
			return fmt.Errorf("detach task file: %w (retry: %v)", err, err2)
		}
	}
	return nil
}

// projectFolderID resolves the Documents folder id for project file uploads.
func (c *Client) projectFolderID(ctx context.Context, projectID string) (string, error) {
	m, err := c.GetProjectByID(ctx, projectID)
	if err != nil {
		return "", err
	}
	if v, ok := m["projectFolder"]; ok && v != nil {
		switch x := v.(type) {
		case float64:
			return strconv.FormatInt(int64(x), 10), nil
		case json.Number:
			return x.String(), nil
		case string:
			if x != "" {
				return x, nil
			}
		}
	}
	// Fallback: first folder from project files listing.
	pf, err := c.GetProjectFiles(ctx, projectID)
	if err != nil {
		return "", err
	}
	if len(pf.Folders) > 0 && pf.Folders[0].ID != nil {
		return pf.Folders[0].ID.String(), nil
	}
	return "", fmt.Errorf("project %s has no projectFolder and no folders in files listing", projectID)
}

// UploadProjectFile uploads a file into the project's Documents folder.
func (c *Client) UploadProjectFile(ctx context.Context, projectID, localPath string) (*FileEntry, error) {
	folderID, err := c.projectFolderID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	// Workspace DocumentsApi.UploadFile: POST .../{folderId}/upload (multipart or raw stream).
	uploadPath := fmt.Sprintf("/api/2.0/files/%s/upload.json", url.PathEscape(folderID))
	raw, err := c.uploadMultipart(ctx, uploadPath, "file", localPath)
	if err != nil {
		uploadPath = fmt.Sprintf("/api/2.0/files/%s/upload", url.PathEscape(folderID))
		raw, err = c.uploadMultipart(ctx, uploadPath, "file", localPath)
		if err != nil {
			return nil, err
		}
	}
	return decodeResponseFileEntry(raw)
}

// GetFile returns file metadata including viewUrl for download.
func (c *Client) GetFile(ctx context.Context, fileID string) (*FileEntry, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file id is required")
	}
	p := fmt.Sprintf("/api/2.0/files/file/%s.json", url.PathEscape(fileID))
	raw, err := c.getJSON(ctx, p)
	if err != nil {
		return nil, err
	}
	return decodeResponseFileEntry(raw)
}

// RenameFile sets a new title (including extension) for the file.
func (c *Client) RenameFile(ctx context.Context, fileID, newTitle string) (*FileEntry, error) {
	if fileID == "" || newTitle == "" {
		return nil, fmt.Errorf("file id and new title are required")
	}
	p := fmt.Sprintf("/api/2.0/files/file/%s.json", url.PathEscape(fileID))
	raw, err := c.putJSON(ctx, p, map[string]string{"title": newTitle})
	if err != nil {
		return nil, err
	}
	return decodeResponseFileEntry(raw)
}

type deleteFilesBody struct {
	FileIDs   []int `json:"fileIds"`
	FolderIDs []int `json:"folderIds"`
}

// DeleteFiles permanently deletes files by numeric id (Documents module).
func (c *Client) DeleteFiles(ctx context.Context, fileIDs []int) error {
	if len(fileIDs) == 0 {
		return fmt.Errorf("no file ids to delete")
	}
	body := deleteFilesBody{FileIDs: fileIDs, FolderIDs: nil}
	_, err := c.putJSON(ctx, "/api/2.0/files/fileops/delete.json", body)
	if err != nil {
		_, err = c.putJSON(ctx, "/api/2.0/files/fileops/delete", body)
	}
	return err
}

// DownloadFile streams file bytes from the file's viewUrl using the same auth
// as API calls. Writes into dst.
func (c *Client) DownloadFile(ctx context.Context, fileID string, dst io.Writer) (int64, error) {
	f, err := c.GetFile(ctx, fileID)
	if err != nil {
		return 0, err
	}
	if f.ViewURL == nil || *f.ViewURL == "" {
		return 0, fmt.Errorf("file %s has no viewUrl", fileID)
	}
	downloadURL := c.resolveAPIURL(*f.ViewURL)
	auth, err := c.authHeader()
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
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
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("GET viewUrl: %d %s", resp.StatusCode, truncate(string(b), 400))
	}
	n, err := io.Copy(dst, resp.Body)
	return n, err
}

func (c *Client) resolveAPIURL(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ref
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	base := c.baseURL()
	if strings.HasPrefix(ref, "/") {
		u, err := url.Parse(base)
		if err != nil {
			return base + ref
		}
		u.Path = ""
		u.RawQuery = ""
		u.Fragment = ""
		return strings.TrimRight(u.String(), "/") + ref
	}
	return base + "/" + strings.TrimPrefix(ref, "/")
}

func decodeResponseFileEntry(raw json.RawMessage) (*FileEntry, error) {
	var env struct {
		Response *FileEntry `json:"response"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Response == nil {
		return nil, fmt.Errorf("empty file response")
	}
	return env.Response, nil
}

// FileEntryNumericID returns the file id as int64, or 0 if missing/invalid.
func FileEntryNumericID(f *FileEntry) int64 {
	if f == nil || f.ID == nil {
		return 0
	}
	n, err := f.ID.Int64()
	if err != nil {
		return 0
	}
	return n
}

// FileEntryTitle returns the title or empty string.
func FileEntryTitle(f *FileEntry) string {
	if f == nil || f.Title == nil {
		return ""
	}
	return *f.Title
}

// SafeLocalFileName sanitizes a server title for use as a local filename.
func SafeLocalFileName(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "download"
	}
	base := path.Base(title)
	base = strings.ReplaceAll(base, "\x00", "")
	if base == "." || base == "/" {
		return "download"
	}
	return base
}
