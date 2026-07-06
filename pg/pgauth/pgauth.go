package pgauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyauth"
	"golazy.dev/lazyauth/fileauth"
)

// Provider authenticates lazyauth password credentials from PostgreSQL.
type Provider struct {
	pool *pgxpool.Pool
}

var _ lazyauth.Authenticator = (*Provider)(nil)

// UserParams describes a user to create or update in lazy_auth_users.
type UserParams struct {
	ID           string
	Email        string
	Password     string
	PasswordHash string
	Data         map[string]any
}

// New creates a PostgreSQL-backed lazyauth provider.
func New(pool *pgxpool.Pool) *Provider {
	return &Provider{pool: pool}
}

// Authenticate implements lazyauth.Authenticator.
func (p *Provider) Authenticate(ctx context.Context, credential lazyauth.Credential) (lazyauth.User, error) {
	if err := p.validate(); err != nil {
		return lazyauth.User{}, err
	}
	if credential.Kind != "" && credential.Kind != "password" {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	identifier := strings.TrimSpace(credential.Identifier)
	if identifier == "" || credential.Secret == "" {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}

	var id string
	var email string
	var passwordHash string
	var rawData []byte
	err := p.pool.QueryRow(ctx, `
SELECT id, email, password_hash, data
FROM lazy_auth_users
WHERE id = $1 OR lower(email) = lower($1)
LIMIT 1`,
		identifier,
	).Scan(&id, &email, &passwordHash, &rawData)
	if errors.Is(err, pgx.ErrNoRows) {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	if err != nil {
		return lazyauth.User{}, fmt.Errorf("pgauth: authenticate %q: %w", identifier, err)
	}
	if !fileauth.VerifyPassword(passwordHash, credential.Secret) {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	data, err := decodeData(rawData)
	if err != nil {
		return lazyauth.User{}, fmt.Errorf("pgauth: decode user %q data: %w", id, err)
	}
	if _, exists := data["email"]; !exists && email != "" {
		data["email"] = email
	}
	return lazyauth.User{ID: id, Data: data}, nil
}

// UpsertUser creates or updates one password-authenticated user.
func (p *Provider) UpsertUser(ctx context.Context, params UserParams) (lazyauth.User, error) {
	if err := p.validate(); err != nil {
		return lazyauth.User{}, err
	}
	email := normalizeEmail(params.Email)
	if email == "" {
		return lazyauth.User{}, fmt.Errorf("pgauth: email is required")
	}
	id := strings.TrimSpace(params.ID)
	if id == "" {
		id = email
	}
	passwordHash := strings.TrimSpace(params.PasswordHash)
	if passwordHash == "" {
		if params.Password == "" {
			return lazyauth.User{}, fmt.Errorf("pgauth: password or password hash is required")
		}
		var err error
		passwordHash, err = HashPassword(params.Password)
		if err != nil {
			return lazyauth.User{}, fmt.Errorf("pgauth: hash password: %w", err)
		}
	}

	data := copyData(params.Data)
	if _, exists := data["email"]; !exists {
		data["email"] = email
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return lazyauth.User{}, fmt.Errorf("pgauth: encode user data: %w", err)
	}

	var savedID string
	var savedEmail string
	var rawData []byte
	if err := p.pool.QueryRow(ctx, `
INSERT INTO lazy_auth_users (
	id,
	email,
	password_hash,
	data
) VALUES (
	$1, $2, $3, $4::jsonb
)
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
	password_hash = EXCLUDED.password_hash,
	data = EXCLUDED.data,
	updated_at = now()
RETURNING id, email, data`,
		id,
		email,
		passwordHash,
		encoded,
	).Scan(&savedID, &savedEmail, &rawData); err != nil {
		return lazyauth.User{}, fmt.Errorf("pgauth: upsert user %q: %w", id, err)
	}
	savedData, err := decodeData(rawData)
	if err != nil {
		return lazyauth.User{}, fmt.Errorf("pgauth: decode user %q data: %w", savedID, err)
	}
	if _, exists := savedData["email"]; !exists && savedEmail != "" {
		savedData["email"] = savedEmail
	}
	return lazyauth.User{ID: savedID, Data: savedData}, nil
}

// HashPassword hashes password with the same PBKDF2 format as fileauth.
func HashPassword(password string) (string, error) {
	return fileauth.HashPassword(password)
}

// VerifyPassword verifies a password hash produced by HashPassword.
func VerifyPassword(encoded string, password string) bool {
	return fileauth.VerifyPassword(encoded, password)
}

func (p *Provider) validate() error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("pgauth: pool is required")
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func decodeData(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	data := map[string]any{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func copyData(data map[string]any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	copied := make(map[string]any, len(data))
	maps.Copy(copied, data)
	return copied
}
