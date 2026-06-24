// Package bootstrap loads CLI environment and constructs an authenticated
// OnlyOffice client. Shared by cmd/oo and cmd/office; the library never
// loads dotfiles.
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
	"github.com/joho/godotenv"
)

// LoadEnv reads .env from the current working directory and applies OO_* aliases.
func LoadEnv() {
	_ = godotenv.Load(".env")
	applyEnvAliases()
}

// NewClient loads env, validates credentials, and authenticates against OnlyOffice.
func NewClient(ctx context.Context) (*onlyoffice.Client, error) {
	LoadEnv()
	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" || creds.User == "" || creds.Password == "" {
		return nil, fmt.Errorf("need ONLYOFFICE_URL (or ONLYOFFICE_HOST/OO_URL), user (ONLYOFFICE_USER or ONLYOFFICE_NAME/OO_USER), password (ONLYOFFICE_PASS or ONLYOFFICE_PASSWORD/OO_PASS)")
	}
	c := onlyoffice.NewClient(creds)
	c.SetDefaults(onlyoffice.GetEnvironmentDefaults())
	if err := c.AuthenticateContext(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func applyEnvAliases() {
	setEnvIfEmpty("ONLYOFFICE_URL", "OO_URL")
	setEnvIfEmpty("ONLYOFFICE_USER", "OO_USER")
	setEnvIfEmpty("ONLYOFFICE_PASS", "OO_PASS")
}

func setEnvIfEmpty(dst, src string) {
	if strings.TrimSpace(os.Getenv(dst)) != "" {
		return
	}
	if v := strings.TrimSpace(os.Getenv(src)); v != "" {
		_ = os.Setenv(dst, v)
	}
}
