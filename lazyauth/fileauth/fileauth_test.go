package fileauth

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"golazy.dev/lazyauth"
)

func TestProviderAuthenticatesJSONLUser(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "users.jsonl")
	if err := os.WriteFile(path, []byte(`{"id":"alice","password_hash":"`+hash+`","data":{"email":"alice@example.com","mcps":["admin"]}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	user, err := provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "alice",
		Secret:     "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "alice" {
		t.Fatalf("ID = %q, want alice", user.ID)
	}
	if user.Data["email"] != "alice@example.com" {
		t.Fatalf("email = %#v", user.Data["email"])
	}
}

func TestProviderRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "users.jsonl")
	if err := os.WriteFile(path, []byte(`{"id":"alice","password_hash":"`+hash+`"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	provider, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "alice",
		Secret:     "wrong",
	})
	if err == nil {
		t.Fatal("Authenticate succeeded with wrong password")
	}
}
