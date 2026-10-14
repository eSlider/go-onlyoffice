package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/eslider/go-onlyoffice/internal/docx"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(docxCmd())
}

func docxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docx",
		Short: "Edit a local .docx: paragraphs, comments, anchors, highlights",
		Long: `Surgical WordprocessingML edits that leave every untouched part of the
package byte for byte as it was.

Every edit op needs a locator. "find" is a substring that must match exactly one
paragraph: ambiguity is an error, never a guess. Comments can be anchored to a
substring of that paragraph (--highlight marks it yellow) or to the whole
paragraph.

  oo docx show    FILE --find "..."
  oo docx text    FILE --find "..." --old "..." --to "..."
  oo docx set     FILE --find "..." --to "replacement paragraph"
  oo docx comment FILE --find "..." --highlight "..." --author AA --text "..."
  oo docx reply   FILE --id 3 --author AA --text "..."
  oo docx apply   FILE --plan plan.json      (all ops in one open and save)

The comments part is written as word/comments.xml. If the file already carries
the OnlyOffice name word/commentsDocument.xml, that one is reused and the
companion parts are kept in step so the portal does not drop the comment.`,
	}
	cmd.AddCommand(docxShowCmd())
	cmd.AddCommand(docxTextCmd())
	cmd.AddCommand(docxSetCmd())
	cmd.AddCommand(docxCommentCmd())
	cmd.AddCommand(docxReplyCmd())
	cmd.AddCommand(docxListCommentsCmd())
	cmd.AddCommand(docxApplyCmd())
	cmd.AddCommand(docxCommentsRebuildCmd())
	return cmd
}

// docxSpec is the comment spec, in YAML or JSON. Both are accepted on import
// and YAML is what --export writes, so the file round-trips through yq.
type docxSpec struct {
	Threads []docxThreadSpec `json:"threads" yaml:"threads"`
}

type docxThreadSpec struct {
	Messages []docxMessageSpec `json:"messages" yaml:"messages"`
}

type docxMessageSpec struct {
	Author string `json:"author" yaml:"author"`
	Text   string `json:"text" yaml:"text"`
	Date   string `json:"date,omitempty" yaml:"date,omitempty"`
}

// specFromComments groups the flat comment list of a file back into threads.
func specFromComments(cs []docx.Comment) docxSpec {
	var spec docxSpec
	index := map[string]int{}
	for _, c := range cs {
		msg := docxMessageSpec{Author: c.Author, Text: c.Text, Date: c.Date}
		if c.ParentID == "" {
			index[c.ID] = len(spec.Threads)
			spec.Threads = append(spec.Threads, docxThreadSpec{Messages: []docxMessageSpec{msg}})
			continue
		}
		if at, ok := index[c.ParentID]; ok {
			spec.Threads[at].Messages = append(spec.Threads[at].Messages, msg)
			continue
		}
		// orphan reply: keep it rather than drop it
		index[c.ID] = len(spec.Threads)
		spec.Threads = append(spec.Threads, docxThreadSpec{Messages: []docxMessageSpec{msg}})
	}
	return spec
}

