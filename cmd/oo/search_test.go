package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSearchCommandRegisteredWithFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"search"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name() != "search" {
		t.Fatalf("search resolved to %q", cmd.Name())
	}
	for _, name := range []string{"content", "folder", "limit", "json"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("search: missing --%s flag", name)
		}
	}
	if got := cmd.Flags().Lookup("limit").DefValue; got != "20" {
		t.Errorf("--limit default = %q, want 20", got)
	}
}

func TestSearchWithoutESURLIsClearError(t *testing.T) {
	clearEnv(t, "ONLYOFFICE_ES_URL", "ONLYOFFICE_ES_INDEX", "ONLYOFFICE_TENANT")
	errBuf := &bytes.Buffer{}
	rootCmd.SetErr(errBuf)
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"search", "Rechnung"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error without ONLYOFFICE_ES_URL")
	}
	if !strings.Contains(err.Error(), "ONLYOFFICE_ES_URL") {
		t.Fatalf("error %q missing ONLYOFFICE_ES_URL", err.Error())
	}
}
