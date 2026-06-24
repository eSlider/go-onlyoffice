package preview

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// ContactMarkdown formats a CRM contact or company for preview.
func ContactMarkdown(m map[string]any) string {
	name := str(m, "displayName")
	if name == "" {
		name = strings.TrimSpace(str(m, "firstName") + " " + str(m, "lastName"))
	}
	kind := "Person"
	if boolVal(m, "isCompany") {
		kind = "Company"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "**Type:** %s\n\n", kind)
	if about := str(m, "about"); about != "" {
		fmt.Fprintf(&b, "## About\n\n%s\n\n", about)
	}
	appendInfoList(&b, m)
	return strings.TrimSpace(b.String()) + "\n"
}

// OpportunityMarkdown formats a CRM deal for preview.
func OpportunityMarkdown(m map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", str(m, "title"))
	if v := m["bidValue"]; v != nil {
		fmt.Fprintf(&b, "**Value:** %v %s\n\n", v, str(m, "bidCurrency"))
	}
	if d := str(m, "description"); d != "" {
		fmt.Fprintf(&b, "## Description\n\n%s\n\n", d)
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// MailMarkdown formats a mail message for preview.
func MailMarkdown(m map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", str(m, "subject"))
	fmt.Fprintf(&b, "**From:** %s\n\n", str(m, "from"))
	if to := str(m, "to"); to != "" {
		fmt.Fprintf(&b, "**To:** %s\n\n", to)
	}
	body := str(m, "body")
	if body == "" {
		body = str(m, "htmlBody")
	}
	body = stripHTML(body)
	if body != "" {
		fmt.Fprintf(&b, "## Body\n\n%s\n\n", body)
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// EventMarkdown formats a calendar event for preview.
func EventMarkdown(m map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", str(m, "title"))
	fmt.Fprintf(&b, "**Start:** %s\n\n", str(m, "start"))
	fmt.Fprintf(&b, "**End:** %s\n\n", str(m, "end"))
	if d := str(m, "description"); d != "" {
		fmt.Fprintf(&b, "## Description\n\n%s\n\n", d)
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// TaskMarkdown formats a project or CRM task for preview.
func TaskMarkdown(m map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", str(m, "title"))
	if s := str(m, "status"); s != "" {
		fmt.Fprintf(&b, "**Status:** %s\n\n", s)
	}
	if d := str(m, "deadline"); d != "" {
		fmt.Fprintf(&b, "**Deadline:** %s\n\n", d)
	}
	if desc := str(m, "description"); desc != "" {
		fmt.Fprintf(&b, "## Description\n\n%s\n\n", desc)
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// ProjectMarkdown formats a project for preview.
func ProjectMarkdown(m map[string]any) string {
	var b strings.Builder
	title := str(m, "title")
	if title == "" {
		title = str(m, "name")
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	if s := str(m, "status"); s != "" {
		fmt.Fprintf(&b, "**Status:** %s\n\n", s)
	}
	if d := str(m, "description"); d != "" {
		fmt.Fprintf(&b, "## Description\n\n%s\n\n", d)
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// EntityMarkdown picks a formatter based on item kind.
func EntityMarkdown(kind string, m map[string]any) string {
	switch kind {
	case "contact", "person", "company":
		return ContactMarkdown(m)
	case "opportunity":
		return OpportunityMarkdown(m)
	case "mail":
		return MailMarkdown(m)
	case "event":
		return EventMarkdown(m)
	case "task", "crm_task":
		return TaskMarkdown(m)
	case "project":
		return ProjectMarkdown(m)
	default:
		return mapToMarkdown(m)
	}
}

func mapToMarkdown(m map[string]any) string {
	var b strings.Builder
	b.WriteString("# Details\n\n")
	for k, v := range m {
		fmt.Fprintf(&b, "- **%s:** %v\n", k, v)
	}
	return b.String()
}

func appendInfoList(b *strings.Builder, m map[string]any) {
	if infos, ok := m["contactInfos"].([]any); ok && len(infos) > 0 {
		b.WriteString("## Contact info\n\n")
		for _, raw := range infos {
			if row, ok := raw.(map[string]any); ok {
				fmt.Fprintf(b, "- %s: %s\n", str(row, "infoType"), str(row, "data"))
			}
		}
		b.WriteString("\n")
	}
}

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}

func str(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	if m[key] == nil {
		return ""
	}
	return fmt.Sprint(m[key])
}

func boolVal(m map[string]any, key string) bool {
	v, ok := m[key].(bool)
	return ok && v
}
