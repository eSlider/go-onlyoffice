package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// outputFormat is the value of the global --output/-o flag. Valid values:
// "table" (default, tabwriter-rendered), "json" (machine-readable).
var outputFormat = "table"

var rootCmd = &cobra.Command{
	Use:           "oo",
	Short:         "OnlyOffice Workspace CLI — subject-based command tree",
	Long:          "oo is a thin CLI over github.com/eslider/go-onlyoffice.\nCommands are grouped by OnlyOffice subject (calendar, projects, tasks, users, persons, companies, opportunities, cases, crm-tasks, applications).",
	SilenceUsage:  true,
	SilenceErrors: false,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format: table|json")
}

// execute runs the root command. Exported only to main.go in the same package.
func execute() error { return rootCmd.Execute() }

// newOO loads env (incl. .env in CWD) and returns an authenticated client.
// godotenv is a CLI-only concern; the library itself never loads dotfiles.
func newOO(cmd *cobra.Command) (*onlyoffice.Client, error) {
	_ = godotenv.Load()
	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" || creds.User == "" || creds.Password == "" {
		return nil, fmt.Errorf("need ONLYOFFICE_URL (or ONLYOFFICE_HOST), user (ONLYOFFICE_USER or ONLYOFFICE_NAME), password (ONLYOFFICE_PASS or ONLYOFFICE_PASSWORD)")
	}
	c := onlyoffice.NewClient(creds)
	c.SetDefaults(onlyoffice.GetEnvironmentDefaults())
	if err := c.AuthenticateContext(cmd.Context()); err != nil {
		return nil, err
	}
	return c, nil
}

// printJSON dumps any value as indented JSON.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// printObject renders a single value. JSON output dumps verbatim; table output
// prints key/value pairs when v is a map, otherwise falls back to JSON.
func printObject(v any) {
	if outputFormat == "json" {
		printJSON(v)
		return
	}
	m, ok := v.(map[string]any)
	if !ok {
		printJSON(v)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, k := range sortedKeys(m) {
		fmt.Fprintf(w, "%s\t%s\n", k, fmtCell(m[k]))
	}
	_ = w.Flush()
}

// printTable renders a list of rows. headers select & order the columns.
// When outputFormat=="json" the raw slice is dumped as-is.
func printTable(headers []string, rows []map[string]any) {
	if outputFormat == "json" {
		printJSON(rows)
		return
	}
	if len(rows) == 0 {
		fmt.Println("(empty)")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i, h := range headers {
			cells[i] = fmtCell(row[h])
		}
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}
	_ = w.Flush()
}

// fmtCell turns an arbitrary value into a compact table cell.
func fmtCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return truncate(x, 80)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int, int32, int64:
		return fmt.Sprintf("%d", x)
	default:
		b, _ := json.Marshal(x)
		return truncate(string(b), 80)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

func sortedKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	// Deterministic ordering; small map so O(n log n) is fine.
	for i := 1; i < len(ks); i++ {
		for j := i; j > 0 && ks[j-1] > ks[j]; j-- {
			ks[j-1], ks[j] = ks[j], ks[j-1]
		}
	}
	return ks
}

// flexIDFloat coerces OnlyOffice numeric id fields surfaced as float64 / int
// / string into a float64.
func flexIDFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(x), 64)
		return f
	}
}

// idString returns the id field as a string (works for int/float/string).
func idString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int:
		return strconv.Itoa(x)
	default:
		return fmt.Sprint(x)
	}
}
