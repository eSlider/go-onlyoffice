package main

import (
	"fmt"
	"strings"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

var invoicesCmd = &cobra.Command{
	Use:     "invoices",
	Aliases: []string{"invoice"},
	Short:   "CRM invoices (billing)",
}

func init() {
	rootCmd.AddCommand(invoicesCmd)
	invoicesCmd.AddCommand(invoiceListCmd())
	invoicesCmd.AddCommand(invoiceGetCmd())
	invoicesCmd.AddCommand(invoiceCreateCmd())
	invoicesCmd.AddCommand(invoiceDeleteCmd())
	invoicesCmd.AddCommand(invoiceItemsCmd())
}

func invoiceListCmd() *cobra.Command {
	var count, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List CRM invoices",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, total, err := c.ListInvoices(cmd.Context(), count, offset)
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(list))
			for _, inv := range list {
				rows = append(rows, onlyoffice.FlattenInvoiceRow(inv))
			}
			if outputFormat == "table" {
				fmt.Printf("total: %d (shown: %d)\n", total, len(rows))
			}
			printTable([]string{"id", "number", "cost", "statusTitle", "contactName", "issueDate", "dueDate"}, rows)
			return nil
		},
	}
	cmd.Flags().IntVar(&count, "count", 50, "")
	cmd.Flags().IntVar(&offset, "offset", 0, "")
	return cmd
}

func invoiceGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get INVOICE_ID",
		Short: "Show an invoice by id (incl. lines)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.GetInvoice(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}

func invoiceCreateCmd() *cobra.Command {
	var (
		number, issueDate, dueDate, language, currency, terms, description, po string
		contactID, itemID                                                      int64
		price, qty                                                             float64
		lineDesc                                                               string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a draft invoice with one line",
		Long: `Create a CRM invoice (Draft) with a single line.

Example:
  oo invoices create --number INV-2026-01 --contact 123 --item 10 \
    --price 300 --description "Service package" \
    --terms "…payment terms…"
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if number == "" || contactID == 0 || itemID == 0 {
				return fmt.Errorf("--number, --contact and --item are required")
			}
			if issueDate == "" {
				issueDate = time.Now().Format("2006-01-02") + "T00:00:00.0000000+01:00"
			} else if !strings.Contains(issueDate, "T") {
				issueDate = issueDate + "T00:00:00.0000000+01:00"
			}
			if dueDate == "" {
				dueDate = time.Now().Add(14 * 24 * time.Hour).Format("2006-01-02") + "T00:00:00.0000000+01:00"
			} else if !strings.Contains(dueDate, "T") {
				dueDate = dueDate + "T00:00:00.0000000+01:00"
			}
			if qty == 0 {
				qty = 1
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.CreateInvoice(cmd.Context(), onlyoffice.CreateInvoiceParams{
				Number:              number,
				IssueDate:           issueDate,
				DueDate:             dueDate,
				ContactID:           contactID,
				Language:            language,
				Currency:            currency,
				ExchangeRate:        1,
				PurchaseOrderNumber: po,
				Terms:               terms,
				Description:         description,
				Lines: []onlyoffice.InvoiceLine{{
					InvoiceItemID: itemID,
					Description:   lineDesc,
					Quantity:      qty,
					Price:         price,
					SortOrder:     0,
				}},
			})
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&number, "number", "", "invoice number (e.g. INV-2026-01)")
	cmd.Flags().Int64Var(&contactID, "contact", 0, "bill-to contact id (company preferred)")
	cmd.Flags().Int64Var(&itemID, "item", 0, "catalog invoice item id")
	cmd.Flags().Float64Var(&price, "price", 0, "line price")
	cmd.Flags().Float64Var(&qty, "qty", 1, "line quantity")
	cmd.Flags().StringVar(&lineDesc, "line-description", "", "line description override")
	cmd.Flags().StringVar(&issueDate, "issue-date", "", "YYYY-MM-DD (default today)")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "YYYY-MM-DD (default +14d)")
	cmd.Flags().StringVar(&language, "language", "de-DE", "invoice language")
	cmd.Flags().StringVar(&currency, "currency", "EUR", "currency abbreviation")
	cmd.Flags().StringVar(&terms, "terms", "", "payment terms / footer")
	cmd.Flags().StringVar(&description, "description", "", "invoice description")
	cmd.Flags().StringVar(&po, "po", "", "purchase order number")
	return cmd
}

func invoiceDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete INVOICE_ID [INVOICE_ID...]",
		Short: "Delete one or more invoices",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				if _, err := c.DeleteInvoice(cmd.Context(), id); err != nil {
					return err
				}
				fmt.Println("deleted", id)
			}
			return nil
		},
	}
}

func invoiceItemsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "items",
		Aliases: []string{"item"},
		Short:   "Invoice catalog items",
	}
	cmd.AddCommand(invoiceItemsListCmd())
	cmd.AddCommand(invoiceItemsCreateCmd())
	cmd.AddCommand(invoiceItemsDeleteCmd())
	return cmd
}

func invoiceItemsListCmd() *cobra.Command {
	var count, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List catalog invoice items",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			list, total, err := c.ListInvoiceItems(cmd.Context(), count, offset)
			if err != nil {
				return err
			}
			if outputFormat == "table" {
				fmt.Printf("total: %d (shown: %d)\n", total, len(list))
				for _, row := range list {
					if cur, ok := row["currency"].(map[string]any); ok {
						row["currency"] = cur["abbreviation"]
					}
				}
			}
			printTable([]string{"id", "title", "price", "currency", "description"}, list)
			return nil
		},
	}
	cmd.Flags().IntVar(&count, "count", 50, "")
	cmd.Flags().IntVar(&offset, "offset", 0, "")
	return cmd
}

func invoiceItemsCreateCmd() *cobra.Command {
	var title, description, currency string
	var price float64
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a catalog invoice item",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.CreateInvoiceItem(cmd.Context(), title, description, price, currency)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "item title")
	cmd.Flags().StringVar(&description, "description", "", "item description")
	cmd.Flags().Float64Var(&price, "price", 0, "unit price")
	cmd.Flags().StringVar(&currency, "currency", "EUR", "currency")
	return cmd
}

func invoiceItemsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete ITEM_ID [ITEM_ID...]",
		Short: "Delete catalog invoice items",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				if _, err := c.DeleteInvoiceItem(cmd.Context(), id); err != nil {
					return err
				}
				fmt.Println("deleted", id)
			}
			return nil
		},
	}
}
