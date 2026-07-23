package main

import (
	"fmt"
	"strconv"
	"strings"

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
	crmTasksCmd.AddCommand(crmTasksReassignSelfCmd())
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
			printTable([]string{"id", "title", "deadLine", "isClosed", "categoryID"}, list)
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
		Short:   "Create a CRM task (assigns self + deadline default now+14d)",
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
	cmd.Flags().StringVar(&deadline, "deadline", "", "deadline (YYYY-MM-DD or ISO8601); default now+14d")
	cmd.Flags().IntVar(&categoryID, "category", 0, "category id (see `oo crm-tasks categories`)")
	cmd.Flags().IntVar(&contactID, "contact", 0, "contact id")
	cmd.Flags().StringVar(&entityType, "entity-type", "", "opportunity|case|contact")
	cmd.Flags().IntVar(&entityID, "entity-id", 0, "parent entity id")
	cmd.Flags().StringVar(&desc, "description", "", "description")
	return cmd
}

func crmTaskNeedsOwner(t map[string]any) (need bool, name string) {
	resp, _ := t["responsible"].(map[string]any)
	rid, _ := resp["id"].(string)
	name, _ = resp["displayName"].(string)
	need = rid == "" || strings.EqualFold(name, "Profile has been removed")
	return need, name
}

func crmTaskCategoryID(t map[string]any) int {
	cat, ok := t["category"].(map[string]any)
	if !ok {
		return 0
	}
	switch v := cat["id"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func crmTasksReassignSelfCmd() *cobra.Command {
	var apply bool
	var count, max int
	cmd := &cobra.Command{
		Use:   "reassign-self",
		Short: "Reassign CRM tasks with missing/removed owner to the authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			uid, err := c.SelfUserID(cmd.Context())
			if err != nil {
				return err
			}

			type item struct {
				id, title, deadline, ownerName string
				categoryID                     int
			}
			var todos []item
			scanned := 0
			for start := 0; ; start += count {
				list, total, err := c.ListCRMTasks(cmd.Context(), count, start)
				if err != nil {
					return err
				}
				for _, t := range list {
					scanned++
					need, name := crmTaskNeedsOwner(t)
					if !need {
						continue
					}
					title, _ := t["title"].(string)
					dl, _ := t["deadLine"].(string)
					todos = append(todos, item{
						id: fmt.Sprint(t["id"]), title: title, deadline: dl,
						ownerName: name, categoryID: crmTaskCategoryID(t),
					})
					if max > 0 && len(todos) >= max {
						break
					}
				}
				if (max > 0 && len(todos) >= max) || start+count >= total || len(list) == 0 {
					break
				}
			}

			fixed := 0
			for _, t := range todos {
				fmt.Printf("  task %s %q owner=%q → %s\n", t.id, t.title, t.ownerName, uid)
				if !apply {
					fixed++
					continue
				}
				if _, err := c.UpdateCRMTask(cmd.Context(), t.id, t.title, t.deadline, t.categoryID, uid); err != nil {
					return fmt.Errorf("update %s: %w", t.id, err)
				}
				fixed++
			}

			mode := "DRY-RUN"
			if apply {
				mode = "APPLIED"
			}
			fmt.Printf("%s: scanned=%d reassigned=%d (self=%s)\n", mode, scanned, fixed, uid)
			return nil
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "write changes (default dry-run)")
	cmd.Flags().IntVar(&count, "count", 100, "page size")
	cmd.Flags().IntVar(&max, "max", 0, "stop after N candidates (0 = all)")
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
