package lazyfiles

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golazy.dev/lazystorage"
)

type tokenPayload struct {
	ID        string `json:"id"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// URL returns a storage URL when available, otherwise an application route URL.
func (f *Files) URL(ctx context.Context, id string, options ...any) (string, []any, error) {
	stored, options, err := f.Find(ctx, id, options...)
	if err != nil {
		return "", options, err
	}
	if storage, ok := f.Storages[stored.Location.Storage]; ok {
		if urler, ok := storage.(lazystorage.URLer); ok {
			resolved, remaining, err := urler.URL(ctx, stored.Location.Key, options...)
			if err == nil && resolved.String != "" {
				return resolved.String, remaining, nil
			}
			options = remaining
		}
	}
	return f.routeURL(stored.File.ID, options...)
}

func (f *Files) routeURL(id string, options ...any) (string, []any, error) {
	prefix := strings.TrimRight(f.RoutePrefix, "/")
	if prefix == "" {
		prefix = "/_lazy/files"
	}
	token, options, err := f.token(id, options...)
	if err != nil {
		return "", options, err
	}
	return prefix + "/" + url.PathEscape(token), options, nil
}

func (f *Files) token(id string, options ...any) (string, []any, error) {
	if len(f.SigningKey) == 0 {
		return id, options, nil
	}
	payload := tokenPayload{ID: id}
	if expiresAt, remaining, ok := lazystorage.Take[lazystorage.ExpiresAt](options); ok {
		options = remaining
		payload.ExpiresAt = expiresAt.Time.Unix()
	} else if expiresIn, remaining, ok := lazystorage.Take[lazystorage.ExpiresIn](options); ok {
		options = remaining
		payload.ExpiresAt = time.Now().Add(expiresIn.Duration).Unix()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", options, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, f.SigningKey)
	_, _ = mac.Write([]byte(encoded))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, options, nil
}

func (f *Files) verifyToken(token string) (string, error) {
	if len(f.SigningKey) == 0 {
		if token == "" {
			return "", fmt.Errorf("lazyfiles: empty file token")
		}
		return token, nil
	}
	encoded, signature, ok := strings.Cut(token, ".")
	if !ok {
		return "", fmt.Errorf("lazyfiles: invalid file token")
	}
	mac := hmac.New(sha256.New, f.SigningKey)
	_, _ = mac.Write([]byte(encoded))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return "", fmt.Errorf("lazyfiles: invalid file token signature")
	}
	if !hmac.Equal(got, want) {
		return "", fmt.Errorf("lazyfiles: invalid file token signature")
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("lazyfiles: invalid file token payload")
	}
	var payload tokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("lazyfiles: invalid file token payload")
	}
	if payload.ID == "" {
		return "", fmt.Errorf("lazyfiles: invalid file token payload")
	}
	if payload.ExpiresAt != 0 && time.Now().Unix() > payload.ExpiresAt {
		return "", fmt.Errorf("lazyfiles: expired file token")
	}
	return payload.ID, nil
}
