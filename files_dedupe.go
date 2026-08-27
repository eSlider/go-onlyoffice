package onlyoffice

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
)

// FileEntryExt returns a normalized extension (lowercase, with leading dot).
func FileEntryExt(f *FileEntry) string {
	if f == nil {
		return ""
	}
	exst := ""
	if f.FileExst != nil {
		exst = strings.TrimSpace(*f.FileExst)
	}
	if exst != "" {
		if !strings.HasPrefix(exst, ".") {
			exst = "." + exst
		}
		return strings.ToLower(exst)
	}
	if f.Title != nil {
		if ext := filepath.Ext(*f.Title); ext != "" {
			return strings.ToLower(ext)
		}
	}
	return ""
}

// FileDedupKey is stem|ext — two files with the same key are duplicates.
func FileDedupKey(f *FileEntry) string {
	st := FileEntryStem(f)
	ext := FileEntryExt(f)
	if st == "" {
		return ""
	}
	if ext == "" {
		return st
	}
	return st + "|" + strings.TrimPrefix(ext, ".")
}

// FindFilesByDedupKey returns folder files matching stem and extension.
func FindFilesByDedupKey(files []*FileEntry, stem, ext string) []*FileEntry {
	key := dedupKeyFromParts(stem, ext)
	if key == "" {
		return nil
	}
	var out []*FileEntry
	for _, f := range files {
		if FileDedupKey(f) == key {
			out = append(out, f)
		}
	}
	return out
}

func dedupKeyFromParts(stem, ext string) string {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return ""
	}
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if ext == "" {
		return stem
	}
	return stem + "|" + strings.TrimPrefix(ext, ".")
}

// UploadExtFromLocal returns the lowercase extension from a local path.
func UploadExtFromLocal(localPath string) string {
	ext := filepath.Ext(localPath)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext)
}

// IsTrashFolderTitle reports staging/trash folders (e.g. _trash-md).
func IsTrashFolderTitle(title string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	return strings.HasPrefix(t, "_") || strings.Contains(t, "trash")
}

// ProjectFolderFile ties a file to its project Documents subfolder.
type ProjectFolderFile struct {
	FolderID    string
	FolderTitle string
	File        *FileEntry
}

// DedupGroup is one duplicate set: keep the newest (or non-trash) file.
type DedupGroup struct {
	Key         string
	FolderID    string
	FolderTitle string
	Keep        *FileEntry
	Remove      []*FileEntry
}

// DedupOptions controls project-wide duplicate scans.
type DedupOptions struct {
	CrossFolder bool
}

// FindProjectDuplicates scans project folders for duplicate files.
func FindProjectDuplicates(folders []*FolderEntry, filesByFolder map[string][]*FileEntry, opts DedupOptions) []DedupGroup {
	var indexed []ProjectFolderFile
	for _, folder := range folders {
		if folder == nil || folder.ID == nil {
			continue
		}
		fid := folder.ID.String()
		title := ""
		if folder.Title != nil {
			title = *folder.Title
		}
		for _, f := range filesByFolder[fid] {
			if f == nil {
				continue
			}
			indexed = append(indexed, ProjectFolderFile{
				FolderID: fid, FolderTitle: title, File: f,
			})
		}
	}
	if opts.CrossFolder {
		return findCrossFolderDuplicates(indexed)
	}
	return findWithinFolderDuplicates(indexed)
}

func findWithinFolderDuplicates(indexed []ProjectFolderFile) []DedupGroup {
	byFolder := map[string][]ProjectFolderFile{}
	for _, it := range indexed {
		byFolder[it.FolderID] = append(byFolder[it.FolderID], it)
	}
	var out []DedupGroup
	for fid, items := range byFolder {
		title := ""
		if len(items) > 0 {
			title = items[0].FolderTitle
		}
		byKey := map[string][]*FileEntry{}
		for _, it := range items {
			k := FileDedupKey(it.File)
			byKey[k] = append(byKey[k], it.File)
		}
		for k, group := range byKey {
			if len(group) < 2 {
				continue
			}
			keep, remove := pickDuplicateKeeper(group, false)
			if keep == nil || len(remove) == 0 {
				continue
			}
			out = append(out, DedupGroup{
				Key: k, FolderID: fid, FolderTitle: title, Keep: keep, Remove: remove,
			})
		}
	}
	sortDedupGroups(out)
	return out
}

