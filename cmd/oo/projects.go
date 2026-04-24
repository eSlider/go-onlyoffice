package main

import (
	"fmt"
	"strconv"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"prj"},
	Short:   "OnlyOffice Projects",
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(prjListCmd())
	projectsCmd.AddCommand(prjGetCmd())
	projectsCmd.AddCommand(prjMilestonesCmd())
	projectsCmd.AddCommand(prjCreateCmd())
	projectsCmd.AddCommand(prjUpdateCmd())
	projectsCmd.AddCommand(prjDeleteCmd())
}

func prjListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, err := c.GetProjects()
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(list))
			for _, p := range list {
				row := map[string]any{
					"id":     derefInt(p.ID),
					"title":  p.String(),
					"status": derefInt(p.Status),
				}
				if p.TaskCount != nil {
					row["tasks"] = *p.TaskCount
				}
				if p.IsPrivate != nil {
					row["private"] = *p.IsPrivate
				}
				rows = append(rows, row)
			}
			printTable([]string{"id", "title", "status", "tasks", "private"}, rows)
			return nil
		},
	}
}

func prjGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [PROJECT_ID]",
		Short: "Show a single project (default: $OO_PROJECT_ID)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			out, err := c.GetProjectByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}

func prjMilestonesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "milestones PROJECT_ID",
		Short: "List milestones of a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("project id must be integer: %w", err)
			}
			ms, err := c.GetProjectMilestones(&onlyoffice.Project{ID: &pid})
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(ms))
			for _, m := range ms {
				rows = append(rows, map[string]any{
					"id":     derefInt64(m.ID),
					"title":  derefString(m.Title),
					"status": derefInt64(m.Status),
				})
			}
			printTable([]string{"id", "title", "status"}, rows)
			return nil
		},
	}
}

func prjCreateCmd() *cobra.Command {
	var desc, resp string
	cmd := &cobra.Command{
		Use:   "create TITLE",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			p, err := c.CreateProject(onlyoffice.NewProjectRequest{
				Title:         args[0],
				Description:   desc,
				ResponsibleID: resp,
			})
			if err != nil {
				return err
			}
			printObject(map[string]any{
				"id":    derefInt(p.ID),
				"title": p.String(),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&desc, "description", "", "project description")
	cmd.Flags().StringVar(&resp, "responsible", "", "responsible user id (default: self)")
	return cmd
}

func prjUpdateCmd() *cobra.Command {
	var title, desc, resp string
	cmd := &cobra.Command{
		Use:   "update PROJECT_ID",
		Short: "Update project fields (only non-empty)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("project id must be integer: %w", err)
			}
			p, err := c.UpdateProject(onlyoffice.ProjectUpdateRequest{
				ID:            id,
				Title:         title,
				Description:   desc,
				ResponsibleID: resp,
			})
			if err != nil {
				return err
			}
			printObject(map[string]any{
				"id":    derefInt(p.ID),
				"title": p.String(),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&desc, "description", "", "new description")
	cmd.Flags().StringVar(&resp, "responsible", "", "new responsible user id")
	return cmd
}

func prjDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete PROJECT_ID [PROJECT_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete one or more projects",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, raw := range args {
				id, err := strconv.Atoi(raw)
				if err != nil {
					return fmt.Errorf("project id %q must be integer: %w", raw, err)
				}
				p, err := c.DeleteProject(id)
				if err != nil {
					return err
				}
				printObject(map[string]any{"id": derefInt(p.ID), "title": p.String()})
			}
			return nil
		},
	}
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
