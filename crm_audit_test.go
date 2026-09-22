package onlyoffice

import "testing"

func TestAuditID(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{float64(12), 12},
		{7, 7},
		{int64(9), 9},
		{"42", 42},
		{nil, 0},
	}
	for _, c := range cases {
		if got := auditID(c.in); got != c.want {
			t.Fatalf("auditID(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestTaskCountsByOpportunity(t *testing.T) {
	open, closed := taskCountsByOpportunity([]map[string]any{
		{"id": 1, "status": 1, "entity": map[string]any{"entityType": "opportunity", "entityId": 10}},
		{"id": 2, "status": "2", "entity": map[string]any{"entityType": "opportunity", "entityId": 10}},
		{"id": 3, "status": 1, "entity": map[string]any{"entityType": "contact", "entityId": 10}},
		{"id": 4, "entity": "not-a-map"},
	})
	if open["10"] != 1 || closed["10"] != 1 {
		t.Fatalf("open=%v closed=%v", open, closed)
	}
	if len(open) != 1 || len(closed) != 1 {
		t.Fatalf("unexpected buckets: open=%v closed=%v", open, closed)
	}
}
