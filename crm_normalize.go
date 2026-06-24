package onlyoffice

import (
	"regexp"
	"strings"
)

var multiSpace = regexp.MustCompile(`\s+`)

// sloganSeparators split a company name from a trailing tagline/slogan.
var sloganSeparators = []string{" — ", " – ", " - ", "—", "–"}

// StripSloganSuffix returns the part before an em/en dash tagline, e.g.
// "Affirm — Fraud Engineering" → "Affirm".
func StripSloganSuffix(s string) string {
	s = strings.TrimSpace(s)
	for _, sep := range sloganSeparators {
		if i := strings.Index(s, sep); i > 0 {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

// CompanyGroupingKey normalizes a company name for dedupe (ignores slogans).
func CompanyGroupingKey(s string) string {
	return NormalizeCompanyName(StripSloganSuffix(s))
}

// NormalizeCompanyName lowercases and collapses whitespace for grouping.
func NormalizeCompanyName(s string) string {
	return collapseKey(s)
}

// NormalizePersonKey builds a grouping key from first and last name.
func NormalizePersonKey(first, last string) string {
	return collapseKey(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
}

// NormalizeOpportunityTitle lowercases and trims a deal title for exact dedupe.
func NormalizeOpportunityTitle(s string) string {
	return collapseKey(s)
}

// StripCompanySuffix removes a trailing " @ Company" segment when present.
func StripCompanySuffix(title string) string {
	title = strings.TrimSpace(title)
	if i := strings.LastIndex(title, " @ "); i >= 0 {
		return strings.TrimSpace(title[:i])
	}
	return title
}

// FixDealTitle strips a leading @, normalizes separator spacing, and collapses
// empty-position titles like " @ 711media" to "711media".
func FixDealTitle(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasPrefix(s, "@") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "@"))
	}
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "@"); i >= 0 {
		left := strings.TrimSpace(s[:i])
		right := strings.TrimSpace(s[i+1:])
		if left == "" && right != "" {
			return right
		}
		if left != "" && right != "" {
			return left + " @ " + right
		}
	}
	return s
}

// ContactInfoKey groups contact info rows by type and normalized value.
func ContactInfoKey(infoType, value string) string {
	return strings.ToLower(strings.TrimSpace(infoType)) + "|" + strings.ToLower(strings.TrimSpace(value))
}

// MemberDisplayKey normalizes a member displayName for duplicate detection.
func MemberDisplayKey(displayName string) string {
	return CompanyGroupingKey(displayName)
}

func collapseKey(s string) string {
	s = strings.TrimSpace(s)
	s = multiSpace.ReplaceAllString(s, " ")
	return strings.ToLower(s)
}

// OpportunityTitlesMatch reports whether two deal titles refer to the same role+company.
func OpportunityTitlesMatch(a, b string) bool {
	return DealTitleKey(a, false) == DealTitleKey(b, false)
}

func DealTitleForApplication(position, company string) string {
	position = strings.TrimSpace(position)
	company = strings.TrimSpace(company)
	if company == "" {
		return position
	}
	if position == "" {
		return company
	}
	return position + " @ " + company
}