func docxCommentsRebuildCmd() *cobra.Command {
	var specPath string
	cmd := &cobra.Command{
		Use:   "comments-rebuild FILE",
		Short: "Replace every comment with threads from a JSON spec",
		Long: `Writes one comment element per message, with its author and date, and
expresses replies with the w15:paraIdParent link the portal expects.

The spec is YAML (JSON is accepted too, since YAML is a superset), so it can be
produced and edited with yq. Take the current threads out with
"oo docx comments FILE --export spec.yaml" and feed them back here.

Spec format:

  {"threads":[
     {"messages":[
        {"author":"Alice Anderson","text":"question","date":"2026-10-12T14:00:00Z"},
        {"author":"Bob Baker","text":"answer","date":"2026-10-13T09:00:00Z"}
     ]}
  ]}

Each message after the first is a reply to the one before it. Existing comments
are replaced and comment anchors in document.xml are removed; run highlighting
stays.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("give exactly one FILE")
			}
			raw, err := os.ReadFile(specPath)
			if err != nil {
				return fmt.Errorf("read spec: %w", err)
			}
			var spec docxSpec
			if err := yaml.Unmarshal(raw, &spec); err != nil {
				return fmt.Errorf("parse spec: %w", err)
			}
			if len(spec.Threads) == 0 {
				return fmt.Errorf("spec has no threads")
			}
			threads := make([]docx.ThreadSpec, 0, len(spec.Threads))
			n := 0
			for _, t := range spec.Threads {
				msgs := make([]docx.MessageSpec, 0, len(t.Messages))
				for _, m := range t.Messages {
					if strings.TrimSpace(m.Author) == "" {
						return fmt.Errorf("thread message has no author")
					}
					msgs = append(msgs, docx.MessageSpec{Author: m.Author, Text: m.Text, Date: m.Date})
					n++
				}
				threads = append(threads, docx.ThreadSpec{Messages: msgs})
			}
			f, err := docx.Open(args[0])
			if err != nil {
				return err
			}
			if err := f.RebuildComments(threads); err != nil {
				return err
			}
			if err := f.Save(args[0]); err != nil {
				return err
			}
			printObject(map[string]any{
				"file": args[0], "threads": len(threads), "messages": n, "changes": f.Changes(),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&specPath, "plan", "", "path to the JSON spec (alias)")
	cmd.Flags().StringVar(&specPath, "spec", "", "path to the JSON spec")
	_ = cmd.MarkFlagRequired("spec")
	return cmd
}

func docxShowCmd() *cobra.Command {
	var find string
	var index int
	cmd := &cobra.Command{
		Use:   "show FILE",
		Short: "Show one paragraph, or list every paragraph with --find omitted",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := openDocxArg(args)
			if err != nil {
				return err
			}
			texts := f.ParagraphTexts()
			if find == "" && !cmd.Flags().Changed("index") {
				for i, t := range texts {
					fmt.Printf("%03d| %s\n", i, t)
				}
				return nil
			}
			i, err := locate(f, find, index, cmd)
			if err != nil {
				return err
			}
			printObject(map[string]any{"index": i, "text": texts[i]})
			return nil
		},
	}
	cmd.Flags().StringVar(&find, "find", "", "substring that must match exactly one paragraph")
	cmd.Flags().IntVar(&index, "index", 0, "paragraph index (alternative to --find)")
	return cmd
}

func docxTextCmd() *cobra.Command {
	var find, old, to string
	cmd := &cobra.Command{
		Use:   "text FILE",
		Short: "Replace a substring inside one paragraph",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := openDocxArg(args)
			if err != nil {
				return err
			}
			i, err := f.FindParagraph(find)
			if err != nil {
				return err
			}
			if err := f.ReplaceInParagraph(i, old, to); err != nil {
				return err
			}
			return saveDocx(f, args[0], map[string]any{"replaced": old, "with": to, "paragraph": i})
		},
	}
	cmd.Flags().StringVar(&find, "find", "", "locator paragraph substring")
	cmd.Flags().StringVar(&old, "old", "", "text to replace inside that paragraph")
	cmd.Flags().StringVar(&to, "to", "", "replacement text")
	_ = cmd.MarkFlagRequired("find")
	_ = cmd.MarkFlagRequired("old")
	return cmd
}

func docxSetCmd() *cobra.Command {
	var find, to string
	var index int
	cmd := &cobra.Command{
		Use:   "set FILE",
		Short: "Replace the whole text of one paragraph",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := openDocxArg(args)
			if err != nil {
				return err
			}
			i, err := locate(f, find, index, cmd)
			if err != nil {
				return err
			}
			if err := f.SetParagraphText(i, to); err != nil {
				return err
			}
			return saveDocx(f, args[0], map[string]any{"paragraph": i, "text": to})
		},
	}
	cmd.Flags().StringVar(&find, "find", "", "locator paragraph substring")
	cmd.Flags().IntVar(&index, "index", 0, "paragraph index (alternative to --find)")
	cmd.Flags().StringVar(&to, "to", "", "new paragraph text")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func docxCommentCmd() *cobra.Command {
	var find, highlight, author, initials, date, text string
	var index int
	cmd := &cobra.Command{
		Use:   "comment FILE",
		Short: "Add a comment, anchored to a paragraph or to a highlighted substring",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := openDocxArg(args)
			if err != nil {
				return err
			}
			i, err := locate(f, find, index, cmd)
			if err != nil {
				return err
			}
			id, err := f.AddComment(i, docx.CommentOptions{
				Author: author, Initials: initials, Date: date, Text: text, Highlight: highlight,
			})
			if err != nil {
				return err
			}
			return saveDocx(f, args[0], map[string]any{"comment_id": id, "paragraph": i, "highlight": highlight})
		},
	}
	cmd.Flags().StringVar(&find, "find", "", "locator paragraph substring")
	cmd.Flags().IntVar(&index, "index", 0, "paragraph index (alternative to --find)")
	cmd.Flags().StringVar(&highlight, "highlight", "", "substring to anchor and highlight; omit to anchor the whole paragraph")
	cmd.Flags().StringVar(&author, "author", "", "comment author")
	cmd.Flags().StringVar(&initials, "initials", "", "author initials")
	cmd.Flags().StringVar(&date, "date", "", "RFC3339 timestamp")
	cmd.Flags().StringVar(&text, "text", "", "comment body")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func docxReplyCmd() *cobra.Command {
	var id, author, text string
	cmd := &cobra.Command{
		Use:   "reply FILE",
		Short: "Append a message to an existing comment thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := openDocxArg(args)
			if err != nil {
				return err
			}
			if err := f.ReplyToComment(id, author, text); err != nil {
				return err
			}
			return saveDocx(f, args[0], map[string]any{"replied_to": id})
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "comment id")
	cmd.Flags().StringVar(&author, "author", "", "author")
	cmd.Flags().StringVar(&text, "text", "", "message")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func docxListCommentsCmd() *cobra.Command {
	var export string
	cmd := &cobra.Command{
		Use:   "comments FILE",
		Short: "List every comment thread, or export them as a YAML spec",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := openDocxArg(args)
			if err != nil {
				return err
			}
			cs, err := f.Comments()
			if err != nil {
				return err
			}
			if export != "" {
				out, err := yaml.Marshal(specFromComments(cs))
				if err != nil {
					return fmt.Errorf("marshal spec: %w", err)
				}
				if export == "-" {
					_, err = os.Stdout.Write(out)
					return err
				}
				if err := os.WriteFile(export, out, 0o644); err != nil {
					return fmt.Errorf("write spec: %w", err)
				}
				printObject(map[string]any{"file": args[0], "spec": export, "threads": len(specFromComments(cs).Threads)})
				return nil
			}
			rows := make([]map[string]any, 0, len(cs))
			for _, c := range cs {
				rows = append(rows, map[string]any{
					"id": c.ID, "author": c.Author, "date": c.Date, "parent": c.ParentID,
					"text": strings.ReplaceAll(c.Text, "\n", " | "),
				})
			}
			printObject(map[string]any{"count": len(rows), "comments": rows})
			return nil
		},
	}
	cmd.Flags().StringVar(&export, "export", "", "write the threads to a YAML spec file (- for stdout)")
	return cmd
}

// docxOp is one entry of a batch plan.
type docxOp struct {
	Op        string `json:"op"`
	Find      string `json:"find"`
	Exact     bool   `json:"exact"`
	Index     *int   `json:"index"`
	Template  string `json:"template"`
	Old       string `json:"old"`
	To        string `json:"to"`
	Text      string `json:"text"`
	Highlight string `json:"highlight"`
	Author    string `json:"author"`
	Initials  string `json:"initials"`
	Date      string `json:"date"`
	ID        string `json:"id"`
}

type docxPlan struct {
	Ops []docxOp `json:"ops"`
}

func docxApplyCmd() *cobra.Command {
	var planPath string
	cmd := &cobra.Command{
		Use:   "apply FILE",
		Short: "Apply a JSON plan of ops in one open and one save",
		Long: `The plan is a JSON object with an "ops" array. Supported ops:

  {"op":"replace","find":"locator","old":"a","to":"b"}
  {"op":"set","find":"locator","exact":true,"to":"whole new paragraph"}
  {"op":"delete","find":"locator","exact":true}
  {"op":"insert_after","find":"locator","template":"locator","text":"new paragraph"}
  {"op":"comment","find":"locator","highlight":"substring","author":"...","text":"..."}
  {"op":"reply","id":"3","author":"...","text":"..."}

Every op is applied in order; the file is written once at the end, so a failed
plan leaves the input untouched.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("give exactly one FILE")
			}
			raw, err := os.ReadFile(planPath)
			if err != nil {
				return fmt.Errorf("read plan: %w", err)
			}
			var plan docxPlan
			if err := json.Unmarshal(raw, &plan); err != nil {
				return fmt.Errorf("parse plan: %w", err)
			}
			f, err := docx.Open(args[0])
			if err != nil {
				return err
			}
			applied := []string{}
			for n, op := range plan.Ops {
				if err := applyDocxOp(f, op); err != nil {
					return fmt.Errorf("op %d (%s): %w", n+1, op.Op, err)
				}
				applied = append(applied, fmt.Sprintf("%d:%s", n+1, op.Op))
			}
			if err := f.Save(args[0]); err != nil {
				return err
			}
			printObject(map[string]any{"file": args[0], "ops_applied": applied, "count": len(applied)})
			return nil
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "path to the JSON plan")
	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

