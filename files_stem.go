package onlyoffice

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
)

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

// DeleteFilesByStem removes all files in folderID matching stem (for put-md upsert).
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

// UploadToFolderReplacing deletes same-stem files then uploads localPath.
func (c *Client) UploadToFolderReplacing(ctx context.Context, folderID, localPath string) (*FileEntry, []int, error) {
	stem := UploadStemFromLocal(localPath)
	deleted, err := c.DeleteFilesByStem(ctx, folderID, stem)
	if err != nil {
		return nil, deleted, err
	}
	ent, err := c.UploadToFolder(ctx, folderID, localPath)
	return ent, deleted, err
}
