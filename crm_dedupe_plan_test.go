package onlyoffice

import (
	"reflect"
	"testing"
)

func TestPickCanonicalID(t *testing.T) {
	if got := PickCanonicalID([]int64{908, 857, 900}); got != 857 {
		t.Fatalf("got %d", got)
	}
	if got := PickCanonicalID(nil); got != 0 {
		t.Fatalf("got %d", got)
	}
}

func TestGroupCompaniesByName(t *testing.T) {
	items := []map[string]any{
		{"id": float64(857), "displayName": "711media", "isCompany": true},
		{"id": float64(908), "displayName": "711media", "isCompany": true},
		{"id": float64(1), "displayName": "Acme", "isCompany": true},
	}
	groups := GroupCompaniesByName(items)
	if len(groups) != 2 {
		t.Fatalf("groups: %d", len(groups))
	}
	key := CompanyGroupingKey("711media")
	if len(groups[key]) != 2 {
		t.Fatalf("711media group: %d", len(groups[key]))
	}
}

func TestDedupeMembersByDisplayName(t *testing.T) {
	members := []map[string]any{
		{"id": float64(857), "displayName": "711media"},
		{"id": float64(908), "displayName": "711media"},
		{"id": float64(10), "displayName": "Acme"},
	}
	remove := DedupeMembersByDisplayName(members)
	want := []int64{908}
	if !reflect.DeepEqual(remove, want) {
		t.Fatalf("remove %v want %v", remove, want)
	}
}

func TestDedupeMemberIDs(t *testing.T) {
	remove := DedupeMemberIDs([]int64{1, 2, 2, 3, 1})
	if !reflect.DeepEqual(remove, []int64{2, 1}) {
		t.Fatalf("got %v", remove)
	}
}

func TestGroupContactInfoRows(t *testing.T) {
	rows := []map[string]any{
		{"id": float64(1), "infoType": "Email", "data": "a@b.com", "isPrimary": false},
		{"id": float64(2), "infoType": "Email", "data": "a@b.com", "isPrimary": true},
	}
	remove := GroupContactInfoRows(rows)
	if len(remove) != 1 || remove[0] != 1 {
		t.Fatalf("remove %v", remove)
	}
}

func TestGroupCompaniesBySlogan(t *testing.T) {
	items := []map[string]any{
		{"id": float64(1), "displayName": "Affirm", "isCompany": true},
		{"id": float64(2), "displayName": "Affirm — Fraud Engineering", "isCompany": true},
	}
	groups := GroupCompaniesByName(items)
	if len(groups) != 1 {
		t.Fatalf("groups: %d", len(groups))
	}
	key := CompanyGroupingKey("Affirm")
	if len(groups[key]) != 2 {
		t.Fatalf("affirm group: %d", len(groups[key]))
	}
}

func TestDedupeMembersBySloganDisplayName(t *testing.T) {
	members := []map[string]any{
		{"id": float64(1), "displayName": "Affirm"},
		{"id": float64(2), "displayName": "Affirm — Fraud Engineering"},
	}
	remove := DedupeMembersByDisplayName(members)
	if !reflect.DeepEqual(remove, []int64{2}) {
		t.Fatalf("remove %v", remove)
	}
}

func TestDealTitleKey(t *testing.T) {
	if got := DealTitleKey("Dev @ Acme", false); got != collapseKey("Dev @ Acme") {
		t.Fatalf("got %q", got)
	}
	if got := DealTitleKey("Dev @ Acme", true); got != NormalizeOpportunityTitle("Dev") {
		t.Fatalf("got %q", got)
	}
	if got := DealTitleKey("Dev @ Affirm — Fraud Engineering", false); got != DealTitleKey("Dev @ Affirm", false) {
		t.Fatalf("slogan keys differ: %q vs %q", got, DealTitleKey("Dev @ Affirm", false))
	}
}
