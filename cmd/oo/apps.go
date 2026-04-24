package main

import (
	"fmt"

	"github.com/eslider/go-onlyoffice/cmd/oo/applications"
	"github.com/spf13/cobra"
)

var applicationsCmd = &cobra.Command{
	Use:     "applications",
	Aliases: []string{"apps"},
	Short:   "Job-application workflow (CV tree → CRM)",
}

func init() {
	rootCmd.AddCommand(applicationsCmd)
	applicationsCmd.AddCommand(applicationsSyncCmd())
}

func applicationsSyncCmd() *cobra.Command {
	var apply, verbose bool
	var base string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync applications/*/README.md into CRM (dry-run by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			if base == "" {
				return fmt.Errorf("--path is required (points at applications year dir, e.g. .../applications/2026)")
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
