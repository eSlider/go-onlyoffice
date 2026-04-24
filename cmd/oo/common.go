package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "oo",
	Short: "OnlyOffice Workspace CLI (Go port of cv/bin/office)",
}

// execute runs the root command. Exported only to main.go in the same package.
func execute() error { return rootCmd.Execute() }

func init() {
	rootCmd.AddCommand(cmdCalList())
	rootCmd.AddCommand(cmdCalEvents())
	rootCmd.AddCommand(cmdCalAdd())
	rootCmd.AddCommand(cmdCalDel())
	rootCmd.AddCommand(cmdTaskList())
	rootCmd.AddCommand(cmdTaskAdd())
	rootCmd.AddCommand(cmdSubtaskAdd())
	rootCmd.AddCommand(cmdTaskUpdate())
	rootCmd.AddCommand(cmdCRMContacts())
	rootCmd.AddCommand(cmdCRMAddContact())
	rootCmd.AddCommand(cmdCRMDeals())
	rootCmd.AddCommand(cmdCRMAddDeal())
	rootCmd.AddCommand(cmdCRMCases())
	rootCmd.AddCommand(cmdAppsSync())
}

// newOO loads env (incl. .env in CWD) and returns an authenticated client.
// godotenv is a CLI-only concern; the library itself never loads dotfiles.
func newOO() (*onlyoffice.Client, error) {
	_ = godotenv.Load()
	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" || creds.User == "" || creds.Password == "" {
		return nil, fmt.Errorf("need ONLYOFFICE_URL (or ONLYOFFICE_HOST), user (ONLYOFFICE_USER or ONLYOFFICE_NAME), password (ONLYOFFICE_PASS or ONLYOFFICE_PASSWORD)")
	}
	c := onlyoffice.NewClient(creds)
	c.SetDefaults(onlyoffice.GetEnvironmentDefaults())
	if err := c.Authenticate(); err != nil {
		return nil, err
	}
	return c, nil
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// flexIDFloat coerces OnlyOffice numeric id fields surfaced as float64 / int
// / string into a float64. CLI-only; the library's own flexInt is unexported.
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
