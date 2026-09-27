package onlyoffice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
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

// AssertNoFileConflict reports ErrFileExists when localPath logical name is
// already in folderID. Matching is conversion-aware, so a .xls upload also
// conflicts with an existing .xlsx and does not create a hidden duplicate.
func (c *Client) AssertNoFileConflict(ctx context.Context, folderID, localPath string) error {
	files, err := c.FolderFiles(ctx, folderID)
	if err != nil {
		return err
	}
	stem := UploadStemFromLocal(localPath)
	ext := UploadExtFromLocal(localPath)
	matches := FindFilesByStemExt(files, stem, ext)
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

// uploadReplacementPlan is how a replacing upload reconciles with a folder.
type uploadReplacementPlan struct {
	// UpdateID is the existing file to overwrite in place; set only when the
	// stored extension equals the local one. UpdateFile keeps the stored name,
	// so overwriting across extensions would leave an unconverted body under a
	// mismatched name.
	UpdateID string
	// DeleteIDs are the ids to remove before uploading. They are the redundant
	// duplicates of an in-place update, or every conversion-equivalent
	// counterpart when the body must be converted again by a fresh upload.
	DeleteIDs []int
}

// planUploadReplacement matches an incoming local upload (stem + ext) against
// the files already in a folder and decides between a fresh upload, an in-place
// update and delete + reupload. Matching tolerates the legacy→OOXML conversion
// OnlyOffice performs on upload, so a repeated upload of f.xls finds the saved
// f.xlsx instead of creating a second file. Because UpdateFile replaces the body
// without re-running that conversion, an equivalent-but-different extension is
// deleted and re-uploaded rather than updated in place (#84).
func planUploadReplacement(files []*FileEntry, stem, ext string) uploadReplacementPlan {
	matches := FindFilesByStemExt(files, stem, ext)
	if len(matches) == 0 {
		return uploadReplacementPlan{}
	}
	keep, remove := pickDuplicateKeeper(matches, false)
	plan := uploadReplacementPlan{}
	if FileEntryExt(keep) == normalizeExt(ext) {
		if id := FileEntryNumericID(keep); id != 0 {
			plan.UpdateID = strconv.FormatInt(id, 10)
		}
	} else if id := int(FileEntryNumericID(keep)); id != 0 {
		plan.DeleteIDs = append(plan.DeleteIDs, id)
	}
	for _, f := range remove {
		if id := int(FileEntryNumericID(f)); id != 0 {
			plan.DeleteIDs = append(plan.DeleteIDs, id)
		}
	}
	return plan
}

// UploadToFolderReplacing upserts localPath into folderID by logical name.
// Matching is stem + extension with OnlyOffice's server-side conversion
// accounted for: a local .xls is stored as .xlsx, so a repeated upload replaces
// the saved document instead of appending a duplicate. An exact-extension
// counterpart is updated in place (stable id, no window without the file); a
// conversion-equivalent one is deleted and uploaded afresh so the server
// converts the body again. Extra duplicates are collapsed either way.
func (c *Client) UploadToFolderReplacing(ctx context.Context, folderID, localPath string) (*FileEntry, []int, error) {
	stem := UploadStemFromLocal(localPath)
	ext := UploadExtFromLocal(localPath)
	files, err := c.FolderFiles(ctx, folderID)
	if err != nil {
		return nil, nil, err
	}
	plan := planUploadReplacement(files, stem, ext)

	if plan.UpdateID != "" {
		ent, err := c.UpdateFile(ctx, plan.UpdateID, localPath)
		if err == nil {
			deleted, derr := c.deleteReplacementStale(ctx, plan.DeleteIDs)
			return ent, deleted, derr
		}
		// Portal rejected the in-place update: delete + fresh upload still
		// leaves one file (server-converted).
		deleted, derr := c.DeleteFilesByStemExt(ctx, folderID, stem, ext)
		if derr != nil {
			return nil, deleted, derr
		}
		ent, uerr := c.UploadToFolder(ctx, folderID, localPath)
		return ent, deleted, uerr
	}

	// Conversion-equivalent (or no counterpart): remove the stale files first so
	// the fresh upload is converted and exactly one file remains.
	deleted, derr := c.deleteReplacementStale(ctx, plan.DeleteIDs)
	if derr != nil {
		return nil, deleted, derr
	}
	ent, uerr := c.UploadToFolder(ctx, folderID, localPath)
	return ent, deleted, uerr
}

// deleteReplacementStale removes the ids collected by planUploadReplacement:
// duplicate files after an in-place update, or every stale counterpart before a
// converted reupload.
func (c *Client) deleteReplacementStale(ctx context.Context, ids []int) ([]int, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if err := c.DeleteFiles(ctx, ids); err != nil {
		return ids, err
	}
	return ids, nil
}
