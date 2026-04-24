package main

import (
	"fmt"

	"github.com/eslider/go-onlyoffice/cmd/oo/applications"
	"github.com/spf13/cobra"
)

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
