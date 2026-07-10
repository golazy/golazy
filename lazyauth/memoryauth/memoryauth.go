package memoryauth

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
	"sync"

	"golazy.dev/lazyauth"
)

const defaultUser = "admin"

// User is an in-memory password user.
type User struct {
	ID       string
	Password string
	Data     map[string]any
}

type record struct {
	password string
	user     lazyauth.User
}

// Provider authenticates password credentials against an in-memory user set.
type Provider struct {
	mu    sync.RWMutex
	users map[string]record
}

// New creates an in-memory authenticator with the provided users.
func New(users ...User) *Provider {
	provider := &Provider{users: map[string]record{}}
	for _, user := range users {
		provider.Set(user)
	}
	return provider
}

// FromEnvironment creates the default lazyapp memory backend.
//
// With no LAZYAUTH_DEFAULT_PASS environment variable, the backend has zero
// users. When LAZYAUTH_DEFAULT_PASS is set, it creates one password user named
// admin, or LAZYAUTH_DEFAULT_USER when that value is non-empty.
func FromEnvironment() *Provider {
	password, ok := os.LookupEnv("LAZYAUTH_DEFAULT_PASS")
	if !ok {
		return New()
	}
	username := strings.TrimSpace(os.Getenv("LAZYAUTH_DEFAULT_USER"))
	if username == "" {
		username = defaultUser
	}
	return New(User{
		ID:       username,
		Password: password,
		Data: map[string]any{
			"admin":    true,
			"username": username,
		},
	})
}

// Set adds or replaces one in-memory user.
func (p *Provider) Set(user User) {
	if p == nil || strings.TrimSpace(user.ID) == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.users == nil {
		p.users = map[string]record{}
	}
	data := copyData(user.Data)
	p.users[user.ID] = record{
		password: user.Password,
		user: lazyauth.User{
			ID:   user.ID,
			Data: data,
		},
	}
}

// Authenticate implements lazyauth.Authenticator.
func (p *Provider) Authenticate(_ context.Context, credential lazyauth.Credential) (lazyauth.User, error) {
	if p == nil {
		return lazyauth.User{}, fmt.Errorf("memoryauth: provider is nil")
	}
	if credential.Kind != "" && credential.Kind != "password" {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	p.mu.RLock()
	rec, ok := p.users[credential.Identifier]
	p.mu.RUnlock()
	if !ok || rec.password != credential.Secret {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	return lazyauth.User{ID: rec.user.ID, Data: copyData(rec.user.Data)}, nil
}

func copyData(data map[string]any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	copied := make(map[string]any, len(data))
	maps.Copy(copied, data)
	return copied
}
