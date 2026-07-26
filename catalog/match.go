package catalog

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

// MatchAgainstOO sets status/oo_id by email then normalized name.
func MatchAgainstOO(ctx context.Context, client *onlyoffice.Client, doc *Document) error {
	all, err := client.ListAllContacts(ctx)
	if err != nil {
		return err
	}

	byEmail := map[string]map[string]any{}
	byPersonName := map[string]map[string]any{}
	byCompanyName := map[string]map[string]any{}

	for _, c := range all {
		id := contactIDString(c)
		isCo, _ := c["isCompany"].(bool)
		for _, em := range contactEmails(c) {
			byEmail[NormalizeEmail(em)] = c
		}
		if isCo {
			key := onlyoffice.CompanyGroupingKey(fmt.Sprint(c["displayName"]))
			if key != "" {
				byCompanyName[key] = c
			}
			continue
		}
		first := strings.TrimSpace(fmt.Sprint(c["firstName"]))
		last := strings.TrimSpace(fmt.Sprint(c["lastName"]))
		key := NormalizeName(first + " " + last)
		if key == "" {
			key = NormalizeName(fmt.Sprint(c["displayName"]))
		}
		if key != "" {
			byPersonName[key] = c
		}
		_ = id
	}

	for i := range doc.Entries {
		e := &doc.Entries[i]
		// preserve approve
		matched := false
		conflict := false
		var oo map[string]any

		for _, em := range e.Emails {
			if c, ok := byEmail[NormalizeEmail(em)]; ok {
				if oo != nil && contactIDString(oo) != contactIDString(c) {
					conflict = true
				}
				oo = c
				matched = true
			}
		}
		if !matched {
			if e.Kind == "company" {
				if c, ok := byCompanyName[onlyoffice.CompanyGroupingKey(e.Name)]; ok {
					oo = c
					matched = true
				}
			} else {
				key := NormalizeName(strings.TrimSpace(e.First + " " + e.Last))
				if key == "" {
					key = NormalizeName(e.Name)
				}
				if c, ok := byPersonName[key]; ok {
					oo = c
					matched = true
				}
			}
		}

		if conflict {
			e.Status = "conflict"
			e.Notes = strings.TrimSpace(e.Notes + " email_matches_multiple_oo")
			if oo != nil {
				e.OOID = contactIDString(oo)
			}
			continue
		}
		if matched && oo != nil {
			e.Status = "exists"
			e.OOID = contactIDString(oo)
			continue
		}
		// Keep a previously applied oo_id (list payloads often omit emails,
		// so email match can miss persons that already exist in CRM).
		if strings.TrimSpace(e.OOID) != "" {
			e.Status = "exists"
			continue
		}
		e.Status = "new"
		e.OOID = ""
	}
	return nil
}

func contactIDString(c map[string]any) string {
	switch v := c["id"].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func contactEmails(c map[string]any) []string {
	var out []string
	if em := strings.TrimSpace(fmt.Sprint(c["email"])); em != "" && em != "<nil>" {
		out = append(out, em)
	}
	if em := strings.TrimSpace(fmt.Sprint(c["primaryEmail"])); em != "" && em != "<nil>" {
		out = append(out, em)
	}
	for _, row := range onlyoffice.ContactInfoRows(c) {
		t := strings.ToLower(fmt.Sprint(row["infoType"]))
		if t != "email" {
			continue
		}
		data := strings.TrimSpace(fmt.Sprint(row["data"]))
		if data != "" {
			out = append(out, data)
		}
	}
	return uniqueEmails(out)
}
