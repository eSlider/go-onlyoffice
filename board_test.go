package onlyoffice

import "testing"

func TestParseBoard(t *testing.T) {
	b, err := ParseBoard([]byte(`
projects:
  - id: 13
    name: Example
    milestones:
      - title: "[lq] Test"
        deadline: "2026-01-15"
        key: true
        tasks:
          - title: Draft
            start: "2026-01-02"
            deadline: "2026-01-10"
            description: "…"
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Projects) != 1 || b.Projects[0].ID != 13 {
		t.Fatalf("projects: %+v", b.Projects)
	}
	ms := b.Projects[0].Milestones[0]
	if ms.Title != "[lq] Test" || !ms.Key || ms.Deadline != "2026-01-15" {
		t.Fatalf("milestone: %+v", ms)
	}
	if len(ms.Tasks) != 1 || ms.Tasks[0].Title != "Draft" || ms.Tasks[0].Start != "2026-01-02" {
		t.Fatalf("task: %+v", ms.Tasks)
	}
}

func TestParseBoardEmpty(t *testing.T) {
	if _, err := ParseBoard([]byte("projects: []")); err == nil {
		t.Fatal("expected error for empty board")
	}
}

func TestBoardMilestoneIDs(t *testing.T) {
	title := "[lq] Test"
	id := int64(9)
	got := boardMilestoneIDs([]*Milestone{{Title: &title, ID: &id}, nil})
	if got[title] != 9 {
		t.Fatalf("%v", got)
	}
}

func TestBoardTaskTitles(t *testing.T) {
	got := boardTaskTitles([]map[string]any{{"title": "a"}, {"title": "b"}, {"nope": 1}})
	if _, ok := got["a"]; !ok {
		t.Fatal("missing a")
	}
	if _, ok := got["b"]; !ok {
		t.Fatal("missing b")
	}
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
}

func TestBoardDay(t *testing.T) {
	if _, err := boardDay("2026-01-15"); err != nil {
		t.Fatal(err)
	}
	if _, err := boardDay("15.01.2026"); err == nil {
		t.Fatal("expected error for non-ISO date")
	}
}
