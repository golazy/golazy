package lazyauth

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidCredentials = errors.New("lazyauth: invalid credentials")
	ErrMissingUser        = errors.New("lazyauth: user is missing")
)

type contextKey struct{}

// User is the authenticated identity returned by an Authenticator.
type User struct {
	ID   string         `json:"id"`
	Data map[string]any `json:"data,omitempty"`
}

// Credential describes one authentication attempt.
type Credential struct {
	Kind       string
	Identifier string
	Secret     string
	Values     map[string]string
}

// Authenticator validates credentials and returns a user identity.
type Authenticator interface {
	Authenticate(context.Context, Credential) (User, error)
}

// Config configures authentication for higher-level packages.
type Config struct {
	Authenticator Authenticator
}

// Authenticate validates credential through config.Authenticator.
func Authenticate(ctx context.Context, config Config, credential Credential) (User, error) {
	if config.Authenticator == nil {
		return User{}, fmt.Errorf("lazyauth: authenticator is required")
	}
	user, err := config.Authenticator.Authenticate(ctx, credential)
	if err != nil {
		return User{}, err
	}
	if user.ID == "" {
		return User{}, ErrMissingUser
	}
	if user.Data == nil {
		user.Data = map[string]any{}
	}
	return user, nil
}

// WithUser stores user in ctx.
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

// FromContext returns the authenticated user stored in ctx.
func FromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(contextKey{}).(User)
	return user, ok && user.ID != ""
}
