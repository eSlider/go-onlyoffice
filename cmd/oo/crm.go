package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
				cid := fmt.Sprint(int(flexIDFloat(out["id"])))
				if email != "" {
					_, _ = c.AddContactInfo(cmd.Context(), cid, "Email", email, "Work", true)
				}
				if website != "" {
					_, _ = c.AddContactInfo(cmd.Context(), cid, "Website", website, "Work", false)
				}
				printJSON(out)
				return nil
			}
			if personFirst != "" && personLast != "" {
				out, err := c.CreatePerson(cmd.Context(), personFirst, personLast, companyID, "", "")
				if err != nil {
					return err
				}
				pid := fmt.Sprint(int(flexIDFloat(out["id"])))
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
			did := fmt.Sprint(int(flexIDFloat(out["id"])))
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
