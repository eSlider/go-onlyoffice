package onlyoffice

import (
	"context"
	"reflect"
	"testing"
)

type fakeDedupeClient struct {
	contacts      []map[string]any
	opportunities []map[string]any
	contactByID   map[string]map[string]any
	oppByID       map[string]map[string]any
	merged        [][2]string
	deletedOpp    []string
	deletedInfo   [][2]string
	removedMember [][2]string
	renamed       map[string]string
}

func (f *fakeDedupeClient) ListAllContacts(ctx context.Context) ([]map[string]any, error) {
	return f.contacts, nil
}

func (f *fakeDedupeClient) ListAllOpportunities(ctx context.Context) ([]map[string]any, error) {
	return f.opportunities, nil
}

func (f *fakeDedupeClient) GetContact(ctx context.Context, contactID string) (map[string]any, error) {
	return f.contactByID[contactID], nil
}

func (f *fakeDedupeClient) GetOpportunity(ctx context.Context, id string) (map[string]any, error) {
	return f.oppByID[id], nil
}

func (f *fakeDedupeClient) MergeContacts(ctx context.Context, primaryID, secondaryID string) (map[string]any, error) {
	f.merged = append(f.merged, [2]string{primaryID, secondaryID})
	return map[string]any{"id": primaryID}, nil
}

func (f *fakeDedupeClient) DeleteOpportunity(ctx context.Context, id string) (map[string]any, error) {
	f.deletedOpp = append(f.deletedOpp, id)
	return map[string]any{}, nil
}

func (f *fakeDedupeClient) DeleteContactInfo(ctx context.Context, contactID, dataID string) (map[string]any, error) {
	f.deletedInfo = append(f.deletedInfo, [2]string{contactID, dataID})
	return map[string]any{}, nil
}

func (f *fakeDedupeClient) ListCompanyPersons(ctx context.Context, companyID string) ([]map[string]any, error) {
	return nil, nil
}

func (f *fakeDedupeClient) AddOpportunityMember(ctx context.Context, oppID, contactID string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (f *fakeDedupeClient) RemoveOpportunityMember(ctx context.Context, oppID, contactID string) (map[string]any, error) {
	f.removedMember = append(f.removedMember, [2]string{oppID, contactID})
	return map[string]any{}, nil
}

func (f *fakeDedupeClient) UpdateOpportunityTitle(ctx context.Context, id, newTitle string) (map[string]any, error) {
	if f.renamed == nil {
		f.renamed = make(map[string]string)
	}
	f.renamed[id] = newTitle
	return map[string]any{"title": newTitle}, nil
}

func TestDedupeCompaniesOrchestration(t *testing.T) {
	f := &fakeDedupeClient{
		contacts: []map[string]any{
			{"id": float64(857), "displayName": "711media", "isCompany": true},
			{"id": float64(908), "displayName": "711media", "isCompany": true},
		},
	}
	res, err := DedupeCompanies(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if res.Merged != 1 || res.Groups != 1 {
		t.Fatalf("res %+v", res)
	}
	want := [][2]string{{"857", "908"}}
	if !reflect.DeepEqual(f.merged, want) {
		t.Fatalf("merged %v", f.merged)
	}
}

func TestDedupeOpportunityMembersOrchestration(t *testing.T) {
	f := &fakeDedupeClient{
		opportunities: []map[string]any{{"id": float64(231), "title": " @ 711media"}},
		oppByID: map[string]map[string]any{
			"231": {
				"id": float64(231),
				"members": []any{
					map[string]any{"id": float64(857), "displayName": "711media"},
					map[string]any{"id": float64(908), "displayName": "711media"},
				},
			},
		},
	}
	res, err := DedupeOpportunityMembers(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed != 1 {
		t.Fatalf("res %+v", res)
	}
	if len(f.removedMember) != 1 || f.removedMember[0][1] != "908" {
		t.Fatalf("removed %v", f.removedMember)
	}
}

func TestFixOpportunityTitlesOrchestration(t *testing.T) {
	f := &fakeDedupeClient{
		opportunities: []map[string]any{{"id": float64(231), "title": " @ 711media"}},
		oppByID:       map[string]map[string]any{"231": {"id": float64(231), "title": " @ 711media"}},
	}
	res, err := FixOpportunityTitles(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if res.Renamed != 1 {
		t.Fatalf("res %+v", res)
	}
	if f.renamed["231"] != "711media" {
		t.Fatalf("renamed %v", f.renamed)
	}
}
