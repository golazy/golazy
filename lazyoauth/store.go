package lazyoauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"golazy.dev/lazyauth"
)

// Client is an OAuth client.
type Client struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
	Domain       string   `json:"domain,omitempty"`
}

// AuthCode is an authorization code record.
type AuthCode struct {
	Code                string        `json:"code"`
	ClientID            string        `json:"client_id"`
	RedirectURI         string        `json:"redirect_uri"`
	CodeChallenge       string        `json:"code_challenge,omitempty"`
	CodeChallengeMethod string        `json:"code_challenge_method,omitempty"`
	User                lazyauth.User `json:"user"`
	Scope               []string      `json:"scope,omitempty"`
	ExpiresAt           time.Time     `json:"expires_at"`
}

// RefreshToken is a refresh token record.
type RefreshToken struct {
	Token     string        `json:"token"`
	ClientID  string        `json:"client_id"`
	User      lazyauth.User `json:"user"`
	Scope     []string      `json:"scope,omitempty"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// Store persists OAuth clients and transient tokens.
type Store interface {
	SaveClient(context.Context, Client) error
	GetClient(context.Context, string) (Client, error)
	SaveAuthCode(context.Context, AuthCode) error
	TakeAuthCode(context.Context, string) (AuthCode, error)
	SaveRefreshToken(context.Context, RefreshToken) error
	GetRefreshToken(context.Context, string) (RefreshToken, error)
}

// MemoryStore is an in-memory OAuth store.
type MemoryStore struct {
	mu       sync.Mutex
	Clients  map[string]Client       `json:"clients"`
	Codes    map[string]AuthCode     `json:"codes"`
	Refresh  map[string]RefreshToken `json:"refresh"`
	onChange func(*MemoryStore) error
}

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		Clients: map[string]Client{},
		Codes:   map[string]AuthCode{},
		Refresh: map[string]RefreshToken{},
	}
}

// NewDiskStore loads or creates a JSON OAuth store at path.
func NewDiskStore(path string) (*MemoryStore, error) {
	store := NewMemoryStore()
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, store); err != nil {
			return nil, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	store.ensure()
	store.onChange = func(store *MemoryStore) error {
		data, err := json.MarshalIndent(store, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0o600)
	}
	return store, nil
}

func (store *MemoryStore) SaveClient(_ context.Context, client Client) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	if client.ID == "" {
		return fmt.Errorf("lazyoauth: client id is required")
	}
	store.Clients[client.ID] = client
	return store.changed()
}

func (store *MemoryStore) GetClient(_ context.Context, id string) (Client, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	client, ok := store.Clients[id]
	if !ok {
		return Client{}, ErrInvalidClient
	}
	return client, nil
}

func (store *MemoryStore) SaveAuthCode(_ context.Context, code AuthCode) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	store.Codes[code.Code] = code
	return store.changed()
}

func (store *MemoryStore) TakeAuthCode(_ context.Context, code string) (AuthCode, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	record, ok := store.Codes[code]
	if !ok {
		return AuthCode{}, ErrInvalidGrant
	}
	delete(store.Codes, code)
	return record, store.changed()
}

func (store *MemoryStore) SaveRefreshToken(_ context.Context, token RefreshToken) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	store.Refresh[token.Token] = token
	return store.changed()
}

func (store *MemoryStore) GetRefreshToken(_ context.Context, token string) (RefreshToken, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	record, ok := store.Refresh[token]
	if !ok {
		return RefreshToken{}, ErrInvalidGrant
	}
	return record, nil
}

func (store *MemoryStore) ensure() {
	if store.Clients == nil {
		store.Clients = map[string]Client{}
	}
	if store.Codes == nil {
		store.Codes = map[string]AuthCode{}
	}
	if store.Refresh == nil {
		store.Refresh = map[string]RefreshToken{}
	}
}

func (store *MemoryStore) changed() error {
	if store.onChange == nil {
		return nil
	}
	return store.onChange(store)
}
