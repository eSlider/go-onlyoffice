package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRootRegistersSubjects(t *testing.T) {
	want := []string{
		"calendar", "projects", "tasks", "users", "whoami",
		"contacts", "persons", "companies",
		"opportunities", "cases", "crm-tasks", "applications", "crm",
	}
	got := make(map[string]bool, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		got[c.Name()] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("missing root subcommand %q; have %v", name, rootCmd.Commands())
		}
	}
}

func TestRootHelpListsSubjects(t *testing.T) {
	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"--help"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, snippet := range []string{"calendar", "projects", "tasks", "users", "opportunities"} {
		if !strings.Contains(help, snippet) {
			t.Fatalf("help missing %q", snippet)
		}
	}
}

func TestProjectsAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"prj"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name() != "projects" {
		t.Fatalf("prj alias resolved to %q", cmd.Name())
	}
}

func TestNewOOReturnsErrorWithoutCredentials(t *testing.T) {
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

	errBuf := &bytes.Buffer{}
	rootCmd.SetErr(errBuf)
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"users", "list"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	err = rootCmd.Execute()
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
