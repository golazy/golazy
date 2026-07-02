package lazyoauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golazy.dev/lazyauth"
	"golazy.dev/lazyjwt"
)

type testAuth struct{}

func (testAuth) Authenticate(_ context.Context, credential lazyauth.Credential) (lazyauth.User, error) {
	if credential.Identifier != "alice" || credential.Secret != "secret" {
		return lazyauth.User{}, lazyauth.ErrInvalidCredentials
	}
	return lazyauth.User{ID: "alice", Data: map[string]any{"mcps": []string{"admin"}}}, nil
}

func TestAuthorizationCodePKCEAndRefreshFlow(t *testing.T) {
	store := NewMemoryStore()
	client := Client{
		ID:           "client",
		RedirectURIs: []string{"http://client.example/callback"},
		Domain:       "client.example",
	}
	if err := store.SaveClient(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	server, err := New(Config{
		Issuer:   "http://auth.example",
		Resource: "http://app.example/mcp",
		Auth:     lazyauth.Config{Authenticator: testAuth{}},
		Store:    store,
		Signer:   lazyjwt.Signer{KeyID: "main", Key: []byte("secret")},
	})
	if err != nil {
		t.Fatal(err)
	}

	authorize := httptest.NewRecorder()
	values := url.Values{
		"response_type":         {"code"},
		"client_id":             {"client"},
		"redirect_uri":          {"http://client.example/callback"},
		"code_challenge":        {"verifier"},
		"code_challenge_method": {"plain"},
		"state":                 {"state"},
		"username":              {"alice"},
		"password":              {"secret"},
		"scope":                 {"openid offline_access"},
	}
	server.ServeHTTP(authorize, httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+values.Encode(), nil))
	if authorize.Code != http.StatusFound {
		t.Fatalf("authorize status = %d", authorize.Code)
	}
	location, err := url.Parse(authorize.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := location.Query().Get("code")
	if code == "" {
		t.Fatal("authorization response has no code")
	}

	token := httptest.NewRecorder()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {"verifier"},
	}
	tokenRequest := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.ServeHTTP(token, tokenRequest)
	if token.Code != http.StatusOK {
		t.Fatalf("token status = %d body=%s", token.Code, token.Body.String())
	}
	if !strings.Contains(token.Body.String(), "access_token") {
		t.Fatalf("token response = %s", token.Body.String())
	}
}

func TestProtectValidatesBearerToken(t *testing.T) {
	server, err := New(Config{
		Issuer:   "http://auth.example",
		Resource: "http://app.example/mcp",
		Signer:   lazyjwt.Signer{KeyID: "main", Key: []byte("secret")},
	})
	if err != nil {
		t.Fatal(err)
	}
	access, err := (lazyjwt.Signer{KeyID: "main", Key: []byte("secret")}).Sign(lazyjwt.Claims{
		Issuer:   "http://auth.example",
		Subject:  "alice",
		Audience: []string{"http://app.example/mcp"},
		Extra:    map[string]any{"data": map[string]any{"email": "alice@example.com"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := lazyauth.FromContext(r.Context())
		if !ok || user.ID != "alice" {
			t.Fatalf("user = %#v ok=%v", user, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+access)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}
