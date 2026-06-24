package main

import (
	"strconv"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

var mailsCmd = &cobra.Command{
	Use:     "mails",
	Aliases: []string{"mail"},
	Short:   "OnlyOffice Workspace mail — list, read, delete",
}

func init() {
	rootCmd.AddCommand(mailsCmd)
	mailsCmd.AddCommand(mailsAccountsCmd())
	mailsCmd.AddCommand(mailsFoldersCmd())
	mailsCmd.AddCommand(mailsListCmd())
	mailsCmd.AddCommand(mailsGetCmd())
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
