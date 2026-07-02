package lazyjwt

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMalformedToken   = errors.New("lazyjwt: malformed token")
	ErrUnsupportedAlg   = errors.New("lazyjwt: unsupported signing algorithm")
	ErrInvalidSignature = errors.New("lazyjwt: invalid signature")
	ErrExpiredToken     = errors.New("lazyjwt: token is expired")
	ErrTokenNotYetValid = errors.New("lazyjwt: token is not yet valid")
	ErrInvalidIssuer    = errors.New("lazyjwt: invalid issuer")
	ErrInvalidAudience  = errors.New("lazyjwt: invalid audience")
	ErrInvalidClient    = errors.New("lazyjwt: invalid client")
)

type contextKey struct{}

// Claims contains the registered JWT claims GoLazy packages need plus an Extra
// map for application or protocol-specific values such as "mcps".
type Claims struct {
	Issuer       string
	Subject      string
	Audience     []string
	ExpiresAt    time.Time
	NotBefore    time.Time
	IssuedAt     time.Time
	ID           string
	Scope        []string
	ClientID     string
	ClientDomain string
	Extra        map[string]any
}

// StringSlice returns an extra claim as a string slice.
func (claims Claims) StringSlice(name string) []string {
	if claims.Extra == nil {
		return nil
	}
	return stringSlice(claims.Extra[name])
}

// HasScope reports whether claims include scope.
func (claims Claims) HasScope(scope string) bool {
	for _, candidate := range claims.Scope {
		if candidate == scope {
			return true
		}
	}
	return false
}

// WithClaims stores validated claims in ctx.
func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, contextKey{}, claims)
}

// ClaimsFromContext returns validated claims stored in ctx.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(contextKey{}).(Claims)
	return claims, ok
}

// Signer signs JWTs with a symmetric HS256 key.
type Signer struct {
	KeyID string
	Key   []byte
}

// Sign signs claims as an HS256 JWT.
func (signer Signer) Sign(claims Claims) (string, error) {
	if len(signer.Key) == 0 {
		return "", fmt.Errorf("lazyjwt: signing key is required")
	}
	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}
	if signer.KeyID != "" {
		header["kid"] = signer.KeyID
	}
	payload := claimsPayload(claims)
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	unsigned := encode(headerJSON) + "." + encode(payloadJSON)
	return unsigned + "." + sign(unsigned, signer.Key), nil
}

// ValidatorConfig configures JWT validation.
type ValidatorConfig struct {
	Keys        map[string][]byte
	Issuer      string
	Audience    []string
	ClientRules []ClientRule
	Now         func() time.Time
}

// ClientRule constrains tokens for one OAuth client.
type ClientRule struct {
	ClientID string
	Domain   string
}

// Verify validates token and returns its claims.
func Verify(token string, config ValidatorConfig) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformedToken
	}
	headerJSON, err := decode(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("%w: header", ErrMalformedToken)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return Claims{}, fmt.Errorf("%w: header", ErrMalformedToken)
	}
	if header.Alg != "HS256" {
		return Claims{}, ErrUnsupportedAlg
	}
	key, ok := validationKey(header.Kid, config.Keys)
	if !ok {
		return Claims{}, fmt.Errorf("lazyjwt: validation key %q not found", header.Kid)
	}
	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(sign(unsigned, key)), []byte(parts[2])) {
		return Claims{}, ErrInvalidSignature
	}
	payloadJSON, err := decode(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("%w: payload", ErrMalformedToken)
	}
	claims, err := parseClaims(payloadJSON)
	if err != nil {
		return Claims{}, err
	}
	if err := validateClaims(claims, config); err != nil {
		return Claims{}, err
	}
	return claims, nil
}

