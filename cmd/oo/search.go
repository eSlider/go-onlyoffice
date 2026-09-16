package main

import (
	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/eslider/go-onlyoffice/cmd/internal/bootstrap"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(searchCmd())
}

// searchCmd queries the OnlyOffice document index through the file facade. The
// REST /api/2.0/files/@search endpoint only searches file names in the database;
// content search needs Elasticsearch (see docs/elasticsearch.md).
func searchCmd() *cobra.Command {
	var (
		content bool
		folder  string
		limit   int
		asJSON  bool
	)
	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Full-text search over documents by name, optionally by content (Elasticsearch)",
		Long: "Search the OnlyOffice Documents index.\n\n" +
			"By default only file names are matched. With --content the query also\n" +
			"matches extracted document text (document.attachment.content); this covers\n" +
			"Office formats (docx/xlsx/pptx) and is slower.\n\n" +
			"Requires ONLYOFFICE_ES_URL (and optionally ONLYOFFICE_ES_INDEX,\n" +
			"ONLYOFFICE_TENANT). See docs/elasticsearch.md for the tunnel setup.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				outputFormat = "json"
			}
			bootstrap.LoadEnv()
			c := onlyoffice.NewClient(onlyoffice.GetEnvironmentCredentials())
			searcher, err := c.Files().Search()
			if err != nil {
				return err
			}
			hits, err := searcher.Search(cmd.Context(), onlyoffice.SearchQuery{
				Text:      args[0],
				InContent: content,
				FolderID:  folder,
				Limit:     limit,
			})
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(hits))
			for _, h := range hits {
				rows = append(rows, map[string]any{
					"id":        h.ID,
					"title":     h.Title,
					"folder":    h.ParentID,
					"score":     h.Score,
					"highlight": h.Highlight,
				})
			}
			if outputFormat == "json" {
				printJSON(rows)
				return nil
			}
			printTable([]string{"id", "title", "folder", "score", "highlight"}, rows)
			return nil
		},
	}
	cmd.Flags().BoolVar(&content, "content", false, "also match extracted document content")
	cmd.Flags().StringVar(&folder, "folder", "", "limit to a Documents folder id")
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum number of results")
	cmd.Flags().BoolVar(&asJSON, "json", false, "shorthand for --output json")
	return cmd
}
