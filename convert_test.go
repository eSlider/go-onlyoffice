package onlyoffice

import (
	"strings"
	"testing"
)

func TestSignJWT(t *testing.T) {
	payload := map[string]any{"url": "u", "outputtype": "pdf"}
	tok, err := SignJWT("secret", payload)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Split(tok, ".")); n != 3 {
		t.Fatalf("JWT must have 3 parts, got %d", n)
	}
	tok2, _ := SignJWT("secret", payload)
	if tok != tok2 {
		t.Fatal("SignJWT must be deterministic for identical input")
	}
	if _, err := SignJWT("", payload); err == nil {
		t.Fatal("expected error for empty secret")
	}
}
