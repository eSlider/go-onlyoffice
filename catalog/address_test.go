package catalog

import "testing"

func TestMergeAddressesDedup(t *testing.T) {
	dst := []Address{{Street: "Weg 1", City: "Stadt", Zip: "1"}}
	got := mergeAddresses(dst, []Address{
		{Street: "weg 1", City: "stadt", Zip: "1"}, // duplicate (case-insensitive)
		{Street: "Weg 2", City: "Stadt", Zip: "2"}, // new
	})
	if len(got) != 2 {
		t.Fatalf("got %d addresses, want 2: %+v", len(got), got)
	}
	if got[1].Street != "Weg 2" {
		t.Errorf("second = %+v", got[1])
	}
}

func TestEntryAddressesYAML(t *testing.T) {
	doc := &Document{Entries: []Entry{{
		ID: "person:max maier", Kind: "person", Name: "Max Maier",
		Addresses: []Address{{Street: "Weg 1", City: "Stadt", Zip: "12345", Category: "Billing", Primary: true}},
	}}}
	merged := MergeDocs(doc)
	if len(merged.Entries) != 1 || len(merged.Entries[0].Addresses) != 1 {
		t.Fatalf("merged = %+v", merged.Entries)
	}
}
