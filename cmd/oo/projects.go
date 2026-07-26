package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/catalog"
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
	projectsCmd.AddCommand(prjContactsCmd())
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
	var country, company string
	cmd := &cobra.Command{
		Use:   "create TITLE",
		Short: "Create a new project",
		Long: `Create a project. Prefer canonical titles:

  CC | Company | Title

Examples:
  oo projects create "Mapbender" --country DE --company "Stadt Mainz"
  oo projects create "DE | Acme | Geo Engineer"

When --country and --company are set, TITLE is only the third segment and the
full title is composed as "CC | Company | Title".`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			title := args[0]
			if country != "" || company != "" {
				title = catalog.FormatProjectTitle(country, company, args[0])
			}
			p, err := c.CreateProject(onlyoffice.NewProjectRequest{
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
	cmd.Flags().StringVar(&desc, "description", "", "project description")
	cmd.Flags().StringVar(&resp, "responsible", "", "responsible user id (default: self)")
	cmd.Flags().StringVar(&country, "country", "", "country/region code (DE, TF, UA, …)")
	cmd.Flags().StringVar(&company, "company", "", "CRM company / engagement org name")
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
			// OO update requires responsibleId; reuse current when not passed.
			if resp == "" {
				cur, gerr := c.GetProjectByID(cmd.Context(), args[0])
				if gerr != nil {
					return gerr
				}
				if r, ok := cur["responsible"].(map[string]any); ok {
					resp = fmt.Sprint(r["id"])
				}
				if resp == "" || resp == "<nil>" {
					resp = strings.TrimSpace(fmt.Sprint(cur["responsibleId"]))
				}
				if title == "" {
					title = strings.TrimSpace(fmt.Sprint(cur["title"]))
				}
				if desc == "" {
					desc = strings.TrimSpace(fmt.Sprint(cur["description"]))
				}
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
			// Confirm via GET — some OO builds return an empty body on PUT.
			got, gerr := c.GetProjectByID(cmd.Context(), args[0])
			if gerr == nil && got != nil {
				printObject(map[string]any{"confirmed_title": got["title"], "confirmed_id": got["id"]})
			}
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

func prjContactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "CRM contacts linked to a project (companies + persons)",
		Long: `OnlyOffice Projects have two participant layers:
  - Team (portal users) — ProjectTeam.aspx
  - Contacts (CRM companies/persons) — linked via this API

Link employers/clients and people who worked on the engagement (e.g. git authors).`,
	}
	cmd.AddCommand(prjContactsListCmd())
	cmd.AddCommand(prjContactsAddCmd())
	cmd.AddCommand(prjContactsRemoveCmd())
	cmd.AddCommand(prjContactsLinkGitCmd())
	cmd.AddCommand(prjContactsLinkAuthorsCmd())
	return cmd
}

func prjContactsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list PROJECT_ID",
		Short: "List CRM contacts linked to the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("project id: %w", err)
			}
			list, err := c.ListProjectContacts(cmd.Context(), pid)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(list))
			for _, row := range list {
				rows = append(rows, map[string]any{
					"id":        row["id"],
					"name":      row["displayName"],
					"company":   row["isCompany"],
					"companyOf": row["companyName"],
				})
			}
			printTable([]string{"id", "name", "company", "companyOf"}, rows)
			return nil
		},
	}
}

func prjContactsAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add PROJECT_ID CONTACT_ID [CONTACT_ID...]",
		Short: "Link CRM contact(s) to the project",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("project id: %w", err)
			}
			for _, raw := range args[1:] {
				cid, err := strconv.Atoi(raw)
				if err != nil {
					return fmt.Errorf("contact id %q: %w", raw, err)
				}
				if _, err := c.AddProjectContact(cmd.Context(), pid, cid); err != nil {
					return fmt.Errorf("add %d: %w", cid, err)
				}
				printObject(map[string]any{"project_id": pid, "contact_id": cid, "linked": true})
			}
			return nil
		},
	}
}

func prjContactsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove PROJECT_ID CONTACT_ID [CONTACT_ID...]",
		Short: "Unlink CRM contact(s) from the project",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("project id: %w", err)
			}
			for _, raw := range args[1:] {
				cid, err := strconv.Atoi(raw)
				if err != nil {
					return fmt.Errorf("contact id %q: %w", raw, err)
				}
				if err := c.RemoveProjectContact(cmd.Context(), pid, cid); err != nil {
					return fmt.Errorf("remove %d: %w", cid, err)
				}
				printObject(map[string]any{"project_id": pid, "contact_id": cid, "unlinked": true})
			}
			return nil
		},
	}
}

