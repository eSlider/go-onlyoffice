package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/internal/applications"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "oo-cli",
	Short: "OnlyOffice Workspace CLI (Go port of cv/bin/office)",
}

// Execute runs the root command; the binary in cmd/oo-cli is a thin wrapper.
func Execute() error {
	return rootCmd.Execute()
}

// newOO loads env (incl. .env in CWD) and returns an authenticated client.
// godotenv is a CLI-only concern; the library itself never loads dotfiles.
func newOO() (*onlyoffice.Client, error) {
	_ = godotenv.Load()
	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" || creds.User == "" || creds.Password == "" {
		return nil, fmt.Errorf("need ONLYOFFICE_URL (or ONLYOFFICE_HOST), user (ONLYOFFICE_USER or ONLYOFFICE_NAME), password (ONLYOFFICE_PASS or ONLYOFFICE_PASSWORD)")
	}
	c := onlyoffice.NewClient(creds)
	c.SetDefaults(onlyoffice.GetEnvironmentDefaults())
	if err := c.Authenticate(); err != nil {
		return nil, err
	}
	return c, nil
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func init() {
	rootCmd.AddCommand(cmdCalList())
	rootCmd.AddCommand(cmdCalEvents())
	rootCmd.AddCommand(cmdCalAdd())
	rootCmd.AddCommand(cmdCalDel())
	rootCmd.AddCommand(cmdTaskList())
	rootCmd.AddCommand(cmdTaskAdd())
	rootCmd.AddCommand(cmdSubtaskAdd())
	rootCmd.AddCommand(cmdTaskUpdate())
	rootCmd.AddCommand(cmdCRMContacts())
	rootCmd.AddCommand(cmdCRMAddContact())
	rootCmd.AddCommand(cmdCRMDeals())
	rootCmd.AddCommand(cmdCRMAddDeal())
	rootCmd.AddCommand(cmdCRMCases())
	rootCmd.AddCommand(cmdAppsSync())
}

func cmdCalList() *cobra.Command {
	return &cobra.Command{
		Use:   "cal-list",
		Short: "List calendars (default date span)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			out, err := c.ListCalendars(cmd.Context(), "", "")
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
}

func cmdCalEvents() *cobra.Command {
	var start, end string
	cmd := &cobra.Command{
		Use:   "cal-events",
		Short: "List calendar data for period (default: next 7 days)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			if start == "" || end == "" {
				// Match list-events.py default window
				start = time.Now().Format("2006-01-02")
				end = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
			}
			out, err := c.ListEvents(cmd.Context(), start, end)
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&start, "start", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&end, "end", "", "end date YYYY-MM-DD")
	return cmd
}

func cmdCalAdd() *cobra.Command {
	var cal, desc string
	var allDay bool
	cmd := &cobra.Command{
		Use:   "cal-add TITLE START END",
		Short: "Add calendar event",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			ev, err := c.AddEvent(cmd.Context(), cal, args[0], args[1], args[2], desc, allDay)
			if err != nil {
				return err
			}
			printJSON(ev)
			return nil
		},
	}
	cmd.Flags().StringVar(&cal, "calendar", "", "calendar id (default from env)")
	cmd.Flags().StringVar(&desc, "description", "", "")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "")
	return cmd
}

