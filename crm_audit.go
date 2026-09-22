package onlyoffice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// OpportunityAudit is one CRM opportunity with its resource counts and a coarse
// class, for hygiene reporting (see `oo crm audit`).
type OpportunityAudit struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Created     string `json:"created,omitempty"`
	Files       int    `json:"files"`
	OpenTasks   int    `json:"open_tasks"`
	ClosedTasks int    `json:"closed_tasks"`
	Members     int    `json:"members"`
	GroupKey    string `json:"group_key,omitempty"`
	Class       string `json:"class"` // ok | dup | empty | junk-title
}

// AuditOpportunities lists every opportunity with file/task/member counts and a
// coarse class. The classification is generic and rule-free:
//
//	dup        — another opportunity shares the same title key
//	empty      — no files, tasks or members
//	junk-title — title is not of the "Role @ Company" shape
//	ok         — everything else
//
// Callers that need stricter business rules can post-process the result.
func (c *Client) AuditOpportunities(ctx context.Context) ([]OpportunityAudit, error) {
	deals, err := c.ListAllOpportunities(ctx)
	if err != nil {
		return nil, err
	}
	tasks, _, err := c.ListCRMTasks(ctx, 5000, 0)
	if err != nil {
		return nil, err
	}
	open, closed := taskCountsByOpportunity(tasks)

	out := make([]OpportunityAudit, 0, len(deals))
	for _, row := range deals {
		id := auditID(row["id"])
		if id == 0 {
			continue
		}
		title := auditStr(row["title"])
		files := 0
		if fl, ferr := c.ListOpportunityFiles(ctx, strconv.FormatInt(id, 10)); ferr == nil {
			files = len(fl)
		}
		key := strconv.FormatInt(id, 10)
		out = append(out, OpportunityAudit{
			ID:          id,
			Title:       title,
			Created:     auditStr(row["created"]),
			Files:       files,
			OpenTasks:   open[key],
			ClosedTasks: closed[key],
			Members:     len(OpportunityMembers(row)),
			GroupKey:    DealTitleKey(title, false),
		})
	}

	groupCount := map[string]int{}
	for _, a := range out {
		groupCount[a.GroupKey]++
	}
	for i := range out {
		a := &out[i]
		switch {
		case groupCount[a.GroupKey] > 1:
			a.Class = "dup"
		case a.Files == 0 && a.OpenTasks == 0 && a.ClosedTasks == 0 && a.Members == 0:
			a.Class = "empty"
		case !strings.Contains(a.Title, "@") || strings.HasPrefix(strings.TrimSpace(a.Title), "@"):
			a.Class = "junk-title"
		default:
			a.Class = "ok"
		}
	}
	return out, nil
}

// taskCountsByOpportunity buckets CRM task statuses per opportunity id.
func taskCountsByOpportunity(tasks []map[string]any) (open, closed map[string]int) {
	open, closed = map[string]int{}, map[string]int{}
	for _, t := range tasks {
		ent, ok := t["entity"].(map[string]any)
		if !ok || auditStr(ent["entityType"]) != "opportunity" {
			continue
		}
		eid := auditStr(ent["entityId"])
		status := strings.ToLower(auditStr(t["status"]))
		if status == "2" || status == "closed" {
			closed[eid]++
		} else {
			open[eid]++
		}
	}
	return open, closed
}

func auditID(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int:
		return int64(x)
	case int64:
		return x
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(x)), 10, 64)
		return n
	}
}

func auditStr(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
