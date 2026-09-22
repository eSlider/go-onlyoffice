package main

import (
	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

func init() {
	projectsCmd.AddCommand(prjBoardSyncCmd())
}

// prjBoardSyncCmd upserts project milestones/tasks from a board YAML.
func prjBoardSyncCmd() *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use:   "board-sync BOARD.yaml",
		Short: "Upsert project milestones/tasks from a board YAML (dry-run by default)",
		Long: `Reads a board YAML (projects → milestones → tasks) and creates only the
milestones/tasks that are missing, matching by exact title. Existing entries are
left untouched, so the same file can be re-applied safely.

Dry-run by default; pass --apply to write to OnlyOffice.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			board, err := onlyoffice.LoadBoard(args[0])
			if err != nil {
				return err
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			res, err := c.SyncBoard(cmd.Context(), board, apply)
			if err != nil {
				return err
			}
			mode := "dry-run"
			if apply {
				mode = "apply"
			}
			printObject(map[string]any{
				"mode":               mode,
				"projects":           len(board.Projects),
				"created_milestones": res.CreatedMilestones,
				"skipped_milestones": res.SkippedMilestones,
				"created_tasks":      res.CreatedTasks,
				"skipped_tasks":      res.SkippedTasks,
			})
			return nil
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "write to OnlyOffice (default: dry-run)")
	return cmd
}
