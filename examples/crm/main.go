// Package main lists CRM contacts and opportunities, and shows how to create
// and remove a demo company + opportunity + history note in OnlyOffice CRM.
//
//	export ONLYOFFICE_URL="https://your-instance.onlyoffice.com"
//	export ONLYOFFICE_USER="admin@example.com"
//	export ONLYOFFICE_PASS="your-password"
//	go run ./examples/crm
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	onlyoffice "github.com/eslider/go-onlyoffice"
)

func main() {
	creds := onlyoffice.GetEnvironmentCredentials()
	if creds.Url == "" {
		fmt.Fprintln(os.Stderr, "ONLYOFFICE_URL is not set")
		os.Exit(1)
	}

	client := onlyoffice.NewClient(creds)
	ctx := context.Background()

	contacts, total, err := client.ListContacts(ctx, 10, 0, "")
	if err != nil {
		log.Fatalf("list contacts: %v", err)
	}
	fmt.Printf("CRM contacts: %d (total=%d)\n", len(contacts), total)
	for _, c := range contacts {
		fmt.Printf("  - id=%v name=%v\n", c["id"], c["displayName"])
	}

	opps, oppTotal, err := client.ListOpportunities(ctx, 10, 0)
	if err != nil {
		log.Fatalf("list opportunities: %v", err)
	}
	fmt.Printf("\nOpportunities: %d (total=%d)\n", len(opps), oppTotal)
	for _, o := range opps {
		fmt.Printf("  - id=%v title=%v\n", o["id"], o["title"])
	}

	// Full create/attach/delete cycle — uncomment to exercise.
	// company, err := client.CreateCompany(ctx, "Demo GmbH")
	// if err != nil { log.Fatalf("create company: %v", err) }
	// cid := fmt.Sprint(company["id"])
	// defer client.DeleteContact(ctx, cid)
	//
	// stages, err := client.ListDealStages(ctx)
	// if err != nil || len(stages) == 0 { log.Fatalf("no deal stages: %v", err) }
	// stageID := int(stages[0]["id"].(float64))
	//
	// opp, err := client.CreateOpportunity(ctx, "Demo opp", stageID, "", "EUR", "from examples/crm", 0)
	// if err != nil { log.Fatalf("create opportunity: %v", err) }
	// oid := fmt.Sprint(opp["id"])
	// _, _ = client.AddOpportunityMember(ctx, oid, cid)
	// _, _ = client.AddHistoryNote(ctx, "opportunity", int(opp["id"].(float64)), "Touched via library example", 0)
	// _, _ = client.DeleteOpportunity(ctx, oid)
}
