package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

var mailsCmd = &cobra.Command{
	Use:     "mails",
	Aliases: []string{"mail"},
	Short:   "OnlyOffice Workspace mail — list, read, draft, delete",
}

func init() {
	rootCmd.AddCommand(mailsCmd)
	mailsCmd.AddCommand(mailsAccountsCmd())
	mailsCmd.AddCommand(mailsFoldersCmd())
	mailsCmd.AddCommand(mailsListCmd())
	mailsCmd.AddCommand(mailsGetCmd())
	mailsCmd.AddCommand(mailsDownloadAttachmentCmd())
	mailsCmd.AddCommand(mailsDraftCmd())
	mailsCmd.AddCommand(mailsAttachCmd())
	mailsCmd.AddCommand(mailsDraftInvoiceCmd())
	mailsCmd.AddCommand(mailsSendCmd())
	mailsCmd.AddCommand(mailsDeleteCmd())
}

func mailsAccountsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "accounts",
		Short: "List mailboxes linked to your OnlyOffice account",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			accounts, err := c.ListMailAccounts(cmd.Context())
			if err != nil {
				return err
			}
			if outputFormat == "json" {
				printObject(accounts)
				return nil
			}
			printTable([]string{"mailboxId", "email", "enabled", "isDefault"}, onlyoffice.MailAccountsAsTableRows(accounts))
			return nil
		},
	}
}

func mailsFoldersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "folders",
		Short: "List mail folders (inbox, sent, trash, …)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			folders, err := c.ListMailFolders(cmd.Context())
			if err != nil {
				return err
			}
			if outputFormat == "json" {
				printObject(folders)
				return nil
			}
			printTable([]string{"id", "unread", "total_count", "time_modified"}, onlyoffice.MailFoldersAsTableRows(folders))
			return nil
		},
	}
}

func mailsListCmd() *cobra.Command {
	var folder string
	var limit, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages in a folder (default inbox)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			folderID, err := onlyoffice.ResolveMailFolder(folder)
			if err != nil {
				return err
			}
			msgs, err := c.ListMailMessages(cmd.Context(), onlyoffice.MailMessagesFilter{
				Folder:     folderID,
				Count:      limit,
				StartIndex: offset,
			})
			if err != nil {
				return err
			}
			if outputFormat == "json" {
				printObject(msgs)
				return nil
			}
			printTable([]string{"id", "subject", "fromName", "fromAddress", "date", "folder", "size", "isNew"}, onlyoffice.MailMessagesAsTableRows(msgs))
			return nil
		},
	}
	cmd.Flags().StringVarP(&folder, "folder", "f", "inbox", "folder name (inbox|sent|drafts|trash|spam) or numeric id")
	cmd.Flags().IntVar(&limit, "limit", 50, "max messages to return (paginates past API page size of 25)")
	cmd.Flags().IntVar(&offset, "offset", 0, "skip this many messages before listing")
	return cmd
}

func mailsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get MESSAGE_ID",
		Short: "Read one message by numeric id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.GetMailMessage(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}

