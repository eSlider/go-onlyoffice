package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var opportunitiesCmd = &cobra.Command{
	Use:     "opportunities",
	Aliases: []string{"deals", "deal"},
	Short:   "CRM opportunities (a.k.a. deals)",
}

func init() {
	rootCmd.AddCommand(opportunitiesCmd)
	opportunitiesCmd.AddCommand(oppListCmd())
	opportunitiesCmd.AddCommand(oppGetCmd())
	opportunitiesCmd.AddCommand(oppCreateCmd())
	opportunitiesCmd.AddCommand(oppDeleteCmd())
	opportunitiesCmd.AddCommand(oppStagesCmd())
	opportunitiesCmd.AddCommand(oppMemberAddCmd())
}

func oppListCmd() *cobra.Command {
	var count, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List opportunities",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, total, err := c.ListOpportunities(cmd.Context(), count, offset)
			if err != nil {
				return err
			}
			if outputFormat == "table" {
				fmt.Printf("total: %d (shown: %d)\n", total, len(list))
				// Flatten nested bidCurrency {abbreviation, ...} for readability.
				for _, row := range list {
					if cur, ok := row["bidCurrency"].(map[string]any); ok {
						row["bidCurrency"] = cur["abbreviation"]
					}
				}
			}
			printTable([]string{"id", "title", "bidValue", "bidCurrency", "stageName"}, list)
			return nil
		},
	}
	cmd.Flags().IntVar(&count, "count", 100, "")
	cmd.Flags().IntVar(&offset, "offset", 0, "")
	return cmd
}

func oppGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get OPPORTUNITY_ID",
		Short: "Show an opportunity by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.GetOpportunity(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}

func oppCreateCmd() *cobra.Command {
	var stage int
	var bid float64
	var currency, responsible, desc string
	var contacts []string
	cmd := &cobra.Command{
		Use:     "create TITLE",
		Aliases: []string{"add"},
		Short:   "Create an opportunity",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			if stage == 0 {
				stage = 1
			}
			out, err := c.CreateOpportunity(cmd.Context(), args[0], stage, responsible, currency, desc, bid)
			if err != nil {
				return err
			}
			did := strconv.Itoa(int(flexIDFloat(out["id"])))
			for _, cid := range contacts {
				_, _ = c.AddOpportunityMember(cmd.Context(), did, cid)
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().IntVar(&stage, "stage", 1, "pipeline stage id")
	cmd.Flags().Float64Var(&bid, "bid", 0, "bid value")
	cmd.Flags().StringVar(&currency, "currency", "EUR", "bid currency")
	cmd.Flags().StringVar(&responsible, "responsible", "", "responsible user id")
	cmd.Flags().StringVar(&desc, "description", "", "description")
	cmd.Flags().StringSliceVar(&contacts, "contact", nil, "contact id to add as member (repeatable)")
	return cmd
}

func oppDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete OPPORTUNITY_ID [OPPORTUNITY_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete one or more opportunities",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				out, err := c.DeleteOpportunity(cmd.Context(), id)
				if err != nil {
					return err
				}
				printObject(out)
			}
			return nil
		},
	}
}

func oppStagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stages",
		Short: "List pipeline stages",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.ListDealStages(cmd.Context())
			if err != nil {
				return err
			}
			printTable([]string{"id", "title", "sortOrder", "color"}, out)
			return nil
		},
	}
}

func oppMemberAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "member-add OPPORTUNITY_ID CONTACT_ID",
		Short: "Attach a contact to an opportunity",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.AddOpportunityMember(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}
