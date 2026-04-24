package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var casesCmd = &cobra.Command{
	Use:     "cases",
	Aliases: []string{"case"},
	Short:   "CRM cases",
}

func init() {
	rootCmd.AddCommand(casesCmd)
	casesCmd.AddCommand(casesListCmd())
	casesCmd.AddCommand(casesCreateCmd())
	casesCmd.AddCommand(casesDeleteCmd())
	casesCmd.AddCommand(casesMemberAddCmd())
}

func casesListCmd() *cobra.Command {
	var count, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List CRM cases",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, total, err := c.ListCases(cmd.Context(), count, offset)
			if err != nil {
				return err
			}
			if outputFormat == "table" {
				fmt.Printf("total: %d (shown: %d)\n", total, len(list))
			}
			printTable([]string{"id", "title", "isClosed", "created"}, list)
			return nil
		},
	}
	cmd.Flags().IntVar(&count, "count", 50, "")
	cmd.Flags().IntVar(&offset, "offset", 0, "")
	return cmd
}

func casesCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "create TITLE",
		Aliases: []string{"add"},
		Short:   "Create a case",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.CreateCase(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}

func casesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete CASE_ID [CASE_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete one or more cases",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				out, err := c.DeleteCase(cmd.Context(), id)
				if err != nil {
					return err
				}
				printObject(out)
			}
			return nil
		},
	}
}

func casesMemberAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "member-add CASE_ID CONTACT_ID",
		Short: "Attach a contact to a case",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.AddCaseMember(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}