func applyDocxOp(f *docx.File, op docxOp) error {
	locate := func() (int, error) {
		if op.Index != nil {
			return *op.Index, nil
		}
		if op.Exact {
			return f.FindParagraphExact(op.Find)
		}
		return f.FindParagraph(op.Find)
	}
	switch op.Op {
	case "replace":
		i, err := locate()
		if err != nil {
			return err
		}
		return f.ReplaceInParagraph(i, op.Old, op.To)
	case "set":
		i, err := locate()
		if err != nil {
			return err
		}
		return f.SetParagraphText(i, op.To)
	case "delete":
		i, err := locate()
		if err != nil {
			return err
		}
		return f.DeleteParagraph(i)
	case "insert_after":
		i, err := locate()
		if err != nil {
			return err
		}
		t, err := f.FindParagraph(op.Template)
		if err != nil {
			return err
		}
		return f.InsertParagraphAfter(i, t, op.Text)
	case "comment":
		i, err := locate()
		if err != nil {
			return err
		}
		_, err = f.AddComment(i, docx.CommentOptions{
			Author: op.Author, Initials: op.Initials, Date: op.Date,
			Text: op.Text, Highlight: op.Highlight,
		})
		return err
	case "reply":
		return f.ReplyToComment(op.ID, op.Author, op.Text)
	default:
		return fmt.Errorf("unknown op %q", op.Op)
	}
}

func locate(f *docx.File, find string, index int, cmd *cobra.Command) (int, error) {
	if cmd.Flags().Changed("index") {
		return index, nil
	}
	if find == "" {
		return 0, fmt.Errorf("give --find or --index")
	}
	return f.FindParagraph(find)
}

func openDocxArg(args []string) (*docx.File, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("give exactly one FILE")
	}
	return docx.Open(args[0])
}

func saveDocx(f *docx.File, path string, extra map[string]any) error {
	if err := f.Save(path); err != nil {
		return err
	}
	extra["file"] = path
	extra["changes"] = f.Changes()
	printObject(extra)
	return nil
}
