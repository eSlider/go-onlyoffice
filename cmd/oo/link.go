package main

import (
	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(linkCmd())
}

// linkCmd prints deep links for file ids. Used to build the third-party
// document packs (e.g. the internal "example-city apartment" Info cover embeds links to
// passports, contracts and the income documents). File ids come from
// `oo projects files list` / `oo dav ls`; `oo projects files replace-in` keeps
// them clean when a document is re-uploaded.
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
