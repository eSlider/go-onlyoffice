package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

func init() {
	projectsCmd.AddCommand(projectFilesCmd())
}

func projectFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Project Documents folder: list, upload, download, rename, delete",
	}
	cmd.AddCommand(prjFilesListCmd())
	cmd.AddCommand(prjFilesUploadCmd())
	cmd.AddCommand(prjFilesReplaceInCmd())
	cmd.AddCommand(prjFilesUpdateCmd())
	cmd.AddCommand(prjFilesDownloadCmd())
	cmd.AddCommand(prjFilesRenameCmd())
	cmd.AddCommand(prjFilesDeleteCmd())
	cmd.AddCommand(prjFilesDedupeCmd())
	// Convenience aliases into oo docs (md↔docx / OCR pipeline).
	cmd.AddCommand(aliasDocsAsMD())
	cmd.AddCommand(aliasDocsPutMD())
	cmd.AddCommand(aliasDocsPutTxt())
	cmd.AddCommand(aliasDocsPutXlsx())
	return cmd
}

func aliasDocsAsMD() *cobra.Command {
	c := docsAsMDCmd()
	c.Use = "as-md FILE_ID"
	c.Short = "Alias of `oo docs as-md` — download OO file as Markdown (OCR if needed)"
	return c
}

func aliasDocsPutMD() *cobra.Command {
	c := docsPutMDCmd()
	c.Use = "put-md PROJECT_ID MARKDOWN_PATH"
	c.Short = "Alias of `oo docs put-md` — Markdown→DOCX upload into project"
	return c
}

func aliasDocsPutTxt() *cobra.Command {
	c := docsPutTxtCmd()
	c.Use = "put-txt PROJECT_ID TEXT_PATH"
	c.Short = "Alias of `oo docs put-txt` — plain text→DOCX upload into project"
	return c
}

func aliasDocsPutXlsx() *cobra.Command {
	c := docsPutXlsxCmd()
	c.Use = "put-xlsx PROJECT_ID [LOCAL_XLSX]"
	c.Short = "Alias of `oo docs put-xlsx` — generate/upload XLSX with formulas"
	return c
}

func prjFilesListCmd() *cobra.Command {
	var showFolders bool
	cmd := &cobra.Command{
		Use:   "list PROJECT_ID",
		Short: "List files (and optionally folders) attached to the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pf, err := c.GetProjectFiles(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if showFolders && len(pf.Folders) > 0 {
				frows := make([]map[string]any, 0, len(pf.Folders))
				for _, f := range pf.Folders {
					if f == nil {
						continue
					}
					frows = append(frows, map[string]any{
						"id":           folderIDStr(f),
						"title":        derefString(f.Title),
						"filesCount":   derefInt(f.FilesCount),
						"foldersCount": derefInt(f.FoldersCount),
					})
				}
				if outputFormat == "table" {
					fmt.Println("folders:")
				}
				printTable([]string{"id", "title", "filesCount", "foldersCount"}, frows)
			}
			rows := fileEntryRows(pf.Files)
			if outputFormat == "table" {
				fmt.Println("files:")
			}
			printTable([]string{"id", "title", "fileExst", "contentLength", "updated"}, rows)
			return nil
		},
	}
	cmd.Flags().BoolVar(&showFolders, "folders", false, "also print project subfolders")
	return cmd
}

func prjFilesUploadCmd() *cobra.Command {
	var replace, allowDuplicate bool
	cmd := &cobra.Command{
		Use:   "upload PROJECT_ID LOCAL_PATH [LOCAL_PATH...]",
		Short: "Upload file(s) into the project's Documents folder (upsert by stem|ext)",
		Long: `Default: replace an existing file with the same logical name (stem|ext), like cp overwrite.
Pass --no-replace to fail when the name is taken; --allow-duplicate to always create a new file id.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid := args[0]
			for _, p := range args[1:] {
				var entry *onlyoffice.FileEntry
				var deleted []int
				switch {
				case allowDuplicate:
					entry, err = c.UploadProjectFile(cmd.Context(), pid, p)
				case replace:
					entry, deleted, err = c.UploadProjectFileReplacing(cmd.Context(), pid, p)
				default:
					entry, err = c.UploadProjectFileNoClobber(cmd.Context(), pid, p)
				}
				if err != nil {
					return err
				}
				obj := fileEntryToMap(entry)
				if len(deleted) > 0 {
					obj["replaced_file_ids"] = deleted
				}
				printObject(obj)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&replace, "replace", true, "replace same stem|ext in project folder (default)")
	cmd.Flags().BoolVar(&allowDuplicate, "allow-duplicate", false, "always create a new file even when the name exists")
	return cmd
}

func prjFilesReplaceInCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "replace-in FOLDER_ID LOCAL_PATH [LOCAL_PATH...]",
		Short: "Replace same-named file(s) in a folder: hard delete + fresh upload (no version history)",
		Long: `Deletes any file in FOLDER_ID with the same stem|ext (hard delete — the CLI
delete is permanent) and uploads the local file fresh. Unlike 'update' this
leaves a single clean version, which matters when the file id is shared.

Note: file ids are server-assigned; a fresh upload gets a new id.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			folderID := args[0]
			for _, p := range args[1:] {
				stem := onlyoffice.UploadStemFromLocal(p)
				ext := onlyoffice.UploadExtFromLocal(p)
				deleted, derr := c.DeleteFilesByDedupKey(cmd.Context(), folderID, stem, ext)
				if derr != nil {
					return derr
				}
				ent, uerr := c.UploadToFolder(cmd.Context(), folderID, p)
				if uerr != nil {
					return uerr
				}
				obj := fileEntryToMap(ent)
				if len(deleted) > 0 {
					obj["replaced_file_ids"] = deleted
				}
				printObject(obj)
			}
			return nil
		},
	}
}

func prjFilesUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update FILE_ID LOCAL_PATH",
		Short: "Overwrite an existing Documents file with new content (new version)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			entry, err := c.UpdateFile(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			printObject(fileEntryToMap(entry))
			return nil
		},
	}
}

func prjFilesDownloadCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "download FILE_ID",
		Short: "Download file bytes via viewUrl (default path: ./<title>)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			store := c.Files()
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
			if outputFormat == "json" {
				printObject(map[string]any{"path": path, "bytes": n})
				return nil
			}
			fmt.Printf("downloaded: %s (%d bytes)\n", path, n)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "output path (default: ./<server title>)")
	return cmd
}

func prjFilesRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename FILE_ID NEW_TITLE",
		Short: "Rename a file (include extension in NEW_TITLE)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			store := c.Files()
			if err := store.Rename(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			entry, err := store.Stat(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			entry.Title = args[1]
			printObject(entryToMap(entry))
			return nil
		},
	}
}

func prjFilesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete FILE_ID [FILE_ID...]",
		Aliases: []string{"rm"},
		Short:   "Permanently delete file(s) from Documents",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ids := make([]int, 0, len(args))
			for _, s := range args {
				id, err := strconv.Atoi(s)
				if err != nil {
					return fmt.Errorf("file id %q: %w", s, err)
				}
				ids = append(ids, id)
			}
			if err := c.Files().Delete(cmd.Context(), args); err != nil {
				return err
			}
			printObject(map[string]any{"deleted": ids})
			return nil
		},
	}
}

func prjFilesDedupeCmd() *cobra.Command {
	var apply, cross bool
	cmd := &cobra.Command{
		Use:   "dedupe PROJECT_ID",
		Short: "Find (and optionally remove) duplicate files in project Documents folders",
		Long: `Duplicates share the same logical name: stem|ext (OO title+fileExst).

Default: dry-run report. Pass --apply to delete older copies (keeps newest; --cross prefers non-trash folders).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			groups, deleted, err := c.DedupeProject(cmd.Context(), args[0], onlyoffice.DedupOptions{
				CrossFolder: cross,
			}, apply)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(groups))
			for _, g := range groups {
				row := map[string]any{
					"key":          g.Key,
					"folder_id":    g.FolderID,
					"folder_title": g.FolderTitle,
					"keep_id":      fileIDStr(g.Keep),
					"keep_title":   onlyoffice.FileEntryTitle(g.Keep),
					"remove_count": len(g.Remove),
				}
				removeIDs := make([]string, 0, len(g.Remove))
				for _, f := range g.Remove {
					removeIDs = append(removeIDs, fileIDStr(f))
				}
				row["remove_ids"] = removeIDs
				rows = append(rows, row)
			}
			out := map[string]any{
				"project_id": args[0],
				"dry_run":    !apply,
				"cross":      cross,
				"groups":     len(groups),
				"duplicates": rows,
			}
			if apply {
				out["deleted_ids"] = deleted
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "delete duplicate files (default: report only)")
	cmd.Flags().BoolVar(&cross, "cross", false, "also dedupe same stem|ext across folders (prefers non-_trash)")
	return cmd
}

func fileEntryRows(files []*onlyoffice.FileEntry) []map[string]any {
	rows := make([]map[string]any, 0, len(files))
	for _, f := range files {
		if f == nil {
			continue
		}
		rows = append(rows, fileEntryToMap(f))
	}
	return rows
}

func fileEntryToMap(f *onlyoffice.FileEntry) map[string]any {
	m := map[string]any{
		"id":            fileIDStr(f),
		"title":         onlyoffice.FileEntryTitle(f),
		"fileExst":      derefString(f.FileExst),
		"contentLength": derefString(f.ContentLength),
	}
	if f.Updated != nil {
		m["updated"] = f.Updated.Format(time.RFC3339)
	}
	return m
}

// entryToMap renders a canonical Entry with the same keys as fileEntryToMap.
func entryToMap(e onlyoffice.Entry) map[string]any {
	m := map[string]any{
		"id":            e.ID,
		"title":         e.Title,
		"fileExst":      filepath.Ext(e.Title),
		"contentLength": contentLengthString(e.Size),
	}
	if e.Updated != "" {
		m["updated"] = e.Updated
	} else if !e.Modified.IsZero() {
		m["updated"] = e.Modified.Format(time.RFC3339)
	}
	return m
}

func contentLengthString(n int64) string {
	if n <= 0 {
		return ""
	}
	return strconv.FormatInt(n, 10)
}

func fileIDStr(f *onlyoffice.FileEntry) string {
	if f == nil || f.ID == nil {
		return ""
	}
	return f.ID.String()
}

func folderIDStr(f *onlyoffice.FolderEntry) string {
	if f == nil || f.ID == nil {
		return ""
	}
	return f.ID.String()
}
