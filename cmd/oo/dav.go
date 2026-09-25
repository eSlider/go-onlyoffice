package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(davCmd())
}

// davCmd exposes the Documents module through the backend-agnostic FileStore
// (DAV backend). The underlying Dav calls are the oo-webdav proven path:
// MoveDavItems sends resolveType=Skip + holdResult=true, which the legacy
// fileops/move call without those params silently ignores (200 without move).
func davCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dav",
		Short: "Documents module by folder/file id (oo-webdav proven path)",
	}
	cmd.AddCommand(davLsCmd())
	cmd.AddCommand(davMoveCmd())
	cmd.AddCommand(davCopyCmd())
	cmd.AddCommand(davMkdirCmd())
	cmd.AddCommand(davEnsurePathCmd())
	cmd.AddCommand(davUploadCmd())
	cmd.AddCommand(davRemoveCmd())
	cmd.AddCommand(davRenameFileCmd())
	cmd.AddCommand(davRenameFolderCmd())
	cmd.AddCommand(davDownloadCmd())
	cmd.AddCommand(davFileOpsCmd())
	return cmd
}

func davLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls FOLDER_ID",
		Short: "List a Documents folder (@root for virtual sections)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if args[0] == "@root" {
				sections, err := c.ListDavSections(ctx)
				if err != nil {
					return err
				}
				rows := make([]map[string]any, 0, len(sections))
				for _, s := range sections {
					rows = append(rows, map[string]any{
						"id":           s.ID,
						"title":        s.Title,
						"filesCount":   s.FilesCount,
						"foldersCount": s.FoldersCount,
					})
				}
				printTable([]string{"id", "title", "filesCount", "foldersCount"}, rows)
				return nil
			}
			entries, err := c.FileStore(onlyoffice.ProviderDAV).List(ctx, args[0])
			if err != nil {
				return err
			}
			folders := make([]onlyoffice.Entry, 0, len(entries))
			files := make([]onlyoffice.Entry, 0, len(entries))
			for _, e := range entries {
				if e.Kind == onlyoffice.Folder {
					folders = append(folders, e)
				} else {
					files = append(files, e)
				}
			}
			frows := make([]map[string]any, 0, len(folders))
			for _, f := range folders {
				frows = append(frows, map[string]any{
					"id":           f.ID,
					"title":        f.Title,
					"filesCount":   f.FilesCount,
					"foldersCount": f.FoldersCount,
				})
			}
			rows := make([]map[string]any, 0, len(files))
			for _, f := range files {
				rows = append(rows, map[string]any{
					"id":      f.ID,
					"title":   f.Title,
					"size":    f.Size,
					"updated": entryUpdated(f),
				})
			}
			if outputFormat == "json" {
				printObject(map[string]any{"folders": frows, "files": rows})
				return nil
			}
			if len(frows) > 0 {
				fmt.Println("folders:")
				printTable([]string{"id", "title", "filesCount", "foldersCount"}, frows)
			}
			fmt.Println("files:")
			printTable([]string{"id", "title", "size", "updated"}, rows)
			return nil
		},
	}
	return cmd
}

// entryUpdated prefers the backend-native timestamp string so table/JSON output
// round-trips what the API returned.
func entryUpdated(e onlyoffice.Entry) string {
	if e.Updated != "" {
		return e.Updated
	}
	if e.Modified.IsZero() {
		return ""
	}
	return e.Modified.Format(time.RFC3339)
}

func davMoveCmd() *cobra.Command {
	var folderIDs []string
	cmd := &cobra.Command{
		Use:   "move DEST_FOLDER_ID FILE_ID [FILE_ID...]",
		Short: "Move file(s) into a Documents folder (resolveType=Skip, holdResult)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ids := append(append([]string{}, folderIDs...), args[1:]...)
			if err := c.FileStore(onlyoffice.ProviderDAV).Move(cmd.Context(), ids, args[0]); err != nil {
				return err
			}
			printObject(map[string]any{"moved_files": args[1:], "moved_folders": folderIDs, "dest": args[0]})
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&folderIDs, "folders", nil, "folder ids to move along with the files")
	return cmd
}

func davCopyCmd() *cobra.Command {
	var folderIDs []string
	cmd := &cobra.Command{
		Use:   "copy DEST_FOLDER_ID FILE_ID [FILE_ID...]",
		Short: "Copy file(s) into a Documents folder (conflictResolveType=Skip)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ids := append(append([]string{}, folderIDs...), args[1:]...)
			if err := c.FileStore(onlyoffice.ProviderDAV).Copy(cmd.Context(), ids, args[0]); err != nil {
				return err
			}
			printObject(map[string]any{"copied_files": args[1:], "copied_folders": folderIDs, "dest": args[0]})
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&folderIDs, "folders", nil, "folder ids to copy along with the files")
	return cmd
}

func davMkdirCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mkdir PARENT_FOLDER_ID TITLE",
		Short: "Create a subfolder in Documents",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			f, err := c.FileStore(onlyoffice.ProviderDAV).CreateFolder(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			printObject(map[string]any{"id": f.ID, "title": f.Title, "parent": args[0]})
			return nil
		},
	}
}

func davEnsurePathCmd() *cobra.Command {
	var under string
	cmd := &cobra.Command{
		Use:   "ensure-path PATH",
		Short: "Resolve or create nested Documents folders (mkdir -p), print the final folder id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			root := under
			if root == "" {
				root, err = myDocumentsID(cmd.Context(), c)
				if err != nil {
					return err
				}
			}
			f, err := ensurePath(cmd.Context(), c.FileStore(onlyoffice.ProviderDAV), root, args[0])
			if err != nil {
				return err
			}
			printObject(map[string]any{"id": f.ID, "title": f.Title, "under": root})
			return nil
		},
	}
	cmd.Flags().StringVar(&under, "under", "", "parent folder id (default: My documents)")
	return cmd
}

func davUploadCmd() *cobra.Command {
	var replace bool
	cmd := &cobra.Command{
		Use:   "upload DEST_FOLDER_ID LOCAL_FILE [LOCAL_FILE...]",
		Short: "Upload local file(s) into a Documents folder (multipart, upsert by name)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			results, err := uploadLocal(cmd.Context(), c, args[0], args[1:], replace)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(results))
			for _, r := range results {
				replaced := make([]string, 0, len(r.Replaced))
				for _, id := range r.Replaced {
					replaced = append(replaced, fmt.Sprint(id))
				}
				rows = append(rows, map[string]any{
					"id":       r.Entry.ID,
					"title":    r.Entry.Title,
					"size":     r.Entry.Size,
					"replaced": strings.Join(replaced, ", "),
				})
			}
			printTable([]string{"id", "title", "size", "replaced"}, rows)
			return nil
		},
	}
	cmd.Flags().BoolVar(&replace, "replace", true, "replace same stem|ext before upload (default); false = fail if name taken")
	return cmd
}

// uploadedFile pairs an upload result with the ids deleted to make room for it.
type uploadedFile struct {
	Entry    onlyoffice.Entry
	Replaced []int
}

// uploader is the slice of *onlyoffice.Client that dav upload needs, so the
// command logic is unit-testable with a fake.
type uploader interface {
	UploadToFolderReplacing(ctx context.Context, folderID, localPath string) (*onlyoffice.FileEntry, []int, error)
	AssertNoFileConflict(ctx context.Context, folderID, localPath string) error
	UploadToFolder(ctx context.Context, folderID, localPath string) (*onlyoffice.FileEntry, error)
}

// uploadLocal uploads each local file into folderID. With replace a same
// stem|ext file is removed first (the UploadToFolderReplacing upsert); without
// replace a name clash fails with ErrFileExists before anything is sent.
func uploadLocal(ctx context.Context, up uploader, folderID string, paths []string, replace bool) ([]uploadedFile, error) {
	results := make([]uploadedFile, 0, len(paths))
	for _, path := range paths {
		var (
			fe      *onlyoffice.FileEntry
			deleted []int
			err     error
		)
		if replace {
			fe, deleted, err = up.UploadToFolderReplacing(ctx, folderID, path)
		} else {
			if err := up.AssertNoFileConflict(ctx, folderID, path); err != nil {
				return nil, err
			}
			fe, err = up.UploadToFolder(ctx, folderID, path)
		}
		if err != nil {
			return nil, fmt.Errorf("dav upload: %s: %w", path, err)
		}
		results = append(results, uploadedFile{
			Entry:    onlyoffice.FileEntryToEntry(fe, onlyoffice.ProviderDAV),
			Replaced: deleted,
		})
	}
	return results, nil
}