func claimsPayload(claims Claims) map[string]any {
	payload := map[string]any{}
	for key, value := range claims.Extra {
		payload[key] = value
	}
	if claims.Issuer != "" {
		payload["iss"] = claims.Issuer
	}
	if claims.Subject != "" {
		payload["sub"] = claims.Subject
	}
	if len(claims.Audience) == 1 {
		payload["aud"] = claims.Audience[0]
	} else if len(claims.Audience) > 1 {
		payload["aud"] = claims.Audience
	}
	if !claims.ExpiresAt.IsZero() {
		payload["exp"] = claims.ExpiresAt.Unix()
	}
	if !claims.NotBefore.IsZero() {
		payload["nbf"] = claims.NotBefore.Unix()
	}
	if !claims.IssuedAt.IsZero() {
		payload["iat"] = claims.IssuedAt.Unix()
	}
	if claims.ID != "" {
		payload["jti"] = claims.ID
	}
	if len(claims.Scope) > 0 {
		payload["scope"] = strings.Join(claims.Scope, " ")
	}
	if claims.ClientID != "" {
		payload["client_id"] = claims.ClientID
	}
	if claims.ClientDomain != "" {
		payload["client_domain"] = claims.ClientDomain
	}
	return payload
}

func parseClaims(payloadJSON []byte) (Claims, error) {
	var payload map[string]any
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return Claims{}, fmt.Errorf("%w: payload", ErrMalformedToken)
	}
	claims := Claims{Extra: map[string]any{}}
	for key, value := range payload {
		switch key {
		case "iss":
			claims.Issuer, _ = value.(string)
		case "sub":
			claims.Subject, _ = value.(string)
		case "aud":
			claims.Audience = stringSlice(value)
		case "exp":
			claims.ExpiresAt = unixTime(value)
		case "nbf":
			claims.NotBefore = unixTime(value)
		case "iat":
			claims.IssuedAt = unixTime(value)
		case "jti":
			claims.ID, _ = value.(string)
		case "scope":
			claims.Scope = stringSlice(value)
		case "client_id":
			claims.ClientID, _ = value.(string)
		case "client_domain":
			claims.ClientDomain, _ = value.(string)
		default:
			claims.Extra[key] = value
		}
	}
	return claims, nil
}

func validateClaims(claims Claims, config ValidatorConfig) error {
	now := time.Now
	if config.Now != nil {
		now = config.Now
	}
	current := now()
	if !claims.ExpiresAt.IsZero() && !current.Before(claims.ExpiresAt) {
		return ErrExpiredToken
	}
	if !claims.NotBefore.IsZero() && current.Before(claims.NotBefore) {
		return ErrTokenNotYetValid
	}
	if config.Issuer != "" && claims.Issuer != config.Issuer {
		return ErrInvalidIssuer
	}
	if len(config.Audience) > 0 && !audienceMatches(claims.Audience, config.Audience) {
		return ErrInvalidAudience
	}
	if len(config.ClientRules) > 0 && !clientMatches(claims, config.ClientRules) {
		return ErrInvalidClient
	}
	return nil
}

func validationKey(kid string, keys map[string][]byte) ([]byte, bool) {
	if len(keys) == 0 {
		return nil, false
	}
	if kid != "" {
		key, ok := keys[kid]
		return key, ok
	}
	if key, ok := keys[""]; ok {
		return key, true
	}
	if len(keys) == 1 {
		for _, key := range keys {
			return key, true
		}
	}
	return nil, false
}

func audienceMatches(tokenAudiences []string, allowed []string) bool {
	for _, tokenAudience := range tokenAudiences {
		for _, audience := range allowed {
			if tokenAudience == audience {
				return true
			}
		}
	}
	return false
}

func clientMatches(claims Claims, rules []ClientRule) bool {
	for _, rule := range rules {
		if rule.ClientID != "" && rule.ClientID != claims.ClientID {
			continue
		}
		if rule.Domain != "" && rule.Domain != claims.ClientDomain {
			continue
		}
		return true
	}
	return false
}

func sign(unsigned string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(unsigned))
	return encode(mac.Sum(nil))
}

func encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decode(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}

func unixTime(value any) time.Time {
	switch v := value.(type) {
	case float64:
		return time.Unix(int64(v), 0)
	case json.Number:
		i, _ := v.Int64()
		return time.Unix(i, 0)
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		if i == 0 {
			return time.Time{}
		}
		return time.Unix(i, 0)
	default:
		return time.Time{}
	}
}

func stringSlice(value any) []string {
	switch v := value.(type) {
	case string:
		return strings.Fields(v)
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
