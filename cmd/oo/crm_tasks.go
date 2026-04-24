package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var crmTasksCmd = &cobra.Command{
	Use:     "crm-tasks",
	Aliases: []string{"crm-task"},
	Short:   "CRM tasks (standalone, not project tasks)",
}

func init() {
	rootCmd.AddCommand(crmTasksCmd)
	crmTasksCmd.AddCommand(crmTasksListCmd())
	crmTasksCmd.AddCommand(crmTasksCreateCmd())
	crmTasksCmd.AddCommand(crmTasksDeleteCmd())
	crmTasksCmd.AddCommand(crmTasksCategoriesCmd())
}

func crmTasksListCmd() *cobra.Command {
	var count, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List CRM tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, total, err := c.ListCRMTasks(cmd.Context(), count, offset)
			if err != nil {
				return err
			}
			if outputFormat == "table" {
				fmt.Printf("total: %d (shown: %d)\n", total, len(list))
			}
			printTable([]string{"id", "title", "deadline", "isClosed", "categoryID"}, list)
			return nil
		},
	}
	cmd.Flags().IntVar(&count, "count", 50, "")
	cmd.Flags().IntVar(&offset, "offset", 0, "")
	return cmd
}

func crmTasksCreateCmd() *cobra.Command {
	var deadline, desc, entityType string
	var categoryID, contactID, entityID int
	cmd := &cobra.Command{
		Use:     "create TITLE",
		Aliases: []string{"add"},
		Short:   "Create a CRM task",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.CreateCRMTask(cmd.Context(), args[0], deadline, categoryID, contactID, entityType, entityID, desc)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&deadline, "deadline", "", "deadline (YYYY-MM-DD or ISO8601)")
	cmd.Flags().IntVar(&categoryID, "category", 0, "category id (see `oo crm-tasks categories`)")
	cmd.Flags().IntVar(&contactID, "contact", 0, "contact id")
	cmd.Flags().StringVar(&entityType, "entity-type", "", "opportunity|case|contact")
	cmd.Flags().IntVar(&entityID, "entity-id", 0, "parent entity id")
	cmd.Flags().StringVar(&desc, "description", "", "description")
	return cmd
}

func crmTasksDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete TASK_ID [TASK_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete CRM tasks",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				if _, err := strconv.Atoi(id); err != nil {
					return fmt.Errorf("task id %q must be integer: %w", id, err)
				}
				out, err := c.DeleteCRMTask(cmd.Context(), id)
				if err != nil {
					return err
				}
				printObject(out)
			}
			return nil
		},
	}
}

func crmTasksCategoriesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "categories",
		Short: "List CRM task categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.ListTaskCategories(cmd.Context())
			if err != nil {
				return err
			}
			printTable([]string{"id", "title", "sortOrder", "imagePath"}, out)
			return nil
		},
	}
}