func findCrossFolderDuplicates(indexed []ProjectFolderFile) []DedupGroup {
	byKey := map[string][]ProjectFolderFile{}
	for _, it := range indexed {
		k := FileDedupKey(it.File)
		byKey[k] = append(byKey[k], it)
	}
	var out []DedupGroup
	for k, items := range byKey {
		if len(items) < 2 {
			continue
		}
		files := make([]*FileEntry, len(items))
		folders := make([]string, len(items))
		folderTitles := make([]string, len(items))
		for i, it := range items {
			files[i] = it.File
			folders[i] = it.FolderID
			folderTitles[i] = it.FolderTitle
		}
		keep, remove := pickDuplicateKeeperWithFolders(files, folders, folderTitles)
		if keep == nil || len(remove) == 0 {
			continue
		}
		fid, ftitle := "", ""
		for _, it := range items {
			if it.File == keep {
				fid, ftitle = it.FolderID, it.FolderTitle
				break
			}
		}
		out = append(out, DedupGroup{
			Key: k, FolderID: fid, FolderTitle: ftitle, Keep: keep, Remove: remove,
		})
	}
	sortDedupGroups(out)
	return out
}

func pickDuplicateKeeper(files []*FileEntry, _ bool) (*FileEntry, []*FileEntry) {
	return pickDuplicateKeeperWithFolders(files, nil, nil)
}

func pickDuplicateKeeperWithFolders(files []*FileEntry, folderIDs, folderTitles []string) (*FileEntry, []*FileEntry) {
	if len(files) == 0 {
		return nil, nil
	}
	type ranked struct {
		file  *FileEntry
		trash bool
	}
	rankedFiles := make([]ranked, len(files))
	for i, f := range files {
		trash := false
		if folderTitles != nil && i < len(folderTitles) {
			trash = IsTrashFolderTitle(folderTitles[i])
		}
		rankedFiles[i] = ranked{file: f, trash: trash}
	}
	sort.SliceStable(rankedFiles, func(i, j int) bool {
		ri, rj := rankedFiles[i], rankedFiles[j]
		if ri.trash != rj.trash {
			return !ri.trash // non-trash first
		}
		ti, tj := rankedFiles[i].file.Updated, rankedFiles[j].file.Updated
		if ti == nil {
			return false
		}
		if tj == nil {
			return true
		}
		return ti.After(*tj) // newest first
	})
	keep := rankedFiles[0].file
	var remove []*FileEntry
	for _, r := range rankedFiles[1:] {
		remove = append(remove, r.file)
	}
	return keep, remove
}

func sortDedupGroups(groups []DedupGroup) {
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].FolderTitle != groups[j].FolderTitle {
			return groups[i].FolderTitle < groups[j].FolderTitle
		}
		return groups[i].Key < groups[j].Key
	})
}

// ApplyDedupGroups deletes Remove files from each group.
func (c *Client) ApplyDedupGroups(ctx context.Context, groups []DedupGroup) ([]int, error) {
	seen := map[int]struct{}{}
	var ids []int
	for _, g := range groups {
		for _, f := range g.Remove {
			n := int(FileEntryNumericID(f))
			if n == 0 {
				continue
			}
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			ids = append(ids, n)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if err := c.DeleteFiles(ctx, ids); err != nil {
		return ids, err
	}
	return ids, nil
}

// DeleteFilesByDedupKey removes all files in folderID matching stem+ext.
func (c *Client) DeleteFilesByDedupKey(ctx context.Context, folderID, stem, ext string) ([]int, error) {
	files, err := c.FolderFiles(ctx, folderID)
	if err != nil {
		return nil, err
	}
	matches := FindFilesByDedupKey(files, stem, ext)
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

// DedupeProject scans project folders and optionally deletes duplicates.
func (c *Client) DedupeProject(ctx context.Context, projectID string, opts DedupOptions, apply bool) ([]DedupGroup, []int, error) {
	pf, err := c.GetProjectFiles(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	filesByFolder := make(map[string][]*FileEntry, len(pf.Folders))
	for _, folder := range pf.Folders {
		if folder == nil || folder.ID == nil {
			continue
		}
		fid := folder.ID.String()
		files, err := c.FolderFiles(ctx, fid)
		if err != nil {
			return nil, nil, err
		}
		filesByFolder[fid] = files
	}
	groups := FindProjectDuplicates(pf.Folders, filesByFolder, opts)
	if !apply || len(groups) == 0 {
		return groups, nil, nil
	}
	deleted, err := c.ApplyDedupGroups(ctx, groups)
	return groups, deleted, err
}
