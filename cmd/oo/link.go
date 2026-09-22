package main

import (
	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(linkCmd())
}

func linkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link FILE_ID [FILE_ID...]",
		Short: "Print OnlyOffice DocEditor deep links (Products/Files/DocEditor.aspx?fileid=…)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				printObject(map[string]any{
					"fileid": id,
					"title":  fileTitle(cmd, c, id),
					"url":    c.FileEditorURL(id),
				})
			}
			return nil
		},
	}
}

// fileTitle best-effort resolves a file title; never fails the command.
func fileTitle(cmd *cobra.Command, c *onlyoffice.Client, id string) string {
	f, err := c.GetFile(cmd.Context(), id)
	if err != nil || f == nil || f.Title == nil {
		return ""
	}
	return *f.Title
}
