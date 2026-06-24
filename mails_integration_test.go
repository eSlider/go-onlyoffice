//go:build integration

package onlyoffice

import (
	"context"
	"testing"
)

func TestIntegrationMailList(t *testing.T) {
	creds := GetEnvironmentCredentials()
	if creds.Url == "" || creds.User == "" || creds.Password == "" {
		t.Skip("ONLYOFFICE_URL/USER/PASS not set")
	}
	c := NewClient(creds)
	ctx := context.Background()
	if err := c.AuthenticateContext(ctx); err != nil {
		t.Fatalf("auth: %v", err)
	}
	accounts, err := c.ListMailAccounts(ctx)
	if err != nil {
		t.Fatalf("ListMailAccounts: %v", err)
	}
	t.Logf("accounts=%d", len(accounts))

	msgs, err := c.ListMailMessages(ctx, MailMessagesFilter{Folder: MailFolderInbox, Count: 3})
	if err != nil {
		t.Fatalf("ListMailMessages: %v", err)
	}
	if len(msgs) == 0 {
		t.Skip("no inbox messages to inspect")
	}
	id := idFromMap(msgs[0], "id")
	msg, err := c.GetMailMessage(ctx, id)
	if err != nil {
		t.Fatalf("GetMailMessage(%s): %v", id, err)
	}
	if msg["subject"] == nil {
		t.Fatalf("message missing subject: %+v", msg)
	}
}