func prjContactsLinkGitCmd() *cobra.Command {
	var gitRoot string
	var minCommits int
	var dryRun bool
	var companyIDs []int
	cmd := &cobra.Command{
		Use:   "link-git PROJECT_ID",
		Short: "Link CRM persons matched from git shortlog (+ optional companies)",
		Long: `Runs git shortlog -sne --all in --git-root, matches author emails to CRM
persons (FindPersonByEmail), and links found contacts to the project.
Does not create new persons — approve/apply them via catalog first.

Also links --company-id contacts (e.g. Acme + end-client company).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if gitRoot == "" && len(companyIDs) == 0 {
				return fmt.Errorf("need --git-root and/or --company-id")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("project id: %w", err)
			}
			var authors []gitAuthor
			if gitRoot != "" {
				authors, err = gitShortlogAuthors(gitRoot, minCommits)
				if err != nil {
					return err
				}
			}
			return linkProjectPeople(cmd, c, pid, authors, companyIDs, dryRun)
		},
	}
	cmd.Flags().StringVar(&gitRoot, "git-root", "", "local git repository path")
	cmd.Flags().IntVar(&minCommits, "min-commits", 3, "ignore authors below this commit count")
	cmd.Flags().IntSliceVar(&companyIDs, "company-id", nil, "CRM company contact ids to link")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print planned links only")
	return cmd
}

func prjContactsLinkAuthorsCmd() *cobra.Command {
	var authorsFile string
	var minCommits int
	var dryRun bool
	var companyIDs []int
	cmd := &cobra.Command{
		Use:   "link-authors PROJECT_ID",
		Short: "Link CRM persons from a git shortlog file (+ companies)",
		Long: `Same as link-git, but reads authors from a file produced by:

  git shortlog -sne --all > authors.txt
  # or via ssh:
  ssh ops-host 'git -C /path shortlog -sne --all' > authors.txt

Then: oo projects contacts link-authors 59 --from authors.txt --company-id 9`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if authorsFile == "" && len(companyIDs) == 0 {
				return fmt.Errorf("need --from and/or --company-id")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("project id: %w", err)
			}
			var authors []gitAuthor
			if authorsFile != "" {
				b, err := os.ReadFile(authorsFile)
				if err != nil {
					return err
				}
				authors = parseShortlog(string(b), minCommits)
			}
			return linkProjectPeople(cmd, c, pid, authors, companyIDs, dryRun)
		},
	}
	cmd.Flags().StringVar(&authorsFile, "from", "", "path to git shortlog -sne output")
	cmd.Flags().IntVar(&minCommits, "min-commits", 3, "ignore authors below this commit count")
	cmd.Flags().IntSliceVar(&companyIDs, "company-id", nil, "CRM company contact ids to link")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print planned links only")
	return cmd
}

func linkProjectPeople(cmd *cobra.Command, c *onlyoffice.Client, pid int, authors []gitAuthor, companyIDs []int, dryRun bool) error {
	ctx := cmd.Context()
	existing, err := c.ListProjectContacts(ctx, pid)
	if err != nil {
		return err
	}
	have := map[int]struct{}{}
	for _, row := range existing {
		have[int(flexIDFloat(row["id"]))] = struct{}{}
	}
	linked, skipped := 0, 0
	for _, cid := range companyIDs {
		if _, ok := have[cid]; ok {
			skipped++
			continue
		}
		if dryRun {
			printObject(map[string]any{"would_link_company": cid})
			linked++
			continue
		}
		if _, err := c.AddProjectContact(ctx, pid, cid); err != nil {
			return fmt.Errorf("company %d: %w", cid, err)
		}
		have[cid] = struct{}{}
		linked++
		printObject(map[string]any{"linked_company": cid})
	}
	for _, a := range authors {
		p, err := c.FindPersonByEmail(ctx, a.Email)
		if err != nil {
			return err
		}
		if p == nil {
			printObject(map[string]any{
				"skipped_email": a.Email, "name": a.Name, "commits": a.Commits, "reason": "no_crm_person",
			})
			skipped++
			continue
		}
		cid := int(flexIDFloat(p["id"]))
		if _, ok := have[cid]; ok {
			skipped++
			continue
		}
		if dryRun {
			printObject(map[string]any{"would_link_person": cid, "email": a.Email, "commits": a.Commits})
			linked++
			continue
		}
		if _, err := c.AddProjectContact(ctx, pid, cid); err != nil {
			return fmt.Errorf("person %d (%s): %w", cid, a.Email, err)
		}
		have[cid] = struct{}{}
		linked++
		printObject(map[string]any{"linked_person": cid, "email": a.Email, "commits": a.Commits})
	}
	printObject(map[string]any{"project_id": pid, "linked": linked, "skipped": skipped, "dry_run": dryRun})
	return nil
}

func parseShortlog(text string, minCommits int) []gitAuthor {
	var authors []gitAuthor
	seen := map[string]struct{}{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		n, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		if n < minCommits {
			continue
		}
		rest := parts[1]
		lt := strings.LastIndex(rest, "<")
		gt := strings.LastIndex(rest, ">")
		if lt < 0 || gt <= lt {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(rest[lt+1 : gt]))
		name := strings.TrimSpace(rest[:lt])
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		authors = append(authors, gitAuthor{Commits: n, Name: name, Email: email})
	}
	return authors
}

type gitAuthor struct {
	Commits int
	Name    string
	Email   string
}

func gitShortlogAuthors(root string, minCommits int) ([]gitAuthor, error) {
	cmd := exec.Command("git", "-C", root, "shortlog", "-sne", "--all")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git shortlog in %s: %w", root, err)
	}
	return parseShortlog(string(out), minCommits), nil
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
