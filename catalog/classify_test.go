package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifierRules(t *testing.T) {
	cl := &Classifier{
		WorkRemotes: []string{"git.internal.example"},
		WorkNames:   []string{"acme"},
		MailOrgs: []MailRule{
			{Domain: "acme.example", Org: "Acme", Zone: "hot", Role: "work"},
			{Suffix: ".gov.example", Org: "Public", Zone: "warm", Role: "work"},
		},
	}
	if role, zone := cl.ClassifyProject("acme-app", "git@git.internal.example:team/acme-app.git"); role != "work" || zone != "hot" {
		t.Fatalf("remote: %s/%s", role, zone)
	}
	if role, zone := cl.ClassifyProject("acme-demo", ""); role != "work" || zone != "warm" {
		t.Fatalf("name: %s/%s", role, zone)
	}
	if role, zone := cl.ClassifyProject("experiment-x", ""); role != "experiment" || zone != "cold" {
		t.Fatalf("experiment: %s/%s", role, zone)
	}
	if org, zone, role := cl.ClassifyMail("", "bob@acme.example"); org != "Acme" || zone != "hot" || role != "work" {
		t.Fatalf("mail: %s/%s/%s", org, zone, role)
	}
	if org, _, _ := cl.ClassifyMail("X", "x@team.gov.example"); org != "Public" {
		t.Fatalf("suffix: %s", org)
	}
	if org, zone, role := cl.ClassifyMail("", "someone@unknown.example"); org != "" || zone != "private" || role != "unknown" {
		t.Fatalf("neutral: %s/%s/%s", org, zone, role)
	}
}

func TestLoadClassifier(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "classify.yaml")
	if err := os.WriteFile(path, []byte("work_names:\n  - acme\nmail_orgs:\n  - domain: acme.example\n    org: Acme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cl, err := LoadClassifier(path)
	if err != nil {
		t.Fatal(err)
	}
	if role, _ := cl.ClassifyProject("acme-app", ""); role != "work" {
		t.Fatalf("role=%s", role)
	}
	if org, zone, _ := cl.ClassifyMail("", "a@acme.example"); org != "Acme" || zone != "hot" {
		t.Fatalf("org=%s zone=%s", org, zone)
	}

	t.Setenv("OO_CATALOG_CONFIG", path)
	if envCl, err := LoadClassifierFromEnv(); err != nil || envCl == nil || len(envCl.WorkNames) == 0 {
		t.Fatalf("env: %+v %v", envCl, err)
	}
	t.Setenv("OO_CATALOG_CONFIG", "")
	neutral, err := LoadClassifierFromEnv()
	if err != nil || len(neutral.WorkNames) != 0 || len(neutral.MailOrgs) != 0 {
		t.Fatalf("neutral: %+v %v", neutral, err)
	}
}