func mailsDownloadAttachmentCmd() *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "download-attachment ATTACHMENT_ID",
		Short: "Download a mail attachment by attachment id",
		Long: `Download a raw attachment from OnlyOffice Mail's download.ashx handler.

Example:
  oo mails download-attachment 12345 --out /tmp/attach.bin
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(outPath) == "" {
				return fmt.Errorf("--out is required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			body, err := c.DownloadMailAttachment(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := writeMailAttachment(outPath, body); err != nil {
				return err
			}
			if outputFormat == "json" {
				printObject(map[string]any{
					"attachmentId": args[0],
					"bytes":        len(body),
					"path":         outPath,
				})
				return nil
			}
			fmt.Printf("saved %d bytes to %s\n", len(body), outPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&outPath, "out", "", "output file path")
	return cmd
}

func mailsDraftCmd() *cobra.Command {
	var from, to, cc, bcc, subject, body, html string
	var id int64
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Create or update a mail draft (OnlyOffice Mail)",
		Long: `Save a draft in /addons/mail (PUT /api/2.0/mail/drafts/save).

  oo mails draft --to a@b.com --subject "…" --body "<p>…</p>"
  oo mails draft --id 123 --to a@b.com --subject "…" --html "<p>…</p>"
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" {
				return fmt.Errorf("--to is required")
			}
			htmlBody := body
			if html != "" {
				htmlBody = html
			}
			if htmlBody == "" {
				return fmt.Errorf("--body or --html is required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.SaveMailDraft(cmd.Context(), onlyoffice.SaveMailDraftParams{
				ID:      id,
				From:    from,
				To:      to,
				Cc:      cc,
				Bcc:     bcc,
				Subject: subject,
				Body:    onlyoffice.PlainTextToMailHTML(htmlBody),
			})
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().Int64Var(&id, "id", 0, "existing draft id (0 = create)")
	cmd.Flags().StringVar(&from, "from", "", "from address (default: first enabled mailbox)")
	cmd.Flags().StringVar(&to, "to", "", "recipient (required)")
	cmd.Flags().StringVar(&cc, "cc", "", "cc")
	cmd.Flags().StringVar(&bcc, "bcc", "", "bcc")
	cmd.Flags().StringVar(&subject, "subject", "", "subject")
	cmd.Flags().StringVar(&body, "body", "", "plain text or HTML body")
	cmd.Flags().StringVar(&html, "html", "", "HTML body (alias of --body when set)")
	return cmd
}

