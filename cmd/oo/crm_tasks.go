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
			fixed, scanned := 0, 0

			if !apply {
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
						fixed++
						fmt.Printf("  task %s %q owner=%q → %s\n", fmt.Sprint(t["id"]), t["title"], name, uid)
					}
					if start+count >= total || len(list) == 0 {
						break
					}
				}
			} else {
				// Always re-fetch offset 0 so fixed tasks don't shift the window.
				for {
					list, _, err := c.ListCRMTasks(cmd.Context(), count, 0)
					if err != nil {
						return err
					}
					if len(list) == 0 {
						break
					}
					pageFixed := 0
					for _, t := range list {
						scanned++
						need, name := crmTaskNeedsOwner(t)
						if !need {
							continue
						}
						id := fmt.Sprint(t["id"])
						title, _ := t["title"].(string)
						dl, _ := t["deadLine"].(string)
						fmt.Printf("  task %s %q owner=%q → %s\n", id, title, name, uid)
						if _, err := c.UpdateCRMTask(cmd.Context(), id, title, dl, crmTaskCategoryID(t), uid); err != nil {
							return fmt.Errorf("update %s: %w", id, err)
						}
						pageFixed++
						fixed++
						if max > 0 && fixed >= max {
							break
						}
					}
					if pageFixed == 0 || (max > 0 && fixed >= max) {
						break
					}
				}
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
	cmd.Flags().IntVar(&max, "max", 0, "stop after N reassignments (0 = all)")
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
