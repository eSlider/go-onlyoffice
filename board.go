package onlyoffice

// Project board (Gantt) upsert from a YAML board file.
//
// A board describes projects, their milestones and tasks by exact title. Sync
// creates only what is missing: existing milestones/tasks (matched by title)
// are left untouched, so the file can be the source of truth for a project
// plan and re-applied safely. Dry-run (apply=false) reports counts without
// writing.
//
// The file format is deliberately small and presentation-free:
//
//	projects:
//	  - id: 42
//	    name: "Example"
//	    milestones:
//	      - title: "Kickoff"
//	        deadline: "2026-01-15"
//	        key: true
//	        tasks:
//	          - title: "Draft"
//	            start: "2026-01-02"
//	            deadline: "2026-01-10"
//	            description: "…"

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Board is a YAML mapping from a project plan onto OnlyOffice milestones/tasks.
type Board struct {
	Projects []BoardProject `yaml:"projects"`
}

// BoardProject is one project with its milestones.
type BoardProject struct {
	ID         int              `yaml:"id"`
	Name       string           `yaml:"name,omitempty"`
	Milestones []BoardMilestone `yaml:"milestones"`
}

// BoardMilestone is a milestone ("key" marks it as a key milestone).
type BoardMilestone struct {
	Title    string      `yaml:"title"`
	Deadline string      `yaml:"deadline"`
	Key      bool        `yaml:"key,omitempty"`
	Tasks    []BoardTask `yaml:"tasks"`
}

// BoardTask is a task inside a milestone. Start falls back to Deadline.
type BoardTask struct {
	Title       string `yaml:"title"`
	Start       string `yaml:"start,omitempty"`
	Deadline    string `yaml:"deadline"`
	Description string `yaml:"description,omitempty"`
}

// BoardSyncResult counts what SyncBoard created or skipped.
type BoardSyncResult struct {
	CreatedMilestones int
	SkippedMilestones int
	CreatedTasks      int
	SkippedTasks      int
	DryRun            bool
}

// LoadBoard reads a board YAML file.
func LoadBoard(path string) (*Board, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseBoard(b)
}

// ParseBoard decodes a board from YAML bytes.
func ParseBoard(data []byte) (*Board, error) {
	var board Board
	if err := yaml.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("board: %w", err)
	}
	if len(board.Projects) == 0 {
		return nil, fmt.Errorf("board: no projects")
	}
	return &board, nil
}

func boardDay(s string) (Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Time{}, err
	}
	return Time(t), nil
}

func boardMilestoneIDs(ms []*Milestone) map[string]int64 {
	out := map[string]int64{}
	for _, m := range ms {
		if m == nil || m.Title == nil || m.ID == nil {
			continue
		}
		out[*m.Title] = *m.ID
	}
	return out
}

func boardTaskTitles(rows []map[string]any) map[string]struct{} {
	out := map[string]struct{}{}
	for _, r := range rows {
		if t, _ := r["title"].(string); t != "" {
			out[t] = struct{}{}
		}
	}
	return out
}

// SyncBoard upserts milestones and tasks by exact title. With apply=false it
// only counts what would be created.
func (c *Client) SyncBoard(ctx context.Context, board *Board, apply bool) (*BoardSyncResult, error) {
	if board == nil || len(board.Projects) == 0 {
		return nil, fmt.Errorf("board: no projects")
	}
	res := &BoardSyncResult{DryRun: !apply}
	for _, p := range board.Projects {
		pid := p.ID
		existing, err := c.GetProjectMilestones(&Project{ID: &pid})
		if err != nil {
			return res, fmt.Errorf("project %d milestones: %w", pid, err)
		}
		haveMS := boardMilestoneIDs(existing)
		tasks, err := c.ListTasks(ctx, strconv.Itoa(pid), "")
		if err != nil {
			return res, fmt.Errorf("project %d tasks: %w", pid, err)
		}
		haveTask := boardTaskTitles(tasks)

		for _, m := range p.Milestones {
			msID, ok := haveMS[m.Title]
			if !ok {
				res.CreatedMilestones++
				if apply {
					dl, err := boardDay(m.Deadline)
					if err != nil {
						return res, fmt.Errorf("milestone %q deadline: %w", m.Title, err)
					}
					created, err := c.CreateMilestone(NewMilestoneRequest{
						ProjectID: pid,
						Title:     m.Title,
						Deadline:  dl,
						IsKey:     m.Key,
					})
					if err != nil {
						return res, fmt.Errorf("create milestone %q: %w", m.Title, err)
					}
					if created.ID != nil {
						msID = *created.ID
					}
					haveMS[m.Title] = msID
				}
			} else {
				res.SkippedMilestones++
			}
			for _, t := range m.Tasks {
				if _, exists := haveTask[t.Title]; exists {
					res.SkippedTasks++
					continue
				}
				res.CreatedTasks++
				if !apply {
					continue
				}
				start := t.Start
				if start == "" {
					start = t.Deadline
				}
				st, err := boardDay(start)
				if err != nil {
					return res, fmt.Errorf("task %q start: %w", t.Title, err)
				}
				dl, err := boardDay(t.Deadline)
				if err != nil {
					return res, fmt.Errorf("task %q deadline: %w", t.Title, err)
				}
				if _, err := c.CreateProjectTask(NewProjectTaskRequest{
					ProjectId:   pid,
					Title:       t.Title,
					Description: t.Description,
					StartDate:   st,
					Deadline:    dl,
					MilestoneId: int(msID),
				}); err != nil {
					return res, fmt.Errorf("create task %q: %w", t.Title, err)
				}
				haveTask[t.Title] = struct{}{}
			}
		}
	}
	return res, nil
}
