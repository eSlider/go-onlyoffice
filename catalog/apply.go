package catalog

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

// ApplyResult summarizes one apply pass.
type ApplyResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
	DryRun  bool     `json:"dry_run"`
}

// ApplyApproved creates/updates OO contacts for entries with approve=true.
// Companies are applied before persons so Org links resolve in the same run.
func ApplyApproved(ctx context.Context, client *onlyoffice.Client, doc *Document, dryRun bool) (*ApplyResult, error) {
	res := &ApplyResult{DryRun: dryRun}
	order := make([]int, 0, len(doc.Entries))
	for i, e := range doc.Entries {
		if e.Kind == "company" {
			order = append(order, i)
		}
	}
	for i, e := range doc.Entries {
		if e.Kind != "company" {
			order = append(order, i)
		}
	}
	for _, i := range order {
		e := &doc.Entries[i]
		if !e.Approve {
			res.Skipped++
			continue
		}
		if e.Status == "conflict" {
			res.Skipped++
			res.Errors = append(res.Errors, e.ID+": conflict — resolve manually")
			continue
		}
		if dryRun {
			if e.OOID != "" || e.Status == "exists" {
				res.Updated++
			} else {
				res.Created++
			}
			continue
		}
		created, err := applyOne(ctx, client, e)
		if err != nil {
			res.Errors = append(res.Errors, e.ID+": "+err.Error())
			continue
		}
		if created {
			res.Created++
		} else {
			res.Updated++
		}
	}
	return res, nil
}

func applyOne(ctx context.Context, client *onlyoffice.Client, e *Entry) (created bool, err error) {
	switch e.Kind {
	case "company":
		return applyCompany(ctx, client, e)
	case "person":
		return applyPerson(ctx, client, e)
	default:
		return false, fmt.Errorf("unknown kind %q", e.Kind)
	}
}

func applyCompany(ctx context.Context, client *onlyoffice.Client, e *Entry) (bool, error) {
	name := strings.TrimSpace(e.Name)
	if name == "" {
		return false, fmt.Errorf("company missing name")
	}
	var co map[string]any
	var err error
	if e.OOID != "" {
		co, err = client.GetContact(ctx, e.OOID)
	} else {
		co, err = client.FindCompany(ctx, name)
	}
	if err != nil {
		return false, err
	}
	created := false
	if co == nil {
		co, err = client.CreateCompany(ctx, name)
		if err != nil {
			return false, err
		}
		created = true
	}
	id := contactIDString(co)
	e.OOID = id
	if err := ensureContactInfos(ctx, client, id, e); err != nil {
		return created, err
	}
	e.Status = "applied"
	if created {
		e.Notes = "created"
	} else {
		e.Notes = "updated"
	}
	return created, nil
}

func applyPerson(ctx context.Context, client *onlyoffice.Client, e *Entry) (bool, error) {
	org := strings.TrimSpace(e.Org)
	first, last := CleanPersonNames(e.First, e.Last, e.Name, org, e.Emails)
	e.First, e.Last = first, last
	if first == "" {
		return false, fmt.Errorf("person missing name")
	}
	e.Name = strings.TrimSpace(first + " " + strings.Trim(last, "-"))

	var p map[string]any
	var err error
	if e.OOID != "" {
		p, err = client.GetContact(ctx, e.OOID)
	} else if len(e.Emails) > 0 {
		p, err = client.FindPersonByEmail(ctx, e.Emails[0])
	} else {
		p, err = client.FindPerson(ctx, first, last)
	}
	if err != nil {
		return false, err
	}
	created := false
	companyID := 0
	if org != "" {
		if co, ferr := client.FindCompany(ctx, org); ferr == nil && co != nil {
			companyID, _ = strconv.Atoi(contactIDString(co))
		}
	}
	if p == nil {
		about := ""
		if org != "" {
			about = "org: " + org
		}
		p, err = client.CreatePerson(ctx, first, last, companyID, "", about)
		if err != nil {
			return false, err
		}
		created = true
	} else {
		// Repair names + ensure company link (never encode company in lastName).
		id := contactIDString(p)
		if _, err := client.UpdatePerson(ctx, id, first, last, companyID, "", ""); err != nil {
			return false, fmt.Errorf("update person %s: %w", id, err)
		}
	}
	id := contactIDString(p)
	e.OOID = id
	if err := ensureContactInfos(ctx, client, id, e); err != nil {
		return created, err
	}
	e.Status = "applied"
	if created {
		e.Notes = "created"
	} else {
		e.Notes = "updated"
	}
	return created, nil
}

func ensureContactInfos(ctx context.Context, client *onlyoffice.Client, contactID string, e *Entry) error {
	existing, err := client.GetContact(ctx, contactID)
	if err != nil {
		return err
	}
	for i, em := range e.Emails {
		if onlyoffice.HasContactInfo(existing, "Email", em) {
			continue
		}
		if _, err := client.AddContactInfo(ctx, contactID, "Email", em, "Work", i == 0); err != nil {
			return fmt.Errorf("add email %s: %w", em, err)
		}
	}
	for _, ph := range e.Phones {
		if onlyoffice.HasContactInfo(existing, "Phone", ph) {
			continue
		}
		if _, err := client.AddContactInfo(ctx, contactID, "Phone", ph, "Work", false); err != nil {
			return fmt.Errorf("add phone %s: %w", ph, err)
		}
	}
	return nil
}
