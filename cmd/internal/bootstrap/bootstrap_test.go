package bootstrap_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/eslider/go-onlyoffice/cmd/internal/bootstrap"
)

func TestLoadEnvFromCWDWithOOAliases(t *testing.T) {
	clearEnv(t, "ONLYOFFICE_URL", "ONLYOFFICE_USER", "ONLYOFFICE_PASS", "OO_URL", "OO_USER", "OO_PASS")
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	if err := os.WriteFile(".env", []byte("OO_URL=https://office.produktor.io\nOO_USER=user@example.com\nOO_PASS=secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	bootstrap.LoadEnv()
	if got := os.Getenv("ONLYOFFICE_URL"); got != "https://office.produktor.io" {
		t.Fatalf("ONLYOFFICE_URL=%q", got)
	}
	if got := os.Getenv("ONLYOFFICE_USER"); got != "user@example.com" {
		t.Fatalf("ONLYOFFICE_USER=%q", got)
	}
	if got := os.Getenv("ONLYOFFICE_PASS"); got != "secret" {
		t.Fatalf("ONLYOFFICE_PASS=%q", got)
	}
}

func TestLoadEnvDoesNotOverrideCanonicalWithAlias(t *testing.T) {
	clearEnv(t, "ONLYOFFICE_URL", "OO_URL")
	t.Setenv("ONLYOFFICE_URL", "https://canonical.example")
	t.Setenv("OO_URL", "https://alias.example")

	bootstrap.LoadEnv()

	if got := os.Getenv("ONLYOFFICE_URL"); got != "https://canonical.example" {
		t.Fatalf("ONLYOFFICE_URL=%q", got)
	}
}

func TestNewClientReturnsErrorWithoutCredentials(t *testing.T) {
	clearEnv(t,
		"ONLYOFFICE_URL", "ONLYOFFICE_HOST", "ONLYOFFICE_USER", "ONLYOFFICE_NAME",
		"ONLYOFFICE_PASS", "ONLYOFFICE_PASSWORD",
		"OO_URL", "OO_USER", "OO_PASS",
	)
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	_, err = bootstrap.NewClient(context.Background())
	if err == nil {
		t.Fatal("expected error without credentials")
	}
	msg := err.Error()
	for _, want := range []string{"ONLYOFFICE_URL", "ONLYOFFICE_USER", "ONLYOFFICE_PASS"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		old, ok := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if ok {
				_ = os.Setenv(key, old)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}
