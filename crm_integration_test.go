//go:build integration

package onlyoffice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testCRMPrefix = "go-onlyoffice-test-"

func TestIntegrationMergeContacts(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	name := testCRMPrefix + "merge-" + suffix

	a, err := c.CreateCompany(ctx, name)
	if err != nil {
		t.Fatalf("CreateCompany a: %v", err)
	}
	b, err := c.CreateCompany(ctx, name)
	if err != nil {
		t.Fatalf("CreateCompany b: %v", err)
	}
	aID := strconv.FormatInt(flexInt(a["id"]), 10)
	bID := strconv.FormatInt(flexInt(b["id"]), 10)
	t.Cleanup(func() { _, _ = c.DeleteContact(ctx, aID) })

	if _, err := c.MergeContacts(ctx, aID, bID); err != nil {
		t.Fatalf("MergeContacts: %v", err)
	}
	got, err := c.GetContact(ctx, aID)
	if err != nil {
		t.Fatalf("GetContact: %v", err)
	}
	if got == nil {
		t.Fatal("primary contact missing after merge")
	}
}

func TestIntegrationUpdateOpportunityTitle(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	stages, err := c.ListDealStages(ctx)
	if err != nil || len(stages) == 0 {
		t.Fatalf("ListDealStages: %v", err)
	}
	stageID := int(flexInt(stages[0]["id"]))
	title := testCRMPrefix + "title-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	opp, err := c.CreateOpportunity(ctx, title, stageID, "", "EUR", "", 0)
	if err != nil {
		t.Fatalf("CreateOpportunity: %v", err)
	}
	id := strconv.FormatInt(flexInt(opp["id"]), 10)
	t.Cleanup(func() { _, _ = c.DeleteOpportunity(ctx, id) })

	newTitle := title + "-renamed"
	if _, err := c.UpdateOpportunityTitle(ctx, id, newTitle); err != nil {
		t.Fatalf("UpdateOpportunityTitle: %v", err)
	}
	got, err := c.GetOpportunity(ctx, id)
	if err != nil {
		t.Fatalf("GetOpportunity: %v", err)
	}
	if strings.TrimSpace(fmt.Sprint(got["title"])) != newTitle {
		t.Fatalf("title %q want %q", got["title"], newTitle)
	}
}

func TestIntegrationDedupeCompaniesSmoke(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	name := testCRMPrefix + "dedupe-" + suffix
	co1, err := c.CreateCompany(ctx, name)
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	co2, err := c.CreateCompany(ctx, name)
	if err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}
	t.Cleanup(func() {
		id1 := strconv.FormatInt(flexInt(co1["id"]), 10)
		_, _ = c.DeleteContact(ctx, id1)
	})

	res, err := DedupeCompanies(ctx, c)
	if err != nil {
		t.Fatalf("DedupeCompanies: %v", err)
	}
	if res.Merged < 1 {
		t.Fatalf("expected merge, got %+v", res)
	}
	_ = co2
}

func TestIntegrationFixOpportunityTitleDeal231Pattern(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	stages, err := c.ListDealStages(ctx)
	if err != nil || len(stages) == 0 {
		t.Fatalf("ListDealStages: %v", err)
	}
	stageID := int(flexInt(stages[0]["id"]))
	opp, err := c.CreateOpportunity(ctx, " @ 711media-test", stageID, "", "EUR", "", 0)
	if err != nil {
		t.Fatalf("CreateOpportunity: %v", err)
	}
	id := strconv.FormatInt(flexInt(opp["id"]), 10)
	t.Cleanup(func() { _, _ = c.DeleteOpportunity(ctx, id) })

	fixed := FixDealTitle(" @ 711media-test")
	if _, err := c.UpdateOpportunityTitle(ctx, id, fixed); err != nil {
		t.Fatalf("UpdateOpportunityTitle: %v", err)
	}
	got, err := c.GetOpportunity(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got["title"]) != "711media-test" {
		t.Fatalf("title %q", got["title"])
	}
}
