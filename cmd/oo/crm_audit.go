package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

func init() {
	crmCmd.AddCommand(crmAuditCmd())
}

func crmAuditCmd() *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit opportunities (files/tasks/members per deal) and classify",
		Long: `Lists every opportunity with its file/task/member counts and a coarse class
(ok | dup | empty | junk-title). --out writes the full JSON audit for later use.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			audits, err := c.AuditOpportunities(cmd.Context())
			if err != nil {
				return err
			}
			if outPath != "" {
				b, merr := json.MarshalIndent(audits, "", "  ")
				if merr != nil {
					return merr
				}
				if werr := os.WriteFile(outPath, b, 0o644); werr != nil {
					return werr
				}
			}
			byClass := map[string]int{}
			for _, a := range audits {
				byClass[a.Class]++
			}
			classes := make([]string, 0, len(byClass))
			for k := range byClass {
				classes = append(classes, k)
			}
			sort.Strings(classes)
			rows := make([]map[string]any, 0, len(classes))
			for _, k := range classes {
				rows = append(rows, map[string]any{"class": k, "count": byClass[k]})
			}
			printTable([]string{"class", "count"}, rows)
			if outPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "wrote %d audits → %s\n", len(audits), outPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outPath, "out", "", "write audit JSON to this file")
	return cmd
}
