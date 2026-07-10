package memoryauth

import (
	"context"
	"errors"
	"os"
	"testing"

	"golazy.dev/lazyauth"
)

func TestProviderStartsEmpty(t *testing.T) {
	provider := New()
	_, err := provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "admin",
		Secret:     "secret",
	})
	if !errors.Is(err, lazyauth.ErrInvalidCredentials) {
		t.Fatalf("Authenticate error = %v, want ErrInvalidCredentials", err)
	}
}

func TestProviderAuthenticatesConfiguredUser(t *testing.T) {
	provider := New(User{
		ID:       "alice",
		Password: "secret",
		Data:     map[string]any{"email": "alice@example.com"},
	})

	user, err := provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "alice",
		Secret:     "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "alice" || user.Data["email"] != "alice@example.com" {
		t.Fatalf("user = %#v, want alice with email", user)
	}
	user.Data["email"] = "changed"

	user, err = provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "alice",
		Secret:     "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Data["email"] != "alice@example.com" {
		t.Fatalf("user data was not copied: %#v", user.Data)
	}
}

func TestFromEnvironmentDefaultsToZeroUsers(t *testing.T) {
	oldPass, hadPass := os.LookupEnv("LAZYAUTH_DEFAULT_PASS")
	oldUser, hadUser := os.LookupEnv("LAZYAUTH_DEFAULT_USER")
	t.Cleanup(func() {
		restoreEnv("LAZYAUTH_DEFAULT_PASS", oldPass, hadPass)
		restoreEnv("LAZYAUTH_DEFAULT_USER", oldUser, hadUser)
	})
	_ = os.Unsetenv("LAZYAUTH_DEFAULT_PASS")
	_ = os.Unsetenv("LAZYAUTH_DEFAULT_USER")

	provider := FromEnvironment()
	_, err := provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "admin",
		Secret:     "secret",
	})
	if !errors.Is(err, lazyauth.ErrInvalidCredentials) {
		t.Fatalf("Authenticate error = %v, want ErrInvalidCredentials", err)
	}
}

func TestFromEnvironmentCreatesAdminUser(t *testing.T) {
	t.Setenv("LAZYAUTH_DEFAULT_PASS", "secret")
	t.Setenv("LAZYAUTH_DEFAULT_USER", "")
	provider := FromEnvironment()

	user, err := provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "admin",
		Secret:     "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "admin" || user.Data["admin"] != true || user.Data["username"] != "admin" {
		t.Fatalf("user = %#v, want admin bootstrap data", user)
	}
}

func TestFromEnvironmentUsesCustomUser(t *testing.T) {
	t.Setenv("LAZYAUTH_DEFAULT_PASS", "secret")
	t.Setenv("LAZYAUTH_DEFAULT_USER", "ops")
	provider := FromEnvironment()

	user, err := provider.Authenticate(context.Background(), lazyauth.Credential{
		Kind:       "password",
		Identifier: "ops",
		Secret:     "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "ops" || user.Data["username"] != "ops" {
		t.Fatalf("user = %#v, want ops", user)
	}
}

func restoreEnv(name string, value string, ok bool) {
	if !ok {
		_ = os.Unsetenv(name)
		return
	}
	_ = os.Setenv(name, value)
}
