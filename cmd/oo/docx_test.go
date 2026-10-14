package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/internal/docx"
	"gopkg.in/yaml.v3"
)

const docxFixture = "../../testdata/docx/sample.docx"

func copyDocxFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(docxFixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "work.docx")
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatalf("write copy: %v", err)
	}
	return dst
}

// runOO executes the CLI and returns whatever it printed. The CLI reports
// through os.Stdout (printObject), so that is what gets captured.
func runOO(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	rootCmd.SetArgs(args)
	runErr := rootCmd.Execute()
	w.Close()
	os.Stdout = old
	rootCmd.SetArgs(nil)

	out, _ := io.ReadAll(r)
	r.Close()
	return string(out), runErr
}

// runOOJSON runs with -o json so output can be asserted on.
func runOOJSON(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runOO(t, append([]string{"-o", "json"}, args...)...)
}

func TestDocxCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "docx" {
			found = true
		}
	}
	if !found {
		t.Fatalf("root command has no docx subcommand")
	}
}

func TestDocxShowListsParagraphs(t *testing.T) {
	path := copyDocxFixture(t)
	out, err := runOO(t, "docx", "show", path)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(out, "The deposit is EUR 480.00") {
		t.Fatalf("expected paragraph listing, got:\n%s", out)
	}
}

func TestDocxApplyPlan(t *testing.T) {
	path := copyDocxFixture(t)

	plan := map[string]any{"ops": []map[string]any{
		{"op": "replace", "find": "payable up to the 5th day of the month", "old": "the 5th day", "to": "the 8th day"},
		{"op": "set", "find": "Either party may terminate", "to": "Either party may terminate this contract on two months of written notice."},
		{"op": "comment", "find": "parish of Encarnação", "highlight": "use permit no. 77", "author": "Alice Anderson", "initials": "AA", "text": "Could you confirm the permit number?"},
		{"op": "delete", "find": "SIXTH CLAUSE"},
		{"op": "insert_after", "find": "FIFTH CLAUSE", "template": "The parties agree that the air conditioning", "text": "SEVENTH CLAUSE"},
	}}
	b, _ := json.Marshal(plan)
	planPath := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(planPath, b, 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	out, err := runOOJSON(t, "docx", "apply", path, "--plan", planPath)
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"count": 5`) {
		t.Fatalf("expected 5 ops applied, got:\n%s", out)
	}

	f, err := docx.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	joined := strings.Join(f.ParagraphTexts(), "\n")
	switch {
	case strings.Contains(joined, "the 5th day"):
		t.Fatalf("replace op did not apply")
	case !strings.Contains(joined, "the 8th day"):
		t.Fatalf("replace op produced nothing")
	case strings.Contains(joined, "SIXTH CLAUSE"):
		t.Fatalf("delete op did not apply")
	case !strings.Contains(joined, "SEVENTH CLAUSE"):
		t.Fatalf("insert op did not apply")
	case !strings.Contains(joined, "two months of written notice"):
		t.Fatalf("set op did not apply")
	}
	cs, err := f.Comments()
	if err != nil || len(cs) != 1 {
		t.Fatalf("comment op: got %d comments err %v", len(cs), err)
	}
}

func TestDocxApplyIsAtomicOnFailure(t *testing.T) {
	path := copyDocxFixture(t)
	before, _ := os.ReadFile(path)

	plan := map[string]any{"ops": []map[string]any{
		{"op": "set", "find": "Either party may terminate", "to": "changed"},
		{"op": "replace", "find": "no such paragraph anywhere", "old": "a", "to": "b"},
	}}
	b, _ := json.Marshal(plan)
	planPath := filepath.Join(t.TempDir(), "plan.json")
	os.WriteFile(planPath, b, 0o644)

	if _, err := runOO(t, "docx", "apply", path, "--plan", planPath); err == nil {
		t.Fatalf("a plan with a failing op must return an error")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatalf("file changed even though the plan failed")
	}
}

func TestDocxAmbiguousLocatorFails(t *testing.T) {
	path := copyDocxFixture(t)
	if _, err := runOO(t, "docx", "set", path, "--find", "the", "--to", "x"); err == nil {
		t.Fatalf("ambiguous locator must fail rather than guess")
	}
}

func TestDocxCommentsListAndReply(t *testing.T) {
	path := copyDocxFixture(t)
	if _, err := runOO(t, "docx", "comment", path, "--find", "The deposit is EUR 480.00",
		"--highlight", "The deposit", "--author", "Bob", "--text", "Noted."); err != nil {
		t.Fatalf("comment: %v", err)
	}
	out, err := runOOJSON(t, "docx", "comments", path)
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	if !strings.Contains(out, "Noted.") {
		t.Fatalf("comment not listed:\n%s", out)
	}
	if _, err := runOO(t, "docx", "reply", path, "--id", "0", "--author", "Alice", "--text", "Thank you."); err != nil {
		t.Fatalf("reply: %v", err)
	}
	f, _ := docx.Open(path)
	cs, _ := f.Comments()
	if len(cs) != 2 {
		t.Fatalf("a reply is its own comment: got %d, want 2", len(cs))
	}
	var reply, parent *docx.Comment
	for k := range cs {
		if cs[k].ParentID != "" {
			reply = &cs[k]
		} else {
			parent = &cs[k]
		}
	}
	if reply == nil || parent == nil || reply.ParentID != parent.ID {
		t.Fatalf("reply link missing: %+v", cs)
	}
	if !strings.Contains(reply.Text, "Thank you.") {
		t.Fatalf("reply body lost: %q", reply.Text)
	}
}

func TestDocxCommentsExportYAMLRoundTrip(t *testing.T) {
	path := copyDocxFixture(t)
	specPath := filepath.Join(t.TempDir(), "spec.yaml")
	spec := `threads:
  - messages:
      - author: Alice Anderson
        text: a question
        date: "2026-10-12T11:50:00Z"
      - author: Bob Baker
        text: an answer
        date: "2026-10-13T09:00:00Z"
  - messages:
      - author: Alice Anderson
        text: a standalone note
        date: "2026-10-14T18:00:00Z"
`
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if _, err := runOOJSON(t, "docx", "comments-rebuild", path, "--spec", specPath); err != nil {
		t.Fatalf("comments-rebuild: %v", err)
	}

	// export back out and check the shape survived
	outPath := filepath.Join(t.TempDir(), "out.yaml")
	if _, err := runOOJSON(t, "docx", "comments", path, "--export", outPath); err != nil {
		t.Fatalf("export: %v", err)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	var got docxSpec
	if err := yaml.Unmarshal(raw, &got); err != nil {
		t.Fatalf("exported spec is not valid YAML: %v\n%s", err, raw)
	}
	if len(got.Threads) != 2 {
		t.Fatalf("want 2 threads, got %d\n%s", len(got.Threads), raw)
	}
	if len(got.Threads[0].Messages) != 2 {
		t.Fatalf("want 2 messages in the first thread, got %d", len(got.Threads[0].Messages))
	}
	if got.Threads[0].Messages[1].Author != "Bob Baker" {
		t.Fatalf("author lost in the round trip: %+v", got.Threads[0].Messages[1])
	}
	if got.Threads[0].Messages[1].Date != "2026-10-13T09:00:00Z" {
		t.Fatalf("date lost in the round trip: %+v", got.Threads[0].Messages[1])
	}
	if got.Threads[1].Messages[0].Text != "a standalone note" {
		t.Fatalf("second thread wrong: %+v", got.Threads[1])
	}
}
