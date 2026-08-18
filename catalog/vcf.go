package catalog

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-vcard"
)

// ParseVCFFile extracts person candidates from a vCard file (may contain many cards).
func ParseVCFFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := vcard.NewDecoder(f)
	var out []Entry
	for {
		card, err := dec.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return out, fmt.Errorf("%s: %w", path, err)
		}
		e := entryFromCard(card, path)
		if e.Name == "" && e.First == "" && len(e.Emails) == 0 {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func entryFromCard(card vcard.Card, source string) Entry {
	fn := strings.TrimSpace(card.PreferredValue(vcard.FieldFormattedName))
	n := card.Name()
	first, last := "", ""
	if n != nil {
		first = strings.TrimSpace(n.GivenName)
		last = strings.TrimSpace(n.FamilyName)
	}
	if fn == "" {
		fn = strings.TrimSpace(first + " " + last)
	}
	if first == "" && last == "" && fn != "" {
		first, last = SplitDisplayName(fn)
	}
	var emails, phones []string
	for _, v := range card.Values(vcard.FieldEmail) {
		if e := NormalizeEmail(v); e != "" {
			emails = append(emails, e)
		}
	}
	for _, v := range card.Values(vcard.FieldTelephone) {
		p := strings.TrimSpace(v)
		if p != "" {
			phones = append(phones, p)
		}
	}
	org := strings.TrimSpace(card.PreferredValue(vcard.FieldOrganization))
	primary := ""
	if len(emails) > 0 {
		primary = emails[0]
	}
	return Entry{
		ID:      EntryID("person", primary, fn),
		Kind:    "person",
		Name:    fn,
		First:   first,
		Last:    last,
		Emails:  uniqueEmails(emails),
		Phones:  uniqueStrings(phones),
		Org:     org,
		Sources: []string{source},
		Zone:    "private",
		Role:    "unknown",
		Approve: false,
		Status:  "new",
	}
}

// ScanContactsRoot walks a contacts directory: *.vcf anywhere, person-named
// subdirs, and *@*.txt email hint files.
func ScanContactsRoot(root string) (*Document, error) {
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}

	var entries []Entry
	seenVCF := map[string]struct{}{}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".vcf") {
			if _, ok := seenVCF[path]; ok {
				return nil
			}
			seenVCF[path] = struct{}{}
			parsed, perr := ParseVCFFile(path)
			if perr != nil {
				entries = append(entries, Entry{
					ID:      EntryID("person", "", filepath.Base(path)),
					Kind:    "person",
					Name:    strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
					Sources: []string{path},
					Zone:    "private",
					Role:    "unknown",
					Notes:   "vcf_parse_error: " + perr.Error(),
					Status:  "new",
				})
				return nil
			}
			entries = append(entries, parsed...)
			return nil
		}
		// email hint files: name contains @ and ends with .txt
		if strings.Contains(name, "@") && strings.HasSuffix(lower, ".txt") {
			email := NormalizeEmail(strings.TrimSuffix(name, filepath.Ext(name)))
			if !strings.Contains(email, "@") {
				return nil
			}
			parent := filepath.Base(filepath.Dir(path))
			first, last := SplitDisplayName(parent)
			entries = append(entries, Entry{
				ID:      EntryID("person", email, parent),
				Kind:    "person",
				Name:    parent,
				First:   first,
				Last:    last,
				Emails:  []string{email},
				Sources: []string{path},
				Zone:    "private",
				Role:    "unknown",
				Status:  "new",
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Top-level person folders without VCF still become stub persons.
	dents, _ := os.ReadDir(root)
	for _, d := range dents {
		if !d.IsDir() {
			continue
		}
		name := d.Name()
		if name == "Contacts VCF's" || strings.HasPrefix(name, ".") {
			continue
		}
		// skip obvious non-person dumps
		lower := strings.ToLower(name)
		if strings.Contains(lower, "vcf") {
			continue
		}
		first, last := SplitDisplayName(name)
		id := EntryID("person", "", name)
		entries = append(entries, Entry{
			ID:      id,
			Kind:    "person",
			Name:    name,
			First:   first,
			Last:    last,
			Sources: []string{filepath.Join(root, name)},
			Zone:    "private",
			Role:    "unknown",
			Status:  "new",
			Notes:   "folder_stub",
		})
	}

	doc := MergeDocs(&Document{Entries: entries})
	return doc, nil
}

// ReadEmailsFromTxt reads loose emails from a text file (one per line or free text).
func ReadEmailsFromTxt(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.Contains(line, "@") && !strings.Contains(line, " ") {
			out = append(out, NormalizeEmail(line))
		}
	}
	return uniqueEmails(out), sc.Err()
}
