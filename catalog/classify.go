package catalog

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Classifier maps project trees and mail identities to catalog org/zone/role.
//
// The library ships with neutral defaults: nothing is classified as work unless
// the deployment supplies rules. Those rules are deployment-specific, so they
// live in a YAML config file (path from OO_CATALOG_CONFIG or the --config flag),
// not in the code. See catalog/classify.example.yaml.
type Classifier struct {
	// WorkRemotes: a git remote containing any of these substrings → work/hot.
	WorkRemotes []string `yaml:"work_remotes,omitempty"`
	// WorkNames: a project name containing any of these substrings → work/warm.
	WorkNames []string `yaml:"work_names,omitempty"`
	// MailOrgs: ordered mail-identity rules; the first match wins.
	MailOrgs []MailRule `yaml:"mail_orgs,omitempty"`
}

// MailRule maps an email domain and/or a display-name substring to an org with
// a zone/role. At least one of Domain, Suffix or Name must be set.
type MailRule struct {
	Domain string `yaml:"domain,omitempty"` // exact domain, case-insensitive
	Suffix string `yaml:"suffix,omitempty"` // domain suffix, e.g. ".example.com"
	Name   string `yaml:"name,omitempty"`   // substring of the display name
	Org    string `yaml:"org"`
	Zone   string `yaml:"zone,omitempty"` // default "hot"
	Role   string `yaml:"role,omitempty"` // default "work"
}

// DefaultClassifier returns the neutral classifier (no deployment rules).
func DefaultClassifier() *Classifier { return &Classifier{} }

// LoadClassifier reads a classifier config from a YAML file.
func LoadClassifier(path string) (*Classifier, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Classifier
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse classifier config %s: %w", path, err)
	}
	return &c, nil
}

// LoadClassifierFromEnv loads the classifier named by OO_CATALOG_CONFIG. An
// empty variable yields the neutral classifier.
func LoadClassifierFromEnv() (*Classifier, error) {
	path := strings.TrimSpace(os.Getenv("OO_CATALOG_CONFIG"))
	if path == "" {
		return DefaultClassifier(), nil
	}
	return LoadClassifier(path)
}

// ClassifyProject returns (role, zone) for a project name and git remote.
// Generic name heuristics come first; deployment rules supply the work cases.
func (c *Classifier) ClassifyProject(name, remote string) (role, zone string) {
	lower := strings.ToLower(name)
	remoteL := strings.ToLower(remote)
	switch {
	case strings.Contains(lower, "experiment") || strings.HasPrefix(lower, "test"):
		return "experiment", "cold"
	case lower == "mama" || lower == "personal" || strings.Contains(lower, "private"):
		return "personal", "private"
	}
	if c != nil {
		for _, r := range c.WorkRemotes {
			if r != "" && strings.Contains(remoteL, strings.ToLower(r)) {
				return "work", "hot"
			}
		}
		for _, n := range c.WorkNames {
			if n != "" && strings.Contains(lower, strings.ToLower(n)) {
				return "work", "warm"
			}
		}
	}
	return "unknown", "warm"
}

// ClassifyMail returns (org, zone, role) for a mail identity. name is the
// display name (may be empty); email is the address.
func (c *Classifier) ClassifyMail(name, email string) (org, zone, role string) {
	em := NormalizeEmail(email)
	_, domain, _ := strings.Cut(em, "@")
	nameL := strings.ToLower(strings.TrimSpace(name))
	if c != nil {
		for _, r := range c.MailOrgs {
			if !mailRuleMatches(r, domain, nameL) {
				continue
			}
			z, ro := r.Zone, r.Role
			if z == "" {
				z = "hot"
			}
			if ro == "" {
				ro = "work"
			}
			return r.Org, z, ro
		}
	}
	if strings.HasSuffix(domain, ".de") && looksPublicSector(domain) {
		return domain, "warm", "work"
	}
	return "", "private", "unknown"
}

func mailRuleMatches(r MailRule, domain, nameL string) bool {
	if r.Domain != "" && domain == strings.ToLower(strings.TrimSpace(r.Domain)) {
		return true
	}
	if r.Suffix != "" && strings.HasSuffix(domain, strings.ToLower(strings.TrimSpace(r.Suffix))) {
		return true
	}
	if r.Name != "" && nameL != "" && strings.Contains(nameL, strings.ToLower(strings.TrimSpace(r.Name))) {
		return true
	}
	return false
}
