package catalog

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	parenSuffixRE   = regexp.MustCompile(`(?i)\s*[\(\[\{][^)\]\}]*[\)\]\}]\s*$`)
	dashCompanyRE   = regexp.MustCompile(`(?i)\s+[-–—]\s+[A-Za-z0-9].*$`)
	emailLocalRE    = regexp.MustCompile(`(?i)^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	nonNameTokenRE  = regexp.MustCompile(`[^a-zA-ZÀ-öø-ÿĀ-ž0-9'’.\-]+`)
)

// CleanPersonNames strips company annotations from display names and fills
// first/last from email local-part when the source used an address as the name.
// Company affiliation belongs on Org / CRM companyId — never in LastName.
func CleanPersonNames(first, last, display, org string, emails []string) (cleanFirst, cleanLast string) {
	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)
	display = strings.TrimSpace(display)
	org = strings.TrimSpace(org)

	if looksLikeEmail(first) {
		ef, el := GuessNameFromEmail(first)
		first, last = ef, el
	}
	if looksLikeEmail(display) && first == "" && last == "" {
		display = ""
	}

	if first == "" && last == "" && display != "" {
		first, last = SplitDisplayName(display)
	}

	first = stripCompanyAnnotation(first, org)
	last = stripCompanyAnnotation(last, org)

	// "Schaefermeyer - WhereGroup" / "Thomsen (WhereGroup)" landed in last.
	last = stripCompanyAnnotation(last, org)
	if i := strings.IndexAny(first, "(["); i > 0 {
		first = strings.TrimSpace(first[:i])
	}
	// Entire last name is just the company (e.g. last="WhereGroup").
	if org != "" && personLastIsOrg(last, org) {
		last = ""
	}

	if (first == "" || looksLikeEmail(first)) && len(emails) > 0 {
		ef, el := GuessNameFromEmail(emails[0])
		if first == "" || looksLikeEmail(first) {
			first = ef
		}
		if last == "" || last == "-" {
			last = el
		}
	}

	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)
	if last == "" {
		last = "-"
	}
	return first, last
}

func stripCompanyAnnotation(s, org string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = parenSuffixRE.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = dashCompanyRE.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if org != "" {
		for _, sep := range []string{" - ", " – ", " — ", " / "} {
			if i := strings.LastIndex(strings.ToLower(s), strings.ToLower(sep+org)); i >= 0 {
				s = strings.TrimSpace(s[:i])
			}
		}
		suf := " (" + org + ")"
		if strings.HasSuffix(strings.ToLower(s), strings.ToLower(suf)) {
			s = strings.TrimSpace(s[:len(s)-len(suf)])
		}
	}
	return strings.TrimSpace(s)
}

func personLastIsOrg(last, org string) bool {
	last = NormalizeName(last)
	org = NormalizeName(org)
	if last == "" || org == "" {
		return false
	}
	if last == org {
		return true
	}
	// "WhereGroup" vs "wheregroup gmbh & co. kg"
	return strings.HasPrefix(org, last+" ") || strings.HasPrefix(org, last+",")
}

func looksLikeEmail(s string) bool {
	return emailLocalRE.MatchString(strings.TrimSpace(s))
}

// GuessNameFromEmail turns local@domain into Title-Case first/last when the
// local part looks like first.last / first_last / first-last.
func GuessNameFromEmail(email string) (first, last string) {
	email = NormalizeEmail(email)
	local, _, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return "", ""
	}
	local = strings.Split(local, "+")[0]
	parts := strings.FieldsFunc(local, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return titleToken(local), ""
	}
	if len(parts) == 1 {
		return titleToken(parts[0]), ""
	}
	return titleToken(parts[0]), titleToken(strings.Join(parts[1:], " "))
}

func titleToken(s string) string {
	s = nonNameTokenRE.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToTitle(runes[0])
	return string(runes)
}

// FormatProjectTitle builds "CC | Company | Title" (spaces around |).
// Country should be a short code (DE, TF, UA, …). Empty segments are dropped.
func FormatProjectTitle(country, company, title string) string {
	parts := make([]string, 0, 3)
	for _, p := range []string{country, company, title} {
		p = strings.TrimSpace(p)
		p = strings.ReplaceAll(p, "|", "/")
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " | ")
}
