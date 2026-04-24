package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func cmdTaskList() *cobra.Command {
	var project, status string
	var all, verbose bool
	cmd := &cobra.Command{
		Use:   "task-list",
		Short: "List project tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
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
			for _, t := range tasks {
				fmt.Printf("[%v] %v status=%v\n", t["id"], t["title"], t["status"])
				if verbose {
					d, _ := c.GetTaskByID(cmd.Context(), fmt.Sprint(t["id"]))
					subs, _ := d["subtasks"].([]any)
					for _, s := range subs {
						sm, _ := s.(map[string]any)
						fmt.Printf("   └─ [%v] %v\n", sm["id"], sm["title"])
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&project, "project", "p", "", "project id")
	cmd.Flags().StringVarP(&status, "status", "s", "", "open|closed")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "all projects (@self)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "")
	return cmd
}

func cmdTaskAdd() *cobra.Command {
	var project, desc, deadline string
	var prio string
	cmd := &cobra.Command{
		Use:   "task-add TITLE",
		Short: "Add project task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
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
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "")
	cmd.Flags().StringVar(&desc, "description", "", "")
	cmd.Flags().StringVar(&deadline, "deadline", "", "")
	cmd.Flags().StringVar(&prio, "priority", "normal", "high|normal|low")
	return cmd
}

func cmdSubtaskAdd() *cobra.Command {
	return &cobra.Command{
		Use:   "subtask-add PARENT_TASK_ID TITLE [TITLE...]",
		Short: "Add subtask(s)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			parent := args[0]
			for _, title := range args[1:] {
				out, err := c.AddSubtask(cmd.Context(), parent, title)
				if err != nil {
					return err
				}
				printJSON(out)
			}
			return nil
		},
	}
}

func cmdTaskUpdate() *cobra.Command {
	var del bool
	cmd := &cobra.Command{
		Use:   "task-update TASK_ID open|closed",
		Short: "Update task status or delete",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			id := args[0]
			if del {
				out, err := c.DeleteTask(cmd.Context(), id)
				if err != nil {
					return err
				}
				printJSON(out)
				return nil
			}
			if len(args) < 2 {
				return fmt.Errorf("need status or --delete")
			}
			out, err := c.UpdateTaskStatus(cmd.Context(), id, args[1])
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().BoolVar(&del, "delete", false, "delete task")
	return cmd
}