// ensurePath resolves-or-creates every "/"-separated segment under rootID
// through store. Existing folders are reused by title (case-insensitive), so
// repeated calls return the same id without creating duplicates.
func ensurePath(ctx context.Context, store onlyoffice.FileStore, rootID, path string) (onlyoffice.Entry, error) {
	segments, err := splitDavPath(path)
	if err != nil {
		return onlyoffice.Entry{}, err
	}
	parent := rootID
	var current onlyoffice.Entry
	for _, name := range segments {
		entries, err := store.List(ctx, parent)
		if err != nil {
			return onlyoffice.Entry{}, fmt.Errorf("dav ensure-path: list %s: %w", parent, err)
		}
		found := false
		for _, e := range entries {
			if e.Kind == onlyoffice.Folder && strings.EqualFold(strings.TrimSpace(e.Title), name) {
				current, found = e, true
				break
			}
		}
		if !found {
			current, err = store.CreateFolder(ctx, parent, name)
			if err != nil {
				return onlyoffice.Entry{}, fmt.Errorf("dav ensure-path: mkdir %s/%s: %w", parent, name, err)
			}
		}
		parent = current.ID
	}
	return current, nil
}

// splitDavPath normalizes a Documents path into non-empty segments. "." is
// ignored; ".." is rejected rather than creating a literal folder named "..".
func splitDavPath(path string) ([]string, error) {
	var segments []string
	for _, s := range strings.Split(path, "/") {
		s = strings.TrimSpace(s)
		switch s {
		case "", ".":
			continue
		case "..":
			return nil, fmt.Errorf("dav ensure-path: %q not allowed", s)
		}
		segments = append(segments, s)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("dav ensure-path: empty path")
	}
	return segments, nil
}

// myDocumentsID resolves the concrete id of the "My documents" section so
// ensure-path can create folders under it (the symbolic "@my" is list-only on
// some portals). rootFolderType 5 = My; the title match covers portals that
// omit rootFolderType.
func myDocumentsID(ctx context.Context, c *onlyoffice.Client) (string, error) {
	sections, err := c.ListDavSections(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range sections {
		if s.RootType == 5 && s.ID != "" {
			return s.ID, nil
		}
	}
	for _, s := range sections {
		if strings.EqualFold(strings.TrimSpace(s.Title), "My documents") && s.ID != "" {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("dav: no My documents section in @root; pass --under FOLDER_ID")
}

func davRemoveCmd() *cobra.Command {
	var folderIDs []string
	cmd := &cobra.Command{
		Use:     "rm [FILE_ID...]",
		Aliases: []string{"delete"},
		Short:   "Permanently delete file(s) and/or folder(s) from Documents",
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && len(folderIDs) == 0 {
				return fmt.Errorf("dav rm: give at least one FILE_ID or --folders")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ids := append(append([]string{}, args...), folderIDs...)
			if err := c.FileStore(onlyoffice.ProviderDAV).Delete(cmd.Context(), ids); err != nil {
				return err
			}
			printObject(map[string]any{"deleted_files": args, "deleted_folders": folderIDs})
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&folderIDs, "folders", nil, "folder ids to delete")
	return cmd
}

func davRenameFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename-file FILE_ID NEW_TITLE",
		Short: "Rename a Documents file (include extension in NEW_TITLE)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			if err := c.FileStore(onlyoffice.ProviderDAV).Rename(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			printObject(map[string]any{"id": args[0], "title": args[1]})
			return nil
		},
	}
}

func davRenameFolderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename-folder FOLDER_ID NEW_TITLE",
		Short: "Rename a Documents folder",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			if err := c.FileStore(onlyoffice.ProviderDAV).Rename(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			printObject(map[string]any{"id": args[0], "title": args[1]})
			return nil
		},
	}
}

func davDownloadCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "download FILE_ID",
		Short: "Download Documents file bytes (default path: ./<title>)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			store := c.FileStore(onlyoffice.ProviderDAV)
			e, err := store.Stat(ctx, args[0])
			if err != nil {
				return err
			}
			path := to
			if path == "" {
				path = onlyoffice.SafeLocalFileName(e.Title)
			}
			out, err := os.Create(path)
			if err != nil {
				return err
			}
			defer out.Close()
			n, err := store.Download(ctx, args[0], out)
			if err != nil {
				_ = os.Remove(path)
				return err
			}
			printObject(map[string]any{"path": path, "bytes": n})
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "output path (default: ./<server title>)")
	return cmd
}

func davFileOpsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fileops",
		Short: "List active file operations (move/copy status polling)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ops, err := c.ListFileOps(cmd.Context())
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(ops))
			for _, op := range ops {
				rows = append(rows, map[string]any{
					"id":        fmt.Sprint(op["id"]),
					"operation": fmt.Sprint(op["operation"]),
					"progress":  fmt.Sprint(op["progress"]),
					"finished":  fmt.Sprint(op["finished"]),
					"error":     fmt.Sprint(op["error"]),
				})
			}
			printTable([]string{"id", "operation", "progress", "finished", "error"}, rows)
			return nil
		},
	}
}
