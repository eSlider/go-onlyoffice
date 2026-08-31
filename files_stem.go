package onlyoffice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrFileExists is returned when --no-replace / no-clobber upload hits an existing stem|ext.
var ErrFileExists = errors.New("onlyoffice: file already exists in folder (use replace or delete first)")

// FileEntryStem returns the logical basename without duplicated extensions.
// OO often stores title="foo.docx" and fileExst=".docx" (UI shows foo.docx.docx).
func FileEntryStem(f *FileEntry) string {
	if f == nil || f.Title == nil {
		return ""
	}
	exst := ""
	if f.FileExst != nil {
		exst = *f.FileExst
	}
	return NormalizeUploadStem(*f.Title, exst)
}

// NormalizeUploadStem derives a stable stem for matching uploads.
func NormalizeUploadStem(title, exst string) string {
	t := strings.TrimSpace(title)
	t = strings.TrimSuffix(t, ".")
	if exst != "" && strings.HasSuffix(t, exst) {
		t = strings.TrimSuffix(t, exst)
	}
	if ext := filepath.Ext(t); ext != "" {
		t = strings.TrimSuffix(t, ext)
	}
	return strings.TrimSpace(t)
}

// UploadStemFromLocal returns the stem used to match/replace folder files.
func UploadStemFromLocal(localPath string) string {
	base := filepath.Base(localPath)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}

// FolderFiles returns file entries in a Documents folder.
func (c *Client) FolderFiles(ctx context.Context, folderID string) ([]*FileEntry, error) {
	raw, err := c.ListFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}
	return ParseFolderFileEntries(raw), nil
}

// ParseFolderFileEntries extracts []*FileEntry from ListFolder JSON.
func ParseFolderFileEntries(raw map[string]any) []*FileEntry {
	items, _ := raw["files"].([]any)
	out := make([]*FileEntry, 0, len(items))
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		b, err := json.Marshal(m)
		if err != nil {
			continue
		}
		var f FileEntry
		if err := json.Unmarshal(b, &f); err != nil {
			continue
		}
		out = append(out, &f)
	}
	return out
}

// FindFilesByStem returns folder files whose logical stem matches.
func FindFilesByStem(files []*FileEntry, stem string) []*FileEntry {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return nil
	}
	var out []*FileEntry
	for _, f := range files {
		if FileEntryStem(f) == stem {
			out = append(out, f)
		}
	}
	return out
}

// DeleteFilesByStem removes all files in folderID matching stem (any extension).
// Prefer DeleteFilesByDedupKey when the upload extension is known.
func (c *Client) DeleteFilesByStem(ctx context.Context, folderID, stem string) ([]int, error) {
	files, err := c.FolderFiles(ctx, folderID)
	if err != nil {
		return nil, err
	}
	matches := FindFilesByStem(files, stem)
	if len(matches) == 0 {
		return nil, nil
	}
	ids := make([]int, 0, len(matches))
	for _, f := range matches {
		n := int(FileEntryNumericID(f))
		if n != 0 {
			ids = append(ids, n)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if err := c.DeleteFiles(ctx, ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// AssertNoFileConflict reports ErrFileExists when localPath stem|ext is already in folderID.
func (c *Client) AssertNoFileConflict(ctx context.Context, folderID, localPath string) error {
	files, err := c.FolderFiles(ctx, folderID)
	if err != nil {
		return err
	}
	stem := UploadStemFromLocal(localPath)
	ext := UploadExtFromLocal(localPath)
	matches := FindFilesByDedupKey(files, stem, ext)
	if len(matches) == 0 {
		return nil
	}
	ids := make([]string, 0, len(matches))
	for _, f := range matches {
		ids = append(ids, fmt.Sprintf("%d", FileEntryNumericID(f)))
	}
	return fmt.Errorf("%w: %s%s in folder %s (existing file ids: %s)",
		ErrFileExists, stem, ext, folderID, strings.Join(ids, ", "))
}

// UploadProjectFileNoClobber uploads only when stem|ext is not already in the project folder.
func (c *Client) UploadProjectFileNoClobber(ctx context.Context, projectID, localPath string) (*FileEntry, error) {
	folderID, err := c.projectFolderID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := c.AssertNoFileConflict(ctx, folderID, localPath); err != nil {
		return nil, err
	}
	return c.UploadProjectFile(ctx, projectID, localPath)
}

// UploadToFolderReplacing deletes same stem+ext files then uploads localPath.
func (c *Client) UploadToFolderReplacing(ctx context.Context, folderID, localPath string) (*FileEntry, []int, error) {
	stem := UploadStemFromLocal(localPath)
	ext := UploadExtFromLocal(localPath)
	deleted, err := c.DeleteFilesByDedupKey(ctx, folderID, stem, ext)
	if err != nil {
		return nil, deleted, err
	}
	ent, err := c.UploadToFolder(ctx, folderID, localPath)
	return ent, deleted, err
}
