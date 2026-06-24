package main

import (
	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

var crmCmd = &cobra.Command{
	Use:   "crm",
	Short: "CRM maintenance (dedupe, cleanup)",
}

func init() {
	rootCmd.AddCommand(crmCmd)
	crmCmd.AddCommand(crmCleanupCmd())
}

func crmCleanupCmd() *cobra.Command {
	var ignoreCompanySuffix bool
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Run all CRM dedupe passes (companies, persons, associations, titles)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := onlyoffice.CleanupCRM(cmd.Context(), c, ignoreCompanySuffix)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().BoolVar(&ignoreCompanySuffix, "ignore-company-suffix", false, "group deals by position only (strip ' @ Company')")
	return cmd
}

func dedupeRunE(fn func(cmd *cobra.Command, c *onlyoffice.Client) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c, err := newOO(cmd)
		if err != nil {
			return err
		}
		return fn(cmd, c)
	}
}
