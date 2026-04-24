package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
	Use:     "tasks",
	Aliases: []string{"task"},
	Short:   "Project tasks and subtasks",
}

func init() {
	rootCmd.AddCommand(tasksCmd)
	tasksCmd.AddCommand(taskListCmd())
	tasksCmd.AddCommand(taskGetCmd())
	tasksCmd.AddCommand(taskCreateCmd())
	tasksCmd.AddCommand(taskUpdateCmd())
	tasksCmd.AddCommand(taskDeleteCmd())
	tasksCmd.AddCommand(subtaskCmd())
}

func taskListCmd() *cobra.Command {
	var project, status string
	var all, verbose bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List project tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			var tasks []map[string]any
			if all {
				tasks, err = c.ListAllTasks(cmd.Context(), status)
			} else {
				tasks, err = c.ListTasks(cmd.Context(), project, status)
			}
			if err != nil {
				return err
			}
			if verbose {
				for i, t := range tasks {
					d, err := c.GetTaskByID(cmd.Context(), idString(t, "id"))
					if err == nil {
						tasks[i] = d
					}
				}
			}
			printTable([]string{"id", "title", "status", "priority", "responsible"}, tasks)
			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "p", "", "project id (default $OO_PROJECT_ID)")
	cmd.Flags().StringVarP(&status, "status", "s", "", "open|closed")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "all projects (@self)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "include subtasks & full details")
	return cmd
}

func taskGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get TASK_ID",
		Short: "Show a single task (incl. subtasks)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.GetTaskByID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}

func taskCreateCmd() *cobra.Command {
	var project, desc, deadline, prio string
	cmd := &cobra.Command{
		Use:     "create TITLE",
		Aliases: []string{"add"},
		Short:   "Create a project task",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			p := 0
			switch prio {
			case "high":
				p = 1
			case "low":
				p = -1
			}
			out, err := c.AddTask(cmd.Context(), project, args[0], desc, p, deadline)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "p", "", "project id (default $OO_PROJECT_ID)")
	cmd.Flags().StringVar(&desc, "description", "", "description")
	cmd.Flags().StringVar(&deadline, "deadline", "", "deadline YYYY-MM-DD")
	cmd.Flags().StringVar(&prio, "priority", "normal", "high|normal|low")
	return cmd
}

func taskUpdateCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "update TASK_ID",
		Short: "Update task status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if status == "" {
				return fmt.Errorf("--status is required (open|closed)")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.UpdateTaskStatus(cmd.Context(), args[0], status)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVarP(&status, "status", "s", "", "new status: open|closed")
	return cmd
}

func taskDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete TASK_ID [TASK_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete one or more tasks",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				out, err := c.DeleteTask(cmd.Context(), id)
				if err != nil {
					return err
				}
				printObject(out)
			}
			return nil
		},
	}
}

// subtaskCmd exposes `oo tasks subtask {add}` — keeps things tidy without a
// dedicated subtasks top-level command.
func subtaskCmd() *cobra.Command {
	sub := &cobra.Command{
		Use:   "subtask",
		Short: "Subtasks of a parent task",
	}
	sub.AddCommand(&cobra.Command{
		Use:   "add PARENT_TASK_ID TITLE [TITLE...]",
		Short: "Add one or more subtasks",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			parent := args[0]
			for _, title := range args[1:] {
				out, err := c.AddSubtask(cmd.Context(), parent, title)
				if err != nil {
					return err
				}
				printObject(out)
			}
			return nil
		},
	})
	return sub
}
