package main

import (
	"fmt"
	"strconv"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/catalog"
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
	contactsCmd.AddCommand(contactsMergeCmd())
	contactsCmd.AddCommand(contactsInfoAddCmd())
	contactsCmd.AddCommand(contactsDedupeInfoCmd())

	only := true
	personsCmd.AddCommand(contactsListCmd(&only)) // persons only
	personsCmd.AddCommand(personsCreateCmd())
	personsCmd.AddCommand(personsFixNamesCmd())
	personsCmd.AddCommand(contactsDeleteCmd())
	personsCmd.AddCommand(personsDedupeCmd())

	onlyCo := false
	companiesCmd.AddCommand(contactsListCmd(&onlyCo)) // companies only
	companiesCmd.AddCommand(companiesCreateCmd())
	companiesCmd.AddCommand(contactsDeleteCmd())
	companiesCmd.AddCommand(companiesDedupeCmd())
	companiesCmd.AddCommand(companiesDedupePersonsCmd())
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

func contactsMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge FROM_ID INTO_ID",
		Short: "Merge FROM contact into INTO (FROM is removed)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.MergeContacts(cmd.Context(), args[1], args[0])
			if err != nil {
				return err
			}
			printObject(out)
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
	var first, last, email, linkedin, phone string
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
			if phone != "" {
				_, _ = c.AddContactInfo(cmd.Context(), pid, "Phone", phone, "Work", true)
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
	cmd.Flags().StringVar(&phone, "phone", "", "primary phone (adds ContactInfo)")
	cmd.Flags().StringVar(&linkedin, "linkedin", "", "linkedin url (adds ContactInfo)")
	return cmd
}

func personsFixNamesCmd() *cobra.Command {
	var companyID int
	var dryRun bool
	var orgHint string
	cmd := &cobra.Command{
		Use:   "fix-names",
		Short: "Strip company annotations from last names; keep companyId link",
		Long: `Repairs CRM persons whose lastName embeds a company ("Thomsen (Acme)")
or whose firstName is an email. Company belongs on companyId, not in the name.

With --company-id, only persons linked to that company are scanned.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			var list []map[string]any
			if companyID > 0 {
				list, err = c.ListCompanyPersons(ctx, strconv.Itoa(companyID))
			} else {
				all, err2 := c.ListAllContacts(ctx)
				err = err2
				for _, row := range all {
					if isCo, _ := row["isCompany"].(bool); !isCo {
						list = append(list, row)
					}
				}
			}
			if err != nil {
				return err
			}
			fixed, skipped := 0, 0
			for _, p := range list {
				id := fmt.Sprint(p["id"])
				// List payloads often omit contactInfos; refresh when name looks like email.
				fn := strings.TrimSpace(fmt.Sprint(p["firstName"]))
				if strings.Contains(fn, "@") || fn == "" {
					if full, gerr := c.GetContact(ctx, id); gerr == nil && full != nil {
						p = full
					}
				}
				fn = strings.TrimSpace(fmt.Sprint(p["firstName"]))
				ln := strings.TrimSpace(fmt.Sprint(p["lastName"]))
				dn := strings.TrimSpace(fmt.Sprint(p["displayName"]))
				org := orgHint
				if org == "" {
					if co, ok := p["company"].(map[string]any); ok {
						org = strings.TrimSpace(fmt.Sprint(co["displayName"]))
					}
				}
				emails := contactEmailsFromMap(p)
				for _, row := range onlyoffice.ContactInfoRows(p) {
					if onlyoffice.NormalizeContactInfoType(fmt.Sprint(row["infoType"])) == "email" {
						if data := strings.TrimSpace(fmt.Sprint(row["data"])); data != "" {
							emails = append(emails, data)
						}
					}
				}
				cf, cl := catalog.CleanPersonNames(fn, ln, dn, org, emails)
				needName := cf != fn || cl != ln
				linkID := companyID
				if linkID == 0 {
					if co, ok := p["company"].(map[string]any); ok {
						linkID = int(flexIDFloat(co["id"]))
					}
				}
				if !needName {
					skipped++
					continue
				}
				row := map[string]any{
					"id": id, "from": fn + " / " + ln, "to": cf + " / " + cl, "companyId": linkID,
				}
				if dryRun {
					printObject(row)
					fixed++
					continue
				}
				if _, err := c.UpdatePerson(ctx, id, cf, cl, linkID, "", ""); err != nil {
					return fmt.Errorf("update %s: %w", id, err)
				}
				printObject(row)
				fixed++
			}
			printObject(map[string]any{"fixed": fixed, "skipped": skipped, "dry_run": dryRun})
			return nil
		},
	}
	cmd.Flags().IntVar(&companyID, "company-id", 0, "limit to persons of this company")
	cmd.Flags().StringVar(&orgHint, "org", "", "org name hint for stripping suffixes")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print planned renames only")
	return cmd
}

func contactEmailsFromMap(p map[string]any) []string {
	var out []string
	for _, k := range []string{"email", "primaryEmail"} {
		if v := strings.TrimSpace(fmt.Sprint(p[k])); v != "" && v != "<nil>" {
			out = append(out, v)
		}
	}
	return out
}

func companiesCreateCmd() *cobra.Command {
	var name, email, website, phone, about, street, city, state, zip, country string
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
			if about != "" || street != "" {
				aboutText := about
				if street != "" {
					addr := strings.TrimSpace(strings.Join([]string{street, zip, city, state, country}, ", "))
					if aboutText != "" {
						aboutText = aboutText + "\n" + addr
					} else {
						aboutText = "Billing address: " + addr
					}
				}
				if updated, err := c.UpdateCompany(cmd.Context(), cid, name, aboutText); err == nil {
					out = updated
				}
			}
			if email != "" {
				_, _ = c.AddContactInfo(cmd.Context(), cid, "Email", email, "Work", true)
			}
			if website != "" {
				_, _ = c.AddContactInfo(cmd.Context(), cid, "Website", website, "Work", false)
			}
			if phone != "" {
				_, _ = c.AddContactInfo(cmd.Context(), cid, "Phone", phone, "Work", true)
			}
			if street != "" {
				if _, err := c.AddContactAddress(cmd.Context(), cid, street, city, state, zip, country, "Billing", true); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: address API failed (stored in about): %v\n", err)
				}
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "company name")
	cmd.Flags().StringVar(&email, "email", "", "primary email (adds ContactInfo)")
	cmd.Flags().StringVar(&website, "website", "", "website url (adds ContactInfo)")
	cmd.Flags().StringVar(&phone, "phone", "", "primary phone (adds ContactInfo)")
	cmd.Flags().StringVar(&about, "about", "", "about / notes")
	cmd.Flags().StringVar(&street, "street", "", "billing street")
	cmd.Flags().StringVar(&city, "city", "", "billing city")
	cmd.Flags().StringVar(&state, "state", "", "billing state")
	cmd.Flags().StringVar(&zip, "zip", "", "billing zip")
	cmd.Flags().StringVar(&country, "country", "", "billing country")
	return cmd
}

func companiesDedupeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dedupe",
		Short: "Merge duplicate companies by normalized name",
		RunE: dedupeRunE(func(cmd *cobra.Command, c *onlyoffice.Client) error {
			res, err := onlyoffice.DedupeCompanies(cmd.Context(), c)
			if err != nil {
				return err
			}
			printObject(res)
			return nil
		}),
	}
}

func companiesDedupePersonsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dedupe-persons",
		Short: "Merge duplicate persons under each company",
		RunE: dedupeRunE(func(cmd *cobra.Command, c *onlyoffice.Client) error {
			res, err := onlyoffice.DedupeCompanyPersons(cmd.Context(), c)
			if err != nil {
				return err
			}
			printObject(res)
			return nil
		}),
	}
}

func personsDedupeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dedupe",
		Short: "Merge duplicate persons by normalized first+last name",
		RunE: dedupeRunE(func(cmd *cobra.Command, c *onlyoffice.Client) error {
			res, err := onlyoffice.DedupePersons(cmd.Context(), c)
			if err != nil {
				return err
			}
			printObject(res)
			return nil
		}),
	}
}

func contactsDedupeInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dedupe-info",
		Short: "Remove duplicate contact info rows (email, phone, …)",
		RunE: dedupeRunE(func(cmd *cobra.Command, c *onlyoffice.Client) error {
			res, err := onlyoffice.DedupeContactInfo(cmd.Context(), c)
			if err != nil {
				return err
			}
			printObject(res)
			return nil
		}),
	}
}
