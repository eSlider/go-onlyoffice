package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eslider/go-onlyoffice/catalog"
	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Inventory clients/contacts from disk; match and apply to OO CRM",
	Long: `Build a YAML catalog of persons/companies from contacts folders and project
trees, match against live OnlyOffice CRM, then create/update only rows with
approve: true.

Typical flow:
  oo catalog scan-contacts --root PATH -O /tmp/contacts.yaml
  oo catalog scan-projects --root ~/projects -O /tmp/projects.yaml
  oo catalog scan-thunderbird --root PATH -O /tmp/thunderbird.yaml
  oo catalog merge -i /tmp/contacts.yaml -i /tmp/projects.yaml -i /tmp/thunderbird.yaml -O docs/catalog/clients-contacts.yaml
  oo catalog match -i docs/catalog/clients-contacts.yaml
  # edit approve: true on pilot rows
  oo catalog apply --dry-run -i docs/catalog/clients-contacts.yaml
  oo catalog apply --apply -i docs/catalog/clients-contacts.yaml
`}

func init() {
	rootCmd.AddCommand(catalogCmd)
	catalogCmd.AddCommand(catalogScanContactsCmd())
	catalogCmd.AddCommand(catalogScanProjectsCmd())
	catalogCmd.AddCommand(catalogScanThunderbirdCmd())
	catalogCmd.AddCommand(catalogMergeCmd())
	catalogCmd.AddCommand(catalogMatchCmd())
	catalogCmd.AddCommand(catalogApplyCmd())
}

func catalogScanContactsCmd() *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "scan-contacts",
		Short: "Parse VCF + folder stubs + email txt under a contacts root",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := cmd.Flags().GetString("root")
			if root == "" {
				return fmt.Errorf("--root is required")
			}
			doc, err := catalog.ScanContactsRoot(root)
			if err != nil {
				return err
			}
			return writeCatalog(cmd, doc, outPath)
		},
	}
	cmd.Flags().String("root", "", "contacts directory (local path)")
	cmd.Flags().StringVarP(&outPath, "out", "O", "", "write YAML to this path (default: stdout summary)")
	_ = cmd.MarkFlagRequired("root")
	return cmd
}

func catalogScanProjectsCmd() *cobra.Command {
	var outPath string
	var maxDepth int
	cmd := &cobra.Command{
		Use:   "scan-projects",
		Short: "Git roots / remotes / top-level dirs → company rows",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := cmd.Flags().GetString("root")
			if root == "" {
				return fmt.Errorf("--root is required")
			}
			doc, err := catalog.ScanProjectsRoot(root, maxDepth)
			if err != nil {
				return err
			}
			return writeCatalog(cmd, doc, outPath)
		},
	}
	cmd.Flags().String("root", "", "projects directory (local path)")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 4, "max directory depth for git roots")
	cmd.Flags().StringVarP(&outPath, "out", "O", "", "write YAML to this path")
	_ = cmd.MarkFlagRequired("root")
	return cmd
}

