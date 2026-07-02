package fileauth

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golazy.dev/lazyauth"
)

const (
	defaultIterations = 120000
	saltSize          = 16
	hashSize          = 32
)

// Provider authenticates users from a JSON Lines file.
type Provider struct {
	users map[string]record
}

type record struct {
	ID           string         `json:"id"`
	PasswordHash string         `json:"password_hash"`
	Data         map[string]any `json:"data"`
}

// Open loads users from path.
func Open(path string) (*Provider, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	users := map[string]record{}
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var rec record
		if err := json.Unmarshal([]byte(text), &rec); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if rec.ID == "" {
			return nil, fmt.Errorf("%s:%d: user id is required", path, line)
		}
		users[rec.ID] = rec
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &Provider{users: users}, nil
}

// MustOpen loads users from path or panics.
func MustOpen(path string) *Provider {
	provider, err := Open(path)
	if err != nil {
		panic(err)
	}
	return provider
}

// Authenticate implements lazyauth.Authenticator.
func (p *Provider) Authenticate(_ context.Context, credential lazyauth.Credential) (lazyauth.User, error) {
	if p == nil {
		return lazyauth.User{}, fmt.Errorf("fileauth: provider is nil")
	}
	if credential.Kind != "" && credential.Kind != "password" {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	rec, ok := p.users[credential.Identifier]
	if !ok {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	if !VerifyPassword(rec.PasswordHash, credential.Secret) {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	return lazyauth.User{ID: rec.ID, Data: copyData(rec.Data)}, nil
}

// HashPassword hashes password with PBKDF2-HMAC-SHA256.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := pbkdf2SHA256([]byte(password), salt, defaultIterations, hashSize)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s",
		defaultIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword checks password against an encoded PBKDF2 hash.
func VerifyPassword(encoded string, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(password), salt, iterations, len(want))
	return hmac.Equal(got, want)
}

func pbkdf2SHA256(password []byte, salt []byte, iterations int, keyLen int) []byte {
	var out []byte
	block := 1
	for len(out) < keyLen {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
		block++
	}
	return out[:keyLen]
}

func copyData(data map[string]any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	copied := make(map[string]any, len(data))
	for key, value := range data {
		copied[key] = value
	}
	return copied
}
