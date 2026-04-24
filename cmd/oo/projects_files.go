package main

import (
	"fmt"
	"os"
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
	cmd.AddCommand(prjFilesDownloadCmd())
	cmd.AddCommand(prjFilesRenameCmd())
	cmd.AddCommand(prjFilesDeleteCmd())
	return cmd
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
						"id":            folderIDStr(f),
						"title":       derefString(f.Title),
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
	return &cobra.Command{
		Use:   "upload PROJECT_ID LOCAL_PATH [LOCAL_PATH...]",
		Short: "Upload file(s) into the project's Documents folder",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid := args[0]
			for _, p := range args[1:] {
				entry, err := c.UploadProjectFile(cmd.Context(), pid, p)
				if err != nil {
					return err
				}
				printObject(fileEntryToMap(entry))
			}
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
			f, err := c.GetFile(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			path := to
			if path == "" {
				path = onlyoffice.SafeLocalFileName(onlyoffice.FileEntryTitle(f))
			}
			out, err := os.Create(path)
			if err != nil {
				return err
			}
			defer out.Close()
			n, err := c.DownloadFile(cmd.Context(), args[0], out)
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
			entry, err := c.RenameFile(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			printObject(fileEntryToMap(entry))
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
			if err := c.DeleteFiles(cmd.Context(), ids); err != nil {
				return err
			}
			printObject(map[string]any{"deleted": ids})
			return nil
		},
	}
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
