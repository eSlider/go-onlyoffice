package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestIndexCommandRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"index"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Name() != "index" {
		t.Fatalf("index resolved to %q", cmd.Name())
	}
	for _, name := range []string{"exts", "limit", "workers", "lang", "min-chars", "work-dir", "backend", "dry-run", "recursive", "json"} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("index: missing --%s flag", name)
		}
	}
	if cmd.PersistentFlags().Lookup("exts").DefValue != "pdf" {
		t.Errorf("--exts default = %q, want pdf", cmd.PersistentFlags().Lookup("exts").DefValue)
	}
	for _, verb := range []string{"index folder", "index files"} {
		sub, _, err := rootCmd.Find(strings.Fields(verb))
		if err != nil {
			t.Fatalf("%s: %v", verb, err)
		}
		if sub.Name() != strings.Fields(verb)[1] {
			t.Errorf("%s resolved to %q", verb, sub.Name())
		}
	}
}

func TestIndexSearchBackendFlag(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"search"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Flags().Lookup("backend") == nil {
		t.Fatal("search: missing --backend flag")
	}
	if cmd.Flags().Lookup("backend").DefValue != "oo" {
		t.Errorf("--backend default = %q, want oo", cmd.Flags().Lookup("backend").DefValue)
	}
}

func TestSplitList(t *testing.T) {
	got := splitList(" pdf , .PDF, docx ,, ")
	want := []string{"pdf", ".PDF", "docx"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitList = %v, want %v", got, want)
	}
}
