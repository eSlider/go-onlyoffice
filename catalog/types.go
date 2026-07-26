// Package catalog builds a YAML inventory of CRM persons/companies from
// filesystem contacts (VCF, folders) and project trees, then matches/applies
// against OnlyOffice CRM.
package catalog

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Entry is one catalog row (person or company).
type Entry struct {
	ID      string   `yaml:"id"`
	Kind    string   `yaml:"kind"` // person | company
	Name    string   `yaml:"name,omitempty"`
	First   string   `yaml:"first,omitempty"`
	Last    string   `yaml:"last,omitempty"`
	Emails  []string `yaml:"emails,omitempty"`
	Phones  []string `yaml:"phones,omitempty"`
	Org     string   `yaml:"org,omitempty"`
	Sources []string `yaml:"sources,omitempty"`
	Zone    string   `yaml:"zone"`
	Role    string   `yaml:"role"`
	OOID    string   `yaml:"oo_id,omitempty"`
	Approve bool     `yaml:"approve"`
	Status  string   `yaml:"status,omitempty"` // new | exists | conflict | applied | skipped
	Notes   string   `yaml:"notes,omitempty"`
	Remote  string   `yaml:"remote,omitempty"`
	GitRoot string   `yaml:"git_root,omitempty"`
}

// Document is the on-disk catalog file.
type Document struct {
	GeneratedAt string  `yaml:"generated_at"`
	Entries     []Entry `yaml:"entries"`
}

// LoadYAML reads a catalog document from path.
func LoadYAML(path string) (*Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc Document
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// SaveYAML writes the catalog document.
func SaveYAML(path string, doc *Document) error {
	doc.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// NormalizeEmail lowercases and trims.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeName collapses whitespace and lowercases for matching.
func NormalizeName(s string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(s)))
	return strings.Join(fields, " ")
}

// SplitDisplayName splits "First Last …" into first/last (last = remainder).
func SplitDisplayName(name string) (first, last string) {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// EntryID builds a stable-ish id from kind + email or name.
func EntryID(kind, email, name string) string {
	if e := NormalizeEmail(email); e != "" {
		return fmt.Sprintf("%s:%s", kind, e)
	}
	return fmt.Sprintf("%s:%s", kind, NormalizeName(name))
}

// MergeDocs unions entries by id; later sources append sources[] and fill empties.
func MergeDocs(docs ...*Document) *Document {
	byID := map[string]*Entry{}
	order := []string{}
	for _, d := range docs {
		if d == nil {
			continue
		}
		for i := range d.Entries {
			e := d.Entries[i]
			id := e.ID
			if id == "" {
				primary := ""
				if len(e.Emails) > 0 {
					primary = e.Emails[0]
				}
				name := e.Name
				if name == "" {
					name = strings.TrimSpace(e.First + " " + e.Last)
				}
				id = EntryID(e.Kind, primary, name)
				e.ID = id
			}
			if prev, ok := byID[id]; ok {
				mergeEntry(prev, &e)
			} else {
				cp := e
				byID[id] = &cp
				order = append(order, id)
			}
		}
	}
	out := &Document{Entries: make([]Entry, 0, len(order))}
	for _, id := range order {
		out.Entries = append(out.Entries, *byID[id])
	}
	return out
}

func mergeEntry(dst, src *Entry) {
	dst.Sources = uniqueStrings(append(dst.Sources, src.Sources...))
	dst.Emails = uniqueEmails(append(dst.Emails, src.Emails...))
	dst.Phones = uniqueStrings(append(dst.Phones, src.Phones...))
	if dst.First == "" {
		dst.First = src.First
	}
	if dst.Last == "" {
		dst.Last = src.Last
	}
	if dst.Name == "" {
		dst.Name = src.Name
	}
	if dst.Org == "" {
		dst.Org = src.Org
	}
	if dst.Remote == "" {
		dst.Remote = src.Remote
	}
	if dst.GitRoot == "" {
		dst.GitRoot = src.GitRoot
	}
	if dst.Notes == "" {
		dst.Notes = src.Notes
	}
	// Prefer more specific zone/role from project scans over default private.
	if dst.Zone == "private" && src.Zone != "" && src.Zone != "private" {
		dst.Zone = src.Zone
	}
	if dst.Role == "" || (dst.Role == "unknown" && src.Role != "" && src.Role != "unknown") {
		dst.Role = src.Role
	}
	// Preserve approve/oo_id/status from whichever already set.
	if !dst.Approve && src.Approve {
		dst.Approve = true
	}
	if dst.OOID == "" {
		dst.OOID = src.OOID
	}
	if dst.Status == "" {
		dst.Status = src.Status
	}
}

func uniqueEmails(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range in {
		e = NormalizeEmail(e)
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
