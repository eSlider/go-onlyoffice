package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	tasksCmd.AddCommand(taskFilesCmd())
}

func taskFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Task attachments: list, upload (project folder + attach), detach",
	}
	cmd.AddCommand(taskFilesListCmd())
	cmd.AddCommand(taskFilesUploadCmd())
	cmd.AddCommand(taskFilesDetachCmd())
	return cmd
}

func taskFilesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list TASK_ID",
		Short: "List files attached to a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, err := c.GetTaskFiles(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printTable([]string{"id", "title", "fileExst", "contentLength", "updated"}, fileEntryRows(list))
			return nil
		},
	}
}

func taskFilesUploadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upload TASK_ID LOCAL_PATH [LOCAL_PATH...]",
		Short: "Upload into the task's project folder and attach each file to the task",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			tid := args[0]
			for _, p := range args[1:] {
				entry, err := c.UploadTaskFile(cmd.Context(), tid, p)
				if err != nil {
					return err
				}
				printObject(fileEntryToMap(entry))
			}
			return nil
		},
	}
}

func taskFilesDetachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detach TASK_ID FILE_ID [FILE_ID...]",
		Short: "Detach file(s) from the task (files remain in Documents)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			tid := args[0]
			for _, fid := range args[1:] {
				if _, err := strconv.Atoi(fid); err != nil {
					return fmt.Errorf("file id %q: %w", fid, err)
				}
				if err := c.DetachTaskFile(cmd.Context(), tid, fid); err != nil {
					return err
				}
				printObject(map[string]any{"taskId": tid, "detachedFileId": fid})
			}
			return nil
		},
	}
}
