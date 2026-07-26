package catalog

import (
	"database/sql"
	"os"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

var emailRE = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

// noisyEmail reports addresses that should not become CRM catalog persons.
func noisyEmail(email string) bool {
	e := NormalizeEmail(email)
	if e == "" || !strings.Contains(e, "@") {
		return true
	}
	local, domain, ok := strings.Cut(e, "@")
	if !ok {
		return true
	}
	noiseLocal := []string{
		"noreply", "no-reply", "donotreply", "mailer-daemon", "postmaster",
		"bounce", "notifications", "newsletter", "unsubscribe",
		"root", "admin", "abuse", "webmaster", "hostmaster",
	}
	for _, p := range noiseLocal {
		if local == p || strings.Contains(local, p) {
			return true
		}
	}
	// role / shared mailboxes (keep human dotted names)
	roleExact := map[string]struct{}{
		"info": {}, "alle": {}, "all": {}, "office": {}, "office@": {},
		"wartung": {}, "starface": {}, "gitlab": {}, "pl": {}, "gf": {},
		"umsetzung": {}, "support": {}, "sales": {}, "billing": {},
	}
	if _, ok := roleExact[local]; ok {
		return true
	}
	if strings.HasSuffix(local, "-request") || strings.HasSuffix(local, "-owner") {
		return true
	}
	noiseDomain := []string{
		"marketplace.amazon.", "reply.github.com", "users.noreply.github.com",
		"groups.facebook.com", "googlegroups.com", "yahoogroups.",
		"email.apple.com", "amazonses.com", "mailchimp.com",
		"sendgrid.net", "mailgun.org", "postmarkapp.com",
		"property.booking.com", "transvascular.com",
	}
	for _, p := range noiseDomain {
		if strings.Contains(domain, p) {
			return true
		}
	}
	// random marketplace / tracking locals
	if len(local) >= 16 && !strings.Contains(local, ".") && !strings.Contains(local, "-") {
		if strings.Contains(domain, "amazon.") {
			return true
		}
	}
	return false
}

func parseMABEmails(path string) ([]Entry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	found := emailRE.FindAllString(string(b), -1)
	var out []Entry
	for _, raw := range found {
		em := NormalizeEmail(raw)
		if noisyEmail(em) {
			continue
		}
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
			Notes:   "thunderbird_mab",
		})
	}
	return out, nil
}

func parseGlodaContacts(dbPath string) ([]Entry, error) {
	// read-only URI; immutable=1 helps when WAL/shm are missing
	dsn := "file:" + dbPath + "?mode=ro&_pragma=query_only(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT COALESCE(c.name, ''), i.value
		FROM identities i
		LEFT JOIN contacts c ON c.id = i.contactID
		WHERE lower(i.kind) = 'email' AND i.value IS NOT NULL AND i.value != ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return out, err
		}
		em := NormalizeEmail(value)
		// Gloda sometimes stores "Name <email>" in value
		if m := emailRE.FindString(em); m != "" {
			em = NormalizeEmail(m)
		}
		if noisyEmail(em) {
			continue
		}
		display := strings.TrimSpace(name)
		// strip wrapping quotes / email leftovers
		display = strings.Trim(display, `"'`)
		if idx := strings.Index(display, "<"); idx > 0 {
			display = strings.TrimSpace(display[:idx])
		}
		first, last := "", ""
		if display != "" {
			first, last = SplitDisplayName(display)
		}
		org, zone, role := classifyMailIdentity(display, em)
		out = append(out, Entry{
			ID:      EntryID("person", em, display),
			Kind:    "person",
			Name:    display,
			First:   first,
			Last:    last,
			Emails:  []string{em},
			Org:     org,
			Sources: []string{dbPath},
			Zone:    zone,
			Role:    role,
			Approve: false,
			Status:  "new",
			Notes:   "thunderbird_gloda",
		})
	}
	return out, rows.Err()
}

func classifyMailIdentity(name, email string) (org, zone, role string) {
	em := NormalizeEmail(email)
	_, domain, _ := strings.Cut(em, "@")
	switch {
	case domain == "wheregroup.com" || strings.Contains(strings.ToLower(name), "wheregroup"):
		return "WhereGroup", "warm", "work"
	case domain == "produktor.io" || domain == "eslider.de" || strings.HasSuffix(domain, ".produktor.io"):
		return "produktor.io", "hot", "work"
	case domain == "dyvenia.com":
		return "Dyvenia", "warm", "work"
	case domain == "immowelt.de" || domain == "immowelt.com":
		return "Immowelt", "warm", "work"
	case strings.HasSuffix(domain, ".de") && looksPublicSector(domain):
		return domain, "warm", "work"
	default:
		return "", "private", "unknown"
	}
}

func looksPublicSector(domain string) bool {
	d := strings.ToLower(domain)
	hints := []string{
		"stadt-", "stadt.", "kreis-", "gemeinde", "landkreis",
		"bund.", "bundes", "lvermgeo", "rvr.", "eba.",
	}
	for _, h := range hints {
		if strings.Contains(d, h) {
			return true
		}
	}
	return false
}
