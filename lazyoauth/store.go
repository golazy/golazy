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

// RefreshToken is a legacy opaque refresh token record.
//
// Deprecated: new refresh tokens are signed, self-contained grants whose
// replay state is persisted through RefreshTokenStateStore. Store retains
// these methods so existing implementations and previously issued opaque
// tokens remain compatible during migration.
type RefreshToken struct {
	Token     string        `json:"token"`
	ClientID  string        `json:"client_id"`
	User      lazyauth.User `json:"user"`
	Scope     []string      `json:"scope,omitempty"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// RefreshTokenFamily is the minimal server-side state for one rotating refresh
// token family. The signed token carries the grant; the store tracks only the
// current token ID and whether the family has been revoked.
type RefreshTokenFamily struct {
	ID             string    `json:"id"`
	CurrentTokenID string    `json:"current_token_id"`
	ExpiresAt      time.Time `json:"expires_at"`
	Revoked        bool      `json:"revoked"`
}

// RefreshTokenRotation describes one atomic refresh-token rotation.
type RefreshTokenRotation struct {
	FamilyID         string
	PresentedTokenID string
	NextTokenID      string
	ExpiresAt        time.Time
}

// RefreshTokenStateStore tracks refresh-token rotation and family revocation.
//
// Rotate must atomically compare PresentedTokenID with the family's current
// token ID. A mismatch means replay: the implementation must revoke the family
// and return ErrRefreshTokenReplay. Missing, expired, or already-revoked
// families return ErrInvalidGrant. The signed refresh token carries user,
// client, scope, resource, and expiry data; implementations must not store the
// token itself. Implementations may opportunistically delete expired or
// revoked families; a missing family must continue to return ErrInvalidGrant.
type RefreshTokenStateStore interface {
	CreateFamily(context.Context, RefreshTokenFamily) error
	Rotate(context.Context, RefreshTokenRotation) error
	RevokeFamily(context.Context, string) error
}

// Store persists OAuth clients, authorization codes, and refresh tokens.
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
	Clients  map[string]Client             `json:"clients"`
	Codes    map[string]AuthCode           `json:"codes"`
	Refresh  map[string]RefreshToken       `json:"refresh"`
	Families map[string]RefreshTokenFamily `json:"refresh_token_families"`
	onChange func(*MemoryStore) error
}

var _ RefreshTokenStateStore = (*MemoryStore)(nil)

// NewMemoryStore creates an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		Clients:  map[string]Client{},
		Codes:    map[string]AuthCode{},
		Refresh:  map[string]RefreshToken{},
		Families: map[string]RefreshTokenFamily{},
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

// CreateFamily starts refresh-token rotation state for a newly issued grant.
func (store *MemoryStore) CreateFamily(_ context.Context, family RefreshTokenFamily) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	if family.ID == "" || family.CurrentTokenID == "" || family.ExpiresAt.IsZero() {
		return ErrInvalidGrant
	}
	if _, exists := store.Families[family.ID]; exists {
		return ErrInvalidGrant
	}
	store.Families[family.ID] = family
	return store.changed()
}

// Rotate atomically advances a refresh-token family or revokes it on replay.
func (store *MemoryStore) Rotate(_ context.Context, rotation RefreshTokenRotation) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	family, ok := store.Families[rotation.FamilyID]
	if !ok || family.Revoked || !time.Now().Before(family.ExpiresAt) {
		return ErrInvalidGrant
	}
	if family.CurrentTokenID != rotation.PresentedTokenID {
		family.Revoked = true
		store.Families[family.ID] = family
		if err := store.changed(); err != nil {
			return err
		}
		return ErrRefreshTokenReplay
	}
	if rotation.NextTokenID == "" || rotation.ExpiresAt.IsZero() {
		return ErrInvalidGrant
	}
	family.CurrentTokenID = rotation.NextTokenID
	family.ExpiresAt = rotation.ExpiresAt
	store.Families[family.ID] = family
	return store.changed()
}

// RevokeFamily invalidates a refresh-token family. It is idempotent.
func (store *MemoryStore) RevokeFamily(_ context.Context, familyID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensure()
	family, ok := store.Families[familyID]
	if !ok || family.Revoked {
		return nil
	}
	family.Revoked = true
	store.Families[familyID] = family
	return store.changed()
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
	if store.Families == nil {
		store.Families = map[string]RefreshTokenFamily{}
	}
}

func (store *MemoryStore) changed() error {
	if store.onChange == nil {
		return nil
	}
	return store.onChange(store)
}
