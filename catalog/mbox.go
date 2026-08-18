package catalog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ScanOptions controls Thunderbird / mbox extraction depth.
type ScanOptions struct {
	// MboxHeaders, when true, also walks mbox-like files under the root and
	// extracts From/To/Cc/Reply-To addresses (headers only, no bodies).
	MboxHeaders bool
	// MboxMaxBytes skips individual mbox files larger than this (0 = 256 MiB default).
	MboxMaxBytes int64
}

// ScanThunderbirdRoot finds Thunderbird profiles under root and emits person rows
// from address books (*.mab) and Gloda global-messages-db.sqlite.
func ScanThunderbirdRoot(root string) (*Document, error) {
	return ScanThunderbirdRootOpts(root, ScanOptions{})
}

// ScanThunderbirdRootOpts is ScanThunderbirdRoot with optional mbox header pass.
func ScanThunderbirdRootOpts(root string, opts ScanOptions) (*Document, error) {
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}
	if opts.MboxMaxBytes <= 0 {
		opts.MboxMaxBytes = 256 << 20
	}

	var entries []Entry
	seenDB := map[string]struct{}{}
	seenMAB := map[string]struct{}{}
	seenMbox := map[string]struct{}{}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case "Cache", "cache2", "startupCache", "OfflineCache", "minidumps",
				"crashes", "safebrowsing", "thumbnails", "chrome", "extensions",
				"node_modules", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		lower := strings.ToLower(name)
		switch {
		case lower == "global-messages-db.sqlite":
			if _, ok := seenDB[path]; ok {
				return nil
			}
			seenDB[path] = struct{}{}
			parsed, perr := parseGlodaContacts(path)
			if perr != nil {
				entries = append(entries, Entry{
					ID:      EntryID("person", "", filepath.Base(path)),
					Kind:    "person",
					Name:    "gloda",
					Sources: []string{path},
					Zone:    "private",
					Role:    "unknown",
					Notes:   "gloda_parse_error: " + perr.Error(),
					Status:  "new",
				})
				return nil
			}
			entries = append(entries, parsed...)
		case strings.HasSuffix(lower, ".mab"):
			if _, ok := seenMAB[path]; ok {
				return nil
			}
			seenMAB[path] = struct{}{}
			parsed, perr := parseMABEmails(path)
			if perr != nil {
				return nil
			}
			entries = append(entries, parsed...)
		case opts.MboxHeaders && isLikelyMboxFile(name, path):
			if _, ok := seenMbox[path]; ok {
				return nil
			}
			seenMbox[path] = struct{}{}
			info, ierr := d.Info()
			if ierr != nil {
				return nil
			}
			if info.Size() > opts.MboxMaxBytes {
				return nil
			}
			parsed, perr := parseMboxHeaderEmails(path)
			if perr != nil {
				return nil
			}
			entries = append(entries, parsed...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return MergeDocs(&Document{Entries: entries}), nil
}

func isLikelyMboxFile(name, path string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".msf") || strings.HasSuffix(lower, ".sqlite") ||
		strings.HasSuffix(lower, ".mab") || strings.HasSuffix(lower, ".json") ||
		strings.HasSuffix(lower, ".dat") || strings.HasSuffix(lower, ".ini") ||
		strings.HasSuffix(lower, ".log") {
		return false
	}
	if strings.HasSuffix(lower, ".mbox") || strings.HasSuffix(lower, ".mbx") {
		return true
	}
	// Thunderbird: …/ImapMail/<server>/INBOX or …/Mail/Local Folders/Inbox
	// or …/INBOX.sbd/<folder>
	sep := string(filepath.Separator)
	norm := filepath.ToSlash(path)
	if strings.Contains(norm, "/ImapMail/") || strings.Contains(norm, "/Mail/") {
		if !strings.Contains(name, ".") {
			return true
		}
		known := map[string]struct{}{
			"inbox": {}, "sent": {}, "drafts": {}, "trash": {}, "archives": {}, "junk": {},
		}
		if _, ok := known[lower]; ok {
			return true
		}
	}
	if strings.Contains(filepath.Base(filepath.Dir(path)), ".sbd") && !strings.Contains(name, ".") {
		return true
	}
	_ = sep
	return false
}

// parseMboxHeaderEmails extracts addresses from From/To/Cc/Reply-To headers only.
func parseMboxHeaderEmails(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	emails := map[string]struct{}{}
	br := bufio.NewReaderSize(f, 1<<20)
	inHeaders := false
	var headerBuf strings.Builder

	flushHeaders := func() {
		if headerBuf.Len() == 0 {
			return
		}
		block := headerBuf.String()
		headerBuf.Reset()
		for _, line := range strings.Split(block, "\n") {
			lower := strings.ToLower(strings.TrimSpace(line))
			if strings.HasPrefix(lower, "from:") || strings.HasPrefix(lower, "to:") ||
				strings.HasPrefix(lower, "cc:") || strings.HasPrefix(lower, "reply-to:") ||
				strings.HasPrefix(lower, "sender:") {
				for _, m := range emailRE.FindAllString(line, -1) {
					em := NormalizeEmail(m)
					if !noisyEmail(em) {
						emails[em] = struct{}{}
					}
				}
			}
		}
	}

	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			// Classic mbox separator
			if strings.HasPrefix(line, "From ") && (len(line) == 5 || line[5] != ':') {
				flushHeaders()
				inHeaders = true
				continue
			}
			if inHeaders {
				trimmed := strings.TrimRight(line, "\r\n")
				if trimmed == "" {
					flushHeaders()
					inHeaders = false
					continue
				}
				headerBuf.WriteString(trimmed)
				headerBuf.WriteByte('\n')
				// cap pathological headers
				if headerBuf.Len() > 64<<10 {
					flushHeaders()
					inHeaders = false
				}
			}
		}
		if err == io.EOF {
			flushHeaders()
			break
		}
		if err != nil {
			flushHeaders()
			return nil, err
		}
	}

	var out []Entry
	for em := range emails {
		org, zone, role := classifyMailIdentity("", em)
		out = append(out, Entry{
			ID:      EntryID("person", em, ""),
			Kind:    "person",
			Emails:  []string{em},
			Org:     org,
			Sources: []string{path},
			Zone:    zone,
			Role:    role,
			Approve: false,
			Status:  "new",
			Notes:   "thunderbird_mbox_header",
		})
	}
	return out, nil
}