func cmdCalDel() *cobra.Command {
	return &cobra.Command{
		Use:   "cal-delete EVENT_ID [EVENT_ID...]",
		Short: "Delete calendar events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			for _, id := range args {
				out, err := c.DeleteEvent(cmd.Context(), id)
				if err != nil {
					return err
				}
				printJSON(out)
			}
			return nil
		},
	}
}

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
			var tasks []map[string]interface{}
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
					subs, _ := d["subtasks"].([]interface{})
					for _, s := range subs {
						sm, _ := s.(map[string]interface{})
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
			if prio == "high" {
				p = 1
			}
			if prio == "low" {
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

func cmdCRMContacts() *cobra.Command {
	var companies, persons bool
	var search string
	cmd := &cobra.Command{
		Use:   "crm-contacts",
		Short: "List CRM contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			list, _, err := c.ListContacts(cmd.Context(), 50, 0, search)
			if err != nil {
				return err
			}
			for _, row := range list {
				isCo, _ := row["isCompany"].(bool)
				if companies && !isCo {
					continue
				}
				if persons && isCo {
					continue
				}
				printJSON(row)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&companies, "companies", false, "")
	cmd.Flags().BoolVar(&persons, "persons", false, "")
	cmd.Flags().StringVar(&search, "search", "", "")
	return cmd
}

func cmdCRMAddContact() *cobra.Command {
	var company, email, website, linkedin string
	var personFirst, personLast string
	var companyID int
	cmd := &cobra.Command{
		Use:   "crm-add-contact",
		Short: "Add company or person",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			if company != "" {
				out, err := c.CreateCompany(cmd.Context(), company)
				if err != nil {
					return err
				}
				cid := int(flexIDMap(out["id"]))
				if email != "" {
					_, _ = c.AddContactInfo(cmd.Context(), fmt.Sprint(cid), "Email", email, "Work", true)
				}
				if website != "" {
					_, _ = c.AddContactInfo(cmd.Context(), fmt.Sprint(cid), "Website", website, "Work", false)
				}
				printJSON(out)
				return nil
			}
			if personFirst != "" && personLast != "" {
				out, err := c.CreatePerson(cmd.Context(), personFirst, personLast, companyID, "", "")
				if err != nil {
					return err
				}
				pid := fmt.Sprint(int(flexIDMap(out["id"])))
				if email != "" {
					_, _ = c.AddContactInfo(cmd.Context(), pid, "Email", email, "Work", true)
				}
				if linkedin != "" {
					_, _ = c.AddContactInfo(cmd.Context(), pid, "LinkedIn", linkedin, "Work", false)
				}
				printJSON(out)
				return nil
			}
			return fmt.Errorf("set --company or --person-first and --person-last")
		},
	}
	cmd.Flags().StringVar(&company, "company", "", "")
	cmd.Flags().StringVar(&personFirst, "person-first", "", "")
	cmd.Flags().StringVar(&personLast, "person-last", "", "")
	cmd.Flags().IntVar(&companyID, "company-id", 0, "")
	cmd.Flags().StringVar(&email, "email", "", "")
	cmd.Flags().StringVar(&website, "website", "", "")
	cmd.Flags().StringVar(&linkedin, "linkedin", "", "")
	return cmd
}

func flexIDMap(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(x), 64)
		return f
	}
}

func cmdCRMDeals() *cobra.Command {
	var stages bool
	cmd := &cobra.Command{
		Use:   "crm-deals",
		Short: "List opportunities / stages",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			if stages {
				out, err := c.ListDealStages(cmd.Context())
				if err != nil {
					return err
				}
				printJSON(out)
				return nil
			}
			list, total, err := c.ListOpportunities(cmd.Context(), 100, 0)
			if err != nil {
				return err
			}
			fmt.Println("total:", total)
			printJSON(list)
			return nil
		},
	}
	cmd.Flags().BoolVar(&stages, "stages", false, "")
	return cmd
}

func cmdCRMAddDeal() *cobra.Command {
	var stage int
	var bid float64
	var contacts []string
	cmd := &cobra.Command{
		Use:   "crm-add-deal TITLE",
		Short: "Create opportunity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			if stage == 0 {
				stage = 1
			}
			out, err := c.CreateOpportunity(cmd.Context(), args[0], stage, "", "EUR", "", bid)
			if err != nil {
				return err
			}
			did := fmt.Sprint(int(flexIDMap(out["id"])))
			for _, cid := range contacts {
				_, _ = c.AddOpportunityMember(cmd.Context(), did, cid)
			}
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().IntVar(&stage, "stage", 1, "")
	cmd.Flags().Float64Var(&bid, "bid", 0, "")
	cmd.Flags().StringSliceVar(&contacts, "contact", nil, "contact id (repeatable)")
	return cmd
}

func cmdCRMCases() *cobra.Command {
	return &cobra.Command{
		Use:   "crm-cases",
		Short: "List CRM cases",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			list, total, err := c.ListCases(cmd.Context(), 50, 0)
			if err != nil {
				return err
			}
			fmt.Println("total:", total)
			printJSON(list)
			return nil
		},
	}
}

func cmdAppsSync() *cobra.Command {
	var apply, verbose bool
	var base string
	cmd := &cobra.Command{
		Use:   "applications-sync",
		Short: "Sync applications/*/README.md into CRM",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			if base == "" {
				return fmt.Errorf("use --path to applications year dir (e.g. .../applications/2026)")
			}
			paths, err := applications.Discover(base)
			if err != nil {
				return err
			}
			var apps []applications.Data
			for _, p := range paths {
				a, err := applications.ParseReadme(p)
				if err != nil {
					return err
				}
				if a.Position == "" && a.Company == "" {
					continue
				}
				apps = append(apps, a)
			}
			fmt.Printf("Found %d application(s)\n", len(apps))
			applications.Sync(cmd.Context(), c, apps, !apply, verbose)
			return nil
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "write to CRM (default dry-run)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "")
	cmd.Flags().StringVar(&base, "path", "", "applications base directory")
	return cmd
}