func catalogScanThunderbirdCmd() *cobra.Command {
	var outPath string
	cmd := &cobra.Command{
		Use:   "scan-thunderbird",
		Short: "Thunderbird profiles: abook/history.mab + Gloda SQLite contacts",
		Long: `Walk a directory tree for Thunderbird profiles and extract person rows from:
  - *.mab address books (email regex)
  - global-messages-db.sqlite Gloda contacts/identities

Noisy senders (noreply, Amazon marketplace, GitHub reply, …) are skipped.
Default zone is private; known work domains (e.g. wheregroup.com) get zone=warm role=work.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := cmd.Flags().GetString("root")
			if root == "" {
				return fmt.Errorf("--root is required")
			}
			doc, err := catalog.ScanThunderbirdRoot(root)
			if err != nil {
				return err
			}
			return writeCatalog(cmd, doc, outPath)
		},
	}
	cmd.Flags().String("root", "", "Thunderbird profile or parent directory (local path)")
	cmd.Flags().StringVarP(&outPath, "out", "O", "", "write YAML to this path")
	_ = cmd.MarkFlagRequired("root")
	return cmd
}

func catalogMergeCmd() *cobra.Command {
	var inputs []string
	var outPath string
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Union + normalize catalog YAML files by entry id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(inputs) == 0 {
				return fmt.Errorf("at least one -i/--input required")
			}
			var docs []*catalog.Document
			for _, p := range inputs {
				d, err := catalog.LoadYAML(p)
				if err != nil {
					return err
				}
				docs = append(docs, d)
			}
			merged := catalog.MergeDocs(docs...)
			if outPath == "" {
				return fmt.Errorf("--out is required for merge")
			}
			if err := ensureParent(outPath); err != nil {
				return err
			}
			if err := catalog.SaveYAML(outPath, merged); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "merged %d entries → %s\n", len(merged.Entries), outPath)
			return nil
		},
	}
	cmd.Flags().StringArrayVarP(&inputs, "input", "i", nil, "input catalog YAML (repeatable)")
	cmd.Flags().StringVarP(&outPath, "out", "O", "", "output catalog YAML")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}

func catalogMatchCmd() *cobra.Command {
	var inPath, outPath string
	cmd := &cobra.Command{
		Use:   "match",
		Short: "Diff catalog vs live OO; set status new|exists|conflict and oo_id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inPath == "" {
				return fmt.Errorf("--input is required")
			}
			doc, err := catalog.LoadYAML(inPath)
			if err != nil {
				return err
			}
			client, err := newOO(cmd)
			if err != nil {
				return err
			}
			if err := catalog.MatchAgainstOO(cmd.Context(), client, doc); err != nil {
				return err
			}
			if outPath == "" {
				outPath = inPath
			}
			if err := catalog.SaveYAML(outPath, doc); err != nil {
				return err
			}
			counts := map[string]int{}
			for _, e := range doc.Entries {
				counts[e.Status]++
			}
			fmt.Fprintf(cmd.OutOrStdout(), "matched → %s  new=%d exists=%d conflict=%d\n",
				outPath, counts["new"], counts["exists"], counts["conflict"])
			return nil
		},
	}
	cmd.Flags().StringVarP(&inPath, "input", "i", "", "catalog YAML")
	cmd.Flags().StringVarP(&outPath, "out", "O", "", "output path (default: overwrite input)")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func catalogApplyCmd() *cobra.Command {
	var inPath, outPath string
	var dryRun, doApply bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Create/update OO contacts for approve:true rows",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun && !doApply {
				return fmt.Errorf("pass --dry-run or --apply")
			}
			if dryRun && doApply {
				return fmt.Errorf("use either --dry-run or --apply, not both")
			}
			if inPath == "" {
				return fmt.Errorf("--input is required")
			}
			doc, err := catalog.LoadYAML(inPath)
			if err != nil {
				return err
			}
			client, err := newOO(cmd)
			if err != nil {
				return err
			}
			res, err := catalog.ApplyApproved(cmd.Context(), client, doc, dryRun)
			if err != nil {
				return err
			}
			if !dryRun {
				if outPath == "" {
					outPath = inPath
				}
				if err := catalog.SaveYAML(outPath, doc); err != nil {
					return err
				}
			}
			if outputFormat == "json" {
				printJSON(res)
				return nil
			}
			mode := "apply"
			if dryRun {
				mode = "dry-run"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: created=%d updated=%d skipped=%d errors=%d\n",
				mode, res.Created, res.Updated, res.Skipped, len(res.Errors))
			for _, e := range res.Errors {
				fmt.Fprintf(cmd.ErrOrStderr(), "  error: %s\n", e)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&inPath, "input", "i", "", "catalog YAML")
	cmd.Flags().StringVarP(&outPath, "out", "O", "", "write updated catalog after apply (default: overwrite input)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "count actions without writing to OO")
	cmd.Flags().BoolVar(&doApply, "apply", false, "perform OO creates/updates")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func writeCatalog(cmd *cobra.Command, doc *catalog.Document, outPath string) error {
	if outPath != "" {
		if err := ensureParent(outPath); err != nil {
			return err
		}
		if err := catalog.SaveYAML(outPath, doc); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %d entries → %s\n", len(doc.Entries), outPath)
		return nil
	}
	if outputFormat == "json" {
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}
	persons, companies := 0, 0
	for _, e := range doc.Entries {
		if e.Kind == "company" {
			companies++
		} else {
			persons++
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "entries=%d persons=%d companies=%d (pass -O PATH to write YAML)\n",
		len(doc.Entries), persons, companies)
	return nil
}

func ensureParent(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
