// Package applications syncs job application README trees into OnlyOffice CRM
// (ported from cv/bin/office/sync-applications.py).
package applications

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

var docExt = map[string]bool{".pdf": true, ".docx": true, ".xlsx": true, ".doc": true, ".xls": true}

type RecruiterInfo struct {
	First, Last, JobTitle, Company, Email, Phone, LinkedIn string
}

type Data struct {
	Path         string
	Folder       string
	Position     string
	Company      string
	Location     string
	Salary       string
	SalaryValue  float64
	Contract     string
	Source       string
	Link         string
	Recruiter    RecruiterInfo
	Summary      string
	Documents    []string
}

func Discover(base string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "pdfs" {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Base(path) != "README.md" {
			return nil
		}
		dir := filepath.Dir(path)
		if filepath.Clean(dir) == filepath.Clean(base) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	return out, err
}

func ParseReadme(path string) (Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Data{}, err
	}
	text := string(b)
	app := Data{Path: path}
	app.Folder = relFolder(path)
	app.Position = extract(text, `(?i)\*\*Position\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	app.Company = extract(text, `(?i)\*\*Company\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	if app.Company != "" {
		app.Company = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(app.Company, "$1")
		app.Company = regexp.MustCompile(`\s*\(.*$`).ReplaceAllString(strings.TrimSpace(app.Company), "")
	}
	app.Location = extract(text, `(?i)\*\*Location\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	app.Salary = extract(text, `(?i)\*\*(?:Salary|Gehalt|Rate)\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	app.SalaryValue = parseSalary(app.Salary)
	app.Contract = extract(text, `(?i)\*\*Contract\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	app.Link = extract(text, `(?i)\*\*Link\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	if m := regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`).FindStringSubmatch(app.Link); len(m) > 2 {
		app.Link = m[2]
	}
	app.Source = extract(text, `(?i)\*\*Source\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	app.Recruiter = parseRecruiter(text)
	if app.Source == "" {
		app.Source = filepath.Base(filepath.Dir(filepath.Dir(path)))
	}
	if app.Company == "" || strings.EqualFold(app.Company, "undisclosed") || app.Company == "?" || strings.EqualFold(app.Company, "tbd") {
		parts := strings.Split(filepath.Base(filepath.Dir(path)), "-")
		if len(parts) >= 2 {
			app.Company = titleWord(parts[0])
		}
	}
	app.Summary = buildSummary(app, text)
	app.Documents = discoverDocs(filepath.Dir(path))
	return app, nil
}

func relFolder(readmePath string) string {
	// Python: path.parent.relative_to(path.parent.parent.parent.parent) — approximate: last 3 parts
	p := filepath.Clean(readmePath)
	parts := strings.Split(p, string(filepath.Separator))
	if len(parts) >= 3 {
		return filepath.Join(parts[len(parts)-3:]...)
	}
	return filepath.Dir(readmePath)
}

func extract(text, pat string) string {
	re := regexp.MustCompile(pat)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func parseSalary(raw string) float64 {
	if raw == "" {
		return 0
	}
	cleaned := strings.ReplaceAll(strings.ReplaceAll(raw, ",", ""), ".", "")
	if m := regexp.MustCompile(`(\d{4,6})`).FindAllStringSubmatch(cleaned, -1); len(m) > 0 {
		var maxv int
		for _, x := range m {
			n, _ := strconv.Atoi(x[1])
			if n > maxv {
				maxv = n
			}
		}
		return float64(maxv)
	}
	if m := regexp.MustCompile(`(?i)(\d{2,4})\s*/\s*day`).FindStringSubmatch(raw); len(m) > 1 {
		n, _ := strconv.ParseFloat(m[1], 64)
		return n * 220
	}
	return 0
}

func parseRecruiter(text string) RecruiterInfo {
	var r RecruiterInfo
	raw := extract(text, `(?i)\*\*(?:Recruiter|Consultant)\*\*:\s*(.+?)(?:\s{2,}|\n)`)
	if raw == "" {
		return r
	}
	if m := regexp.MustCompile(`<([^>]+@[^>]+)>`).FindStringSubmatch(raw); len(m) > 1 {
		r.Email = strings.TrimSpace(m[1])
		if i := strings.Index(raw, "<"); i >= 0 {
			raw = strings.TrimSpace(raw[:i])
		}
		raw = strings.TrimRight(raw, " —–-")
	}
	if m := regexp.MustCompile(`(?i)\s+@\s+(.+)$`).FindStringSubmatch(raw); len(m) > 1 {
		r.Company = strings.TrimSpace(m[1])
		if i := strings.Index(raw, m[0]); i >= 0 {
			raw = strings.TrimSpace(raw[:i])
		}
	}
	if m := regexp.MustCompile(`,?\s*\[([^\]]+)\]\([^)]+\)`).FindStringSubmatch(raw); len(m) > 1 {
		r.Company = regexp.MustCompile(`\s*\(.*$`).ReplaceAllString(strings.TrimSpace(m[1]), "")
		if i := strings.Index(raw, m[0]); i >= 0 {
			raw = strings.TrimSpace(raw[:i])
		}
	}
	parts := strings.SplitN(raw, ",", 2)
	namePart := strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		r.JobTitle = strings.TrimSpace(parts[1])
	}
	toks := strings.Fields(namePart)
	if len(toks) >= 2 {
		r.First = toks[0]
		r.Last = strings.Join(toks[1:], " ")
	} else if len(toks) == 1 {
		r.First = toks[0]
	}
	if m := regexp.MustCompile(`(?i)\*\*?(?:Telefon|Phone)\*\*?:\s*([\d+\-\s]+)`).FindStringSubmatch(text); len(m) > 1 {
		r.Phone = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`(?i)\*\*LinkedIn\*\*:\s*(https://[^\s]+)`).FindStringSubmatch(text); len(m) > 1 {
		r.LinkedIn = strings.TrimSpace(m[1])
	}
	return r
}

func buildSummary(app Data, text string) string {
	var lines []string
	lines = append(lines, "Position: "+app.Position)
	lines = append(lines, "Company: "+app.Company)
	if app.Location != "" {
		lines = append(lines, "Location: "+app.Location)
	}
	if app.Salary != "" {
		lines = append(lines, "Salary: "+app.Salary)
	}
	if app.Contract != "" {
		lines = append(lines, "Contract: "+app.Contract)
	}
	if app.Source != "" {
		lines = append(lines, "Source: "+app.Source)
	}
	if app.Link != "" {
		lines = append(lines, "Link: "+app.Link)
	}
	lines = append(lines, "Folder: "+app.Folder)
	if m := regexp.MustCompile(`(?i)## Application Status\s*\n((?:[-*\[\]xX ].+\n?)+)`).FindStringSubmatch(text); len(m) > 1 {
		lines = append(lines, "", "Status:", strings.TrimSpace(m[1]))
	}
	if m := regexp.MustCompile(`(?i)## (?:Fit Assessment|Match|Candidate Fit)\s*\n([\s\S]+?)(?=\n## |\z)`).FindStringSubmatch(text); len(m) > 1 {
		ft := strings.TrimSpace(m[1])
		if len(ft) > 500 {
			ft = ft[:500] + "..."
		}
		lines = append(lines, "", "Fit Assessment:", ft)
	}
	return strings.Join(lines, "\n")
}

func discoverDocs(dir string) []string {
	var docs []string
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if docExt[ext] {
			docs = append(docs, filepath.Join(dir, e.Name()))
		}
	}
	return docs
}

type Stats struct {
	Companies, Persons, Deals, Tasks, Docs, Notes int
}

// Sync mirrors Python sync_to_crm.
func Sync(ctx context.Context, client *onlyoffice.Client, apps []Data, dryRun, verbose bool) Stats {
	var st Stats
	deadline := time.Now().Add(14 * 24 * time.Hour).Format("2006-01-02T15:04:05")
	const stageInitial = 1
	for _, app := range apps {
		dealTitle := app.Position
		if app.Company != "" {
			dealTitle = app.Position + " @ " + app.Company
		}
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println(" ", app.Folder)
		fmt.Println(" ", dealTitle)
		if verbose {
			// minimal verbose
			if app.Salary != "" {
				fmt.Printf("  Salary: %s (parsed: %.0f)\n", app.Salary, app.SalaryValue)
			}
		}
		action := "would create"
		if !dryRun {
			action = "creating"
		}
		companyID := 0
		if app.Company != "" {
			if !dryRun {
				if ex, _ := client.FindCompany(ctx, app.Company); ex != nil {
					companyID = int(flexID(ex["id"]))
					fmt.Printf("  [company] found: [%d] %s\n", companyID, app.Company)
				} else {
					co, err := client.CreateCompany(ctx, app.Company)
					if err != nil {
						fmt.Printf("  [company] ERR: %v\n", err)
					} else {
						companyID = int(flexID(co["id"]))
						fmt.Printf("  [company] created: [%d] %s\n", companyID, app.Company)
						if app.Link != "" {
							_, _ = client.AddContactInfo(ctx, strconv.Itoa(companyID), "Website", app.Link, "Work", false)
						}
					}
				}
			} else {
				fmt.Printf("  [company] %s: %s\n", action, app.Company)
			}
			st.Companies++
		}
		personID := 0
		r := app.Recruiter
		if r.First != "" && r.Last != "" {
			if !dryRun {
				if ex, _ := client.FindPerson(ctx, r.First, r.Last); ex != nil {
					personID = int(flexID(ex["id"]))
					fmt.Printf("  [person]  found: [%d] %s %s\n", personID, r.First, r.Last)
				} else {
					p, err := client.CreatePerson(ctx, r.First, r.Last, companyID, r.JobTitle, "")
					if err != nil {
						fmt.Printf("  [person]  ERR: %v\n", err)
					} else {
						personID = int(flexID(p["id"]))
						fmt.Printf("  [person]  created: [%d] %s %s\n", personID, r.First, r.Last)
						if r.Email != "" {
							_, _ = client.AddContactInfo(ctx, strconv.Itoa(personID), "Email", r.Email, "Work", true)
						}
						if r.Phone != "" {
							_, _ = client.AddContactInfo(ctx, strconv.Itoa(personID), "Phone", r.Phone, "Work", false)
						}
						if r.LinkedIn != "" {
							_, _ = client.AddContactInfo(ctx, strconv.Itoa(personID), "LinkedIn", r.LinkedIn, "Work", false)
						}
					}
				}
			} else {
				fmt.Printf("  [person]  %s: %s %s\n", action, r.First, r.Last)
			}
			st.Persons++
		}
		var dealID int
		if dryRun {
			fmt.Printf("  [deal]    %s: %s\n", action, dealTitle)
		} else {
			if ex := findExistingOpp(ctx, client, dealTitle); ex != nil {
				dealID = int(flexID(ex["id"]))
				fmt.Printf("  [deal]    found: [%d] %s\n", dealID, dealTitle)
			} else {
				d, err := client.CreateOpportunity(ctx, dealTitle, stageInitial, "", "EUR", "", app.SalaryValue)
				if err != nil {
					fmt.Printf("  [deal]    ERR: %v\n", err)
				} else {
					dealID = int(flexID(d["id"]))
					fmt.Printf("  [deal]    created: [%d] %s\n", dealID, dealTitle)
				}
			}
		}
		st.Deals++
		if !dryRun && dealID != 0 {
			if companyID != 0 {
				_, _ = client.AddOpportunityMember(ctx, strconv.Itoa(dealID), strconv.Itoa(companyID))
			}
			if personID != 0 {
				_, _ = client.AddOpportunityMember(ctx, strconv.Itoa(dealID), strconv.Itoa(personID))
			}
		}
		if app.Summary != "" {
			if dryRun {
				fmt.Printf("  [note]    %s: application summary\n", action)
			} else if dealID != 0 {
				if _, err := client.AddHistoryNote(ctx, "opportunity", dealID, app.Summary, 0); err == nil {
					fmt.Println("  [note]    added application summary")
					st.Notes++
				}
			}
		}
		titles := []string{"Write application", "Apply application", "Send CV"}
		if dryRun {
			for _, t := range titles {
				fmt.Printf("  [task]    %s: %s\n", action, t)
			}
		} else if dealID != 0 {
			for _, t := range titles {
				if _, err := client.CreateCRMTask(ctx, t, deadline, 2, 0, "opportunity", dealID, ""); err == nil {
					fmt.Printf("  [task]    created: %s\n", t)
					st.Tasks++
				}
			}
		}
		for _, doc := range app.Documents {
			if dryRun {
				fmt.Printf("  [doc]     %s: %s\n", action, filepath.Base(doc))
			} else if dealID != 0 {
				if _, err := client.UploadOpportunityFile(ctx, strconv.Itoa(dealID), doc); err == nil {
					fmt.Printf("  [doc]     uploaded: %s\n", filepath.Base(doc))
					st.Docs++
				}
			}
		}
	}
	mode := "DRY RUN"
	if !dryRun {
		mode = "APPLIED"
	}
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("  %s: deals=%d companies=%d persons=%d notes=%d tasks=%d docs=%d\n",
		mode, st.Deals, st.Companies, st.Persons, st.Notes, st.Tasks, st.Docs)
	return st
}

func findExistingOpp(ctx context.Context, c *onlyoffice.Client, title string) map[string]interface{} {
	deals, total, _ := c.ListOpportunities(ctx, 100, 0)
	for _, d := range deals {
		if strings.TrimSpace(fmt.Sprint(d["title"])) == strings.TrimSpace(title) {
			return d
		}
	}
	if total > 100 {
		deals2, _, _ := c.ListOpportunities(ctx, 100, 100)
		for _, d := range deals2 {
			if strings.TrimSpace(fmt.Sprint(d["title"])) == strings.TrimSpace(title) {
				return d
			}
		}
	}
	return nil
}

func titleWord(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func flexID(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(x), 64)
		return f
	}
}
