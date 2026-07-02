package lazyjwt

import (
	"errors"
	"testing"
	"time"
)

func TestSignAndVerifyClaims(t *testing.T) {
	key := []byte("secret")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	token, err := (Signer{KeyID: "main", Key: key}).Sign(Claims{
		Issuer:       "https://auth.example.com",
		Subject:      "alice",
		Audience:     []string{"https://app.example.com/mcp"},
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Hour),
		Scope:        []string{"openid", "profile"},
		ClientID:     "client",
		ClientDomain: "example.com",
		Extra:        map[string]any{"mcps": []string{"admin"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(token, ValidatorConfig{
		Keys:     map[string][]byte{"main": key},
		Issuer:   "https://auth.example.com",
		Audience: []string{"https://app.example.com/mcp"},
		ClientRules: []ClientRule{{
			ClientID: "client",
			Domain:   "example.com",
		}},
		Now: func() time.Time { return now.Add(time.Minute) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "alice" {
		t.Fatalf("Subject = %q, want alice", claims.Subject)
	}
	if got := claims.StringSlice("mcps"); len(got) != 1 || got[0] != "admin" {
		t.Fatalf("mcps = %#v, want admin", got)
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	key := []byte("secret")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	token, err := (Signer{KeyID: "main", Key: key}).Sign(Claims{
		ExpiresAt: now.Add(-time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Verify(token, ValidatorConfig{
		Keys: map[string][]byte{"main": key},
		Now:  func() time.Time { return now },
	})
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Verify error = %v, want ErrExpiredToken", err)
	}
}
