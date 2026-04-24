package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// Contacts cover both persons and companies; OnlyOffice exposes them through
// the same `crm/contact/*` endpoint family. `oo persons` and `oo companies`
// are filtered views on top of `oo contacts list`, plus dedicated create
// commands that pick the right library call.

var contactsCmd = &cobra.Command{
	Use:     "contacts",
	Aliases: []string{"contact"},
	Short:   "CRM contacts (persons + companies)",
}

var personsCmd = &cobra.Command{
	Use:     "persons",
	Aliases: []string{"person"},
	Short:   "CRM persons (contacts with isCompany=false)",
}

var companiesCmd = &cobra.Command{
	Use:     "companies",
	Aliases: []string{"company"},
	Short:   "CRM companies (contacts with isCompany=true)",
}

func init() {
	rootCmd.AddCommand(contactsCmd)
	rootCmd.AddCommand(personsCmd)
	rootCmd.AddCommand(companiesCmd)

	contactsCmd.AddCommand(contactsListCmd(nil))
	contactsCmd.AddCommand(contactsGetCmd())
	contactsCmd.AddCommand(contactsDeleteCmd())
	contactsCmd.AddCommand(contactsInfoAddCmd())

	only := true
	personsCmd.AddCommand(contactsListCmd(&only)) // persons only
	personsCmd.AddCommand(personsCreateCmd())
	personsCmd.AddCommand(contactsDeleteCmd())

	onlyCo := false
	companiesCmd.AddCommand(contactsListCmd(&onlyCo)) // companies only
	companiesCmd.AddCommand(companiesCreateCmd())
	companiesCmd.AddCommand(contactsDeleteCmd())
}

// contactsListCmd returns a `list` subcommand.
//
//   - personsOnly == nil        → list all contacts
//   - personsOnly == &true      → keep only persons (isCompany=false)
//   - personsOnly == &false     → keep only companies (isCompany=true)
func contactsListCmd(personsOnly *bool) *cobra.Command {
	var search string
	var count, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, total, err := c.ListContacts(cmd.Context(), count, offset, search)
			if err != nil {
				return err
			}
			filtered := list
			if personsOnly != nil {
				filtered = filtered[:0]
				for _, row := range list {
					isCo, _ := row["isCompany"].(bool)
					if *personsOnly && isCo {
						continue
					}
					if !*personsOnly && !isCo {
						continue
					}
					filtered = append(filtered, row)
				}
			}
			if outputFormat == "table" {
				fmt.Printf("total: %d (shown: %d)\n", total, len(filtered))
			}
			printTable([]string{"id", "displayName", "isCompany", "email", "companyName"}, filtered)
			return nil
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "search filter")
	cmd.Flags().IntVar(&count, "count", 50, "")
	cmd.Flags().IntVar(&offset, "offset", 0, "")
	return cmd
}

func contactsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get CONTACT_ID",
		Short: "Show a contact by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.GetContact(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}

func contactsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete CONTACT_ID [CONTACT_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete one or more contacts",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				out, err := c.DeleteContact(cmd.Context(), id)
				if err != nil {
					return err
				}
				printObject(out)
			}
			return nil
		},
	}
}

func contactsInfoAddCmd() *cobra.Command {
	var infoType, value, category string
	var isPrimary bool
	cmd := &cobra.Command{
		Use:   "info-add CONTACT_ID",
		Short: "Add a contact info entry (email, phone, website, linkedin, …)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if infoType == "" || value == "" {
				return fmt.Errorf("--type and --value are required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.AddContactInfo(cmd.Context(), args[0], infoType, value, category, isPrimary)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&infoType, "type", "", "Email|Phone|Website|LinkedIn|…")
	cmd.Flags().StringVar(&value, "value", "", "value")
	cmd.Flags().StringVar(&category, "category", "Work", "Work|Home|Other")
	cmd.Flags().BoolVar(&isPrimary, "primary", false, "mark as primary")
	return cmd
}

func personsCreateCmd() *cobra.Command {
	var first, last, email, linkedin string
	var companyID int
	var jobTitle, about string
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a person",
		RunE: func(cmd *cobra.Command, args []string) error {
			if first == "" || last == "" {
				return fmt.Errorf("--first and --last are required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.CreatePerson(cmd.Context(), first, last, companyID, jobTitle, about)
			if err != nil {
				return err
			}
			pid := strconv.Itoa(int(flexIDFloat(out["id"])))
			if email != "" {
				_, _ = c.AddContactInfo(cmd.Context(), pid, "Email", email, "Work", true)
			}
			if linkedin != "" {
				_, _ = c.AddContactInfo(cmd.Context(), pid, "LinkedIn", linkedin, "Work", false)
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&first, "first", "", "first name")
	cmd.Flags().StringVar(&last, "last", "", "last name")
	cmd.Flags().IntVar(&companyID, "company-id", 0, "employer company id")
	cmd.Flags().StringVar(&jobTitle, "job-title", "", "")
	cmd.Flags().StringVar(&about, "about", "", "about / bio")
	cmd.Flags().StringVar(&email, "email", "", "primary email (adds ContactInfo)")
	cmd.Flags().StringVar(&linkedin, "linkedin", "", "linkedin url (adds ContactInfo)")
	return cmd
}

func companiesCreateCmd() *cobra.Command {
	var name, email, website string
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a company",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.CreateCompany(cmd.Context(), name)
			if err != nil {
				return err
			}
			cid := strconv.Itoa(int(flexIDFloat(out["id"])))
			if email != "" {
				_, _ = c.AddContactInfo(cmd.Context(), cid, "Email", email, "Work", true)
			}
			if website != "" {
				_, _ = c.AddContactInfo(cmd.Context(), cid, "Website", website, "Work", false)
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "company name")
	cmd.Flags().StringVar(&email, "email", "", "primary email (adds ContactInfo)")
	cmd.Flags().StringVar(&website, "website", "", "website url (adds ContactInfo)")
	return cmd
}