func mailsAttachCmd() *cobra.Command {
	var fileID int64
	cmd := &cobra.Command{
		Use:   "attach MESSAGE_ID",
		Short: "Attach an OnlyOffice Files document to a draft/message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if fileID <= 0 {
				return fmt.Errorf("--file-id is required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.AttachMailDocument(cmd.Context(), args[0], fileID)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
	cmd.Flags().Int64Var(&fileID, "file-id", 0, "OnlyOffice file id (e.g. invoice PDF)")
	return cmd
}

func mailsDraftInvoiceCmd() *cobra.Command {
	var to, from, subject, body string
	var invoiceID int
	cmd := &cobra.Command{
		Use:   "draft-invoice",
		Short: "Create a mail draft with regenerated invoice PDF attached",
		Long: `Regenerate the CRM invoice PDF, save an OnlyOffice Mail draft, and attach the PDF.

Does not send. Open /addons/mail/#drafts to review.

  oo mails draft-invoice --invoice 16 --to info@example.com
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if invoiceID <= 0 || to == "" {
				return fmt.Errorf("--invoice and --to are required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			pdf, err := c.ForceRegenerateInvoicePDF(cmd.Context(), strconv.Itoa(invoiceID))
			if err != nil {
				return err
			}
			fileID := onlyoffice.Int64FromMap(pdf, "id")
			if fileID <= 0 {
				return fmt.Errorf("invoice PDF has no file id: %+v", pdf)
			}
			inv, err := c.GetInvoice(cmd.Context(), strconv.Itoa(invoiceID))
			if err != nil {
				return err
			}
			number := strings.TrimSpace(fmt.Sprint(inv["number"]))
			cost := formatInvoiceCostEUR(inv["cost"])
			if subject == "" {
				subject = fmt.Sprintf("Invoice %s (%s EUR)", number, cost)
			}
			if body == "" {
				body = fmt.Sprintf(`Hello,

please find attached invoice %s for %s EUR.

Payment terms: see invoice notes.

Best regards`, number, cost)
			}
			draft, err := c.SaveMailDraft(cmd.Context(), onlyoffice.SaveMailDraftParams{
				From:    from,
				To:      to,
				Subject: subject,
				Body:    onlyoffice.MailHTMLWithBlankParagraphs(onlyoffice.PlainTextToMailHTML(body)),
			})
			if err != nil {
				return err
			}
			mid := fmt.Sprint(draft["id"])
			att, err := c.AttachMailDocument(cmd.Context(), mid, fileID)
			if err != nil {
				return fmt.Errorf("draft %s created but attach failed: %w", mid, err)
			}
			if outputFormat == "json" {
				printObject(map[string]any{"draft": draft, "attachment": att, "pdfFileId": fileID})
				return nil
			}
			fmt.Printf("draft %s  to=%s  subject=%q  pdfFileId=%d\n", mid, to, subject, fileID)
			fmt.Printf("open: %s/addons/mail/#drafts\n", strings.TrimRight(os.Getenv("ONLYOFFICE_URL"), "/"))
			return nil
		},
	}
	cmd.Flags().IntVar(&invoiceID, "invoice", 0, "CRM invoice id")
	cmd.Flags().StringVar(&to, "to", "", "recipient")
	cmd.Flags().StringVar(&from, "from", "", "from address (default mailbox)")
	cmd.Flags().StringVar(&subject, "subject", "", "override subject")
	cmd.Flags().StringVar(&body, "body", "", "override plain-text body")
	return cmd
}

func formatInvoiceCostEUR(v any) string {
	s := strings.TrimSpace(fmt.Sprint(v))
	s = strings.TrimSuffix(s, ".00")
	s = strings.TrimSuffix(s, ".0")
	if s == "" || s == "<nil>" {
		return "?"
	}
	return s
}

func writeMailAttachment(path string, body []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("attachment output path is required")
	}
	return os.WriteFile(path, body, 0o644)
}

func mailsSendCmd() *cobra.Command {
	var from, to, cc, bcc, subject, body, html string
	var id int64
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a mail message (OnlyOffice Mail)",
		Long: `Send via PUT /api/2.0/mail/messages/send.json.

  oo mails send --id 7803 --body "…"            # send referencing a draft id
  oo mails send --to a@b.com --subject "…" --body "…"
  oo mails send --id 7803 --to a@b.com --subject "…" --body "…" --cc x@y.com

IMPORTANT: send.json does NOT copy subject/body from the referenced draft — the
content must be in this request (--subject/--body). Cc/Bcc are omitted when empty
(the API 400s on empty strings). The API send does not append the UI signature —
put the chat line in --body if needed.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == "" && id == 0 {
				return fmt.Errorf("--to is required (or --id of an existing draft)")
			}
			htmlBody := body
			if html != "" {
				htmlBody = html
			}
			if htmlBody == "" && id != 0 {
				// The send.json endpoint does NOT copy subject/body from the
				// referenced draft — an empty body here sends an empty message.
				// Warn instead of silently mailing an empty email.
				return fmt.Errorf("--body/--html is required when sending by --id (send.json needs the content in the request)")
			}
			if htmlBody == "" && to == "" {
				return fmt.Errorf("--body is required for a fresh message")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			raw, err := c.SendMail(cmd.Context(), onlyoffice.SendMailParams{
				ID:      id,
				From:    from,
				To:      to,
				Cc:      cc,
				Bcc:     bcc,
				Subject: subject,
				Body:    htmlBody,
			})
			if err != nil {
				return err
			}
			fmt.Println(string(raw))
			return nil
		},
	}
	cmd.Flags().Int64Var(&id, "id", 0, "existing draft id to send (0 = fresh message)")
	cmd.Flags().StringVar(&from, "from", "", "from address (default: first enabled mailbox)")
	cmd.Flags().StringVar(&to, "to", "", "recipient (required unless --id)")
	cmd.Flags().StringVar(&cc, "cc", "", "cc")
	cmd.Flags().StringVar(&bcc, "bcc", "", "bcc")
	cmd.Flags().StringVar(&subject, "subject", "", "subject")
	cmd.Flags().StringVar(&body, "body", "", "plain text or HTML body")
	cmd.Flags().StringVar(&html, "html", "", "HTML body (alias of --body when set)")
	return cmd
}

func mailsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete ID [ID...]",
		Aliases: []string{"rm"},
		Short:   "Remove messages from the mailbox (moves to trash or deletes permanently per server rules)",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ids := make([]int, 0, len(args))
			for _, a := range args {
				id, err := strconv.Atoi(strings.TrimSpace(a))
				if err != nil {
					return err
				}
				ids = append(ids, id)
			}
			out, err := c.RemoveMailMessages(cmd.Context(), ids...)
			if err != nil {
				return err
			}
			printObject(out)
			return nil
		},
	}
}
