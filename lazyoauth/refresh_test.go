package lazyoauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazyauth"
	"golazy.dev/lazyjwt"
)

const (
	refreshTestIssuer   = "https://auth.example"
	refreshTestResource = "https://app.example/mcp"
)

var refreshTestSigner = lazyjwt.Signer{KeyID: "main", Key: []byte("test-signing-key")}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func newRefreshTestServer(t *testing.T, store Store, states RefreshTokenStateStore, configure ...func(*Config)) *Server {
	t.Helper()
	config := Config{
		Issuer:             refreshTestIssuer,
		Resource:           refreshTestResource,
		Store:              store,
		RefreshTokenStates: states,
		Signer:             refreshTestSigner,
	}
	for _, apply := range configure {
		apply(&config)
	}
	server, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func issueRefreshTestTokens(t *testing.T, server *Server, store Store) oauthTokenResponse {
	t.Helper()
	client := Client{
		ID:           "client",
		Name:         "Test Client",
		RedirectURIs: []string{"https://client.example/callback"},
		Domain:       "client.example",
	}
	if err := store.SaveClient(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAuthCode(context.Background(), AuthCode{
		Code:        "authorization-code",
		ClientID:    client.ID,
		RedirectURI: client.RedirectURIs[0],
		User: lazyauth.User{
			ID:   "alice",
			Data: map[string]any{"mcps": []string{"paperless"}},
		},
		Scope:     []string{"openid", "offline_access", "profile"},
		ExpiresAt: time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	return exchangeAuthorizationCode(t, server, "authorization-code")
}

func exchangeAuthorizationCode(t *testing.T, server *Server, code string) oauthTokenResponse {
	t.Helper()
	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}
	response := postToken(t, server, form)
	if response.Code != http.StatusOK {
		t.Fatalf("authorization-code status = %d body=%s", response.Code, response.Body.String())
	}
	return decodeTokenResponse(t, response)
}

func exchangeRefreshToken(t *testing.T, server *Server, token string) (*httptest.ResponseRecorder, oauthTokenResponse) {
	t.Helper()
	response := postToken(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token},
		"client_id":     {"client"},
	})
	if response.Code != http.StatusOK {
		return response, oauthTokenResponse{}
	}
	return response, decodeTokenResponse(t, response)
}

func postToken(t *testing.T, server *Server, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

func decodeTokenResponse(t *testing.T, response *httptest.ResponseRecorder) oauthTokenResponse {
	t.Helper()
	var tokens oauthTokenResponse
	if err := json.Unmarshal(response.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("token response = %s", response.Body.String())
	}
	return tokens
}

func refreshClaims(t *testing.T, token string) lazyjwt.Claims {
	t.Helper()
	claims, err := lazyjwt.Verify(token, lazyjwt.ValidatorConfig{
		Keys:     map[string][]byte{refreshTestSigner.KeyID: refreshTestSigner.Key},
		Issuer:   refreshTestIssuer,
		Audience: []string{refreshTestIssuer + defaultTokenPath},
		Type:     refreshTokenType,
	})
	if err != nil {
		t.Fatal(err)
	}
	return claims
}

func signRefreshClaims(t *testing.T, claims lazyjwt.Claims, signer lazyjwt.Signer) string {
	t.Helper()
	signer.Type = refreshTokenType
	token, err := signer.Sign(claims)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestSignedRefreshTokenSurvivesServerRestart(t *testing.T) {
	states := NewMemoryStore()
	firstStore := NewMemoryStore()
	firstServer := newRefreshTestServer(t, firstStore, states)
	issued := issueRefreshTestTokens(t, firstServer, firstStore)

	secondServer := newRefreshTestServer(t, NewMemoryStore(), states)
	response, rotated := exchangeRefreshToken(t, secondServer, issued.RefreshToken)
	if response.Code != http.StatusOK {
		t.Fatalf("refresh status = %d body=%s", response.Code, response.Body.String())
	}
	if rotated.RefreshToken == issued.RefreshToken {
		t.Fatal("refresh token did not rotate after restart")
	}

	protected := secondServer.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := lazyauth.FromContext(r.Context())
		if !ok || user.ID != "alice" {
			t.Fatalf("user = %#v ok=%v", user, ok)
		}
		claims, ok := lazyjwt.ClaimsFromContext(r.Context())
		if !ok || !claims.HasScope("offline_access") || claims.ClientID != "client" {
			t.Fatalf("claims = %#v ok=%v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+rotated.AccessToken)
	access := httptest.NewRecorder()
	protected.ServeHTTP(access, request)
	if access.Code != http.StatusNoContent {
		t.Fatalf("access status = %d body=%s", access.Code, access.Body.String())
	}
}

func TestRefreshTokenRotationAdvancesCurrentTokenID(t *testing.T) {
	store := NewMemoryStore()
	server := newRefreshTestServer(t, store, store)
	issued := issueRefreshTestTokens(t, server, store)
	initialGrant, err := server.validateRefreshToken(issued.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}

	response, rotated := exchangeRefreshToken(t, server, issued.RefreshToken)
	if response.Code != http.StatusOK {
		t.Fatalf("refresh status = %d body=%s", response.Code, response.Body.String())
	}
	rotatedGrant, err := server.validateRefreshToken(rotated.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotatedGrant.FamilyID != initialGrant.FamilyID {
		t.Fatalf("rotated family = %q, want %q", rotatedGrant.FamilyID, initialGrant.FamilyID)
	}
	if rotatedGrant.TokenID == initialGrant.TokenID {
		t.Fatal("rotated token reused the previous jti")
	}
	if got := store.Families[initialGrant.FamilyID].CurrentTokenID; got != rotatedGrant.TokenID {
		t.Fatalf("current token id = %q, want %q", got, rotatedGrant.TokenID)
	}
}

func TestRefreshTokenReplayRevokesFamily(t *testing.T) {
	store := NewMemoryStore()
	server := newRefreshTestServer(t, store, store)
	issued := issueRefreshTestTokens(t, server, store)
	grant, err := server.validateRefreshToken(issued.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	response, rotated := exchangeRefreshToken(t, server, issued.RefreshToken)
	if response.Code != http.StatusOK {
		t.Fatalf("first refresh status = %d body=%s", response.Code, response.Body.String())
	}

	replay, _ := exchangeRefreshToken(t, server, issued.RefreshToken)
	if replay.Code != http.StatusBadRequest || !strings.Contains(replay.Body.String(), ErrInvalidGrant.Error()) {
		t.Fatalf("replay status = %d body=%s", replay.Code, replay.Body.String())
	}
	if !store.Families[grant.FamilyID].Revoked {
		t.Fatal("replayed refresh token did not revoke its family")
	}
	active, _ := exchangeRefreshToken(t, server, rotated.RefreshToken)
	if active.Code != http.StatusBadRequest || !strings.Contains(active.Body.String(), ErrInvalidGrant.Error()) {
		t.Fatalf("revoked-family status = %d body=%s", active.Code, active.Body.String())
	}
}

func TestRefreshTokenRejectsExpiredGrant(t *testing.T) {
	store := NewMemoryStore()
	server := newRefreshTestServer(t, store, store)
	issued := issueRefreshTestTokens(t, server, store)
	claims := refreshClaims(t, issued.RefreshToken)
	claims.ExpiresAt = time.Now().Add(-time.Minute)
	expired := signRefreshClaims(t, claims, refreshTestSigner)

	response, _ := exchangeRefreshToken(t, server, expired)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), ErrInvalidGrant.Error()) {
		t.Fatalf("expired status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestRefreshTokenRejectsExpiredFamilyState(t *testing.T) {
	store := NewMemoryStore()
	server := newRefreshTestServer(t, store, store)
	issued := issueRefreshTestTokens(t, server, store)
	grant, err := server.validateRefreshToken(issued.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	family := store.Families[grant.FamilyID]
	family.ExpiresAt = time.Now().Add(-time.Minute)
	store.Families[grant.FamilyID] = family

	response, _ := exchangeRefreshToken(t, server, issued.RefreshToken)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), ErrInvalidGrant.Error()) {
		t.Fatalf("expired-family status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestAccessAndRefreshTokensCannotBeSubstituted(t *testing.T) {
	store := NewMemoryStore()
	server := newRefreshTestServer(t, store, store)
	issued := issueRefreshTestTokens(t, server, store)

	resource := server.Protect(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+issued.RefreshToken)
	access := httptest.NewRecorder()
	resource.ServeHTTP(access, request)
	if access.Code != http.StatusUnauthorized {
		t.Fatalf("refresh-as-access status = %d body=%s", access.Code, access.Body.String())
	}

	refresh, _ := exchangeRefreshToken(t, server, issued.AccessToken)
	if refresh.Code != http.StatusBadRequest || !strings.Contains(refresh.Body.String(), ErrInvalidGrant.Error()) {
		t.Fatalf("access-as-refresh status = %d body=%s", refresh.Code, refresh.Body.String())
	}
}

func TestRefreshTokenRejectsWrongTrustContext(t *testing.T) {
	store := NewMemoryStore()
	server := newRefreshTestServer(t, store, store)
	issued := issueRefreshTestTokens(t, server, store)

	tests := map[string]func() string{
		"issuer": func() string {
			claims := refreshClaims(t, issued.RefreshToken)
			claims.Issuer = "https://attacker.example"
			return signRefreshClaims(t, claims, refreshTestSigner)
		},
		"audience": func() string {
			claims := refreshClaims(t, issued.RefreshToken)
			claims.Audience = []string{refreshTestResource}
			return signRefreshClaims(t, claims, refreshTestSigner)
		},
		"signature": func() string {
			return signRefreshClaims(t, refreshClaims(t, issued.RefreshToken), lazyjwt.Signer{KeyID: "main", Key: []byte("wrong-key")})
		},
		"purpose": func() string {
			claims := refreshClaims(t, issued.RefreshToken)
			claims.Extra[tokenUseClaim] = accessTokenUse
			return signRefreshClaims(t, claims, refreshTestSigner)
		},
		"resource": func() string {
			claims := refreshClaims(t, issued.RefreshToken)
			claims.Extra[refreshResourceClaim] = "https://other.example/mcp"
			return signRefreshClaims(t, claims, refreshTestSigner)
		},
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			response, _ := exchangeRefreshToken(t, server, token())
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), ErrInvalidGrant.Error()) {
				t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRefreshPreservesMappedTrustContextAndForcesAccessTokenType(t *testing.T) {
	const mappedIssuer = "https://tokens.example"
	const mappedAudience = "https://custom-resource.example"
	store := NewMemoryStore()
	server := newRefreshTestServer(t, store, store, func(config *Config) {
		config.ClaimsMapper = ClaimsMapperFunc(func(_ context.Context, user lazyauth.User, client Client) (lazyjwt.Claims, error) {
			return lazyjwt.Claims{
				Issuer:       mappedIssuer,
				Audience:     []string{mappedAudience},
				Subject:      "mapped-" + user.ID,
				ClientID:     "mapped-" + client.ID,
				ClientDomain: "mapped." + client.Domain,
				Extra:        map[string]any{"role": "reader", "data": user.Data},
			}, nil
		})
		config.Validator = lazyjwt.ValidatorConfig{
			Keys:     map[string][]byte{refreshTestSigner.KeyID: refreshTestSigner.Key},
			Issuer:   mappedIssuer,
			Audience: []string{mappedAudience},
			Type:     refreshTokenType,
		}
	})
	issued := issueRefreshTestTokens(t, server, store)

	response, rotated := exchangeRefreshToken(t, server, issued.RefreshToken)
	if response.Code != http.StatusOK {
		t.Fatalf("refresh status = %d body=%s", response.Code, response.Body.String())
	}
	claims, err := lazyjwt.Verify(rotated.AccessToken, lazyjwt.ValidatorConfig{
		Keys:     map[string][]byte{refreshTestSigner.KeyID: refreshTestSigner.Key},
		Issuer:   mappedIssuer,
		Audience: []string{mappedAudience},
		Type:     accessTokenType,
	})
	if err != nil {
		t.Fatal(err)
	}
	if claims.Extra["role"] != "reader" {
		t.Fatalf("mapped claims = %#v", claims.Extra)
	}
	if claims.Subject != "mapped-alice" || claims.ClientID != "mapped-client" || claims.ClientDomain != "mapped.client.example" {
		t.Fatalf("mapped identity claims = %#v", claims)
	}

	protected := server.Protect(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+rotated.AccessToken)
	access := httptest.NewRecorder()
	protected.ServeHTTP(access, request)
	if access.Code != http.StatusNoContent {
		t.Fatalf("custom-validator access status = %d body=%s", access.Code, access.Body.String())
	}
	request.Header.Set("Authorization", "Bearer "+rotated.RefreshToken)
	confused := httptest.NewRecorder()
	protected.ServeHTTP(confused, request)
	if confused.Code != http.StatusUnauthorized {
		t.Fatalf("custom-validator refresh-as-access status = %d body=%s", confused.Code, confused.Body.String())
	}
}

func TestLegacyOpaqueRefreshTokenMigratesToSignedRotation(t *testing.T) {
	store := NewMemoryStore()
	client := Client{ID: "client", Domain: "client.example"}
	if err := store.SaveClient(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRefreshToken(context.Background(), RefreshToken{
		Token:     "refresh_legacy",
		ClientID:  client.ID,
		User:      lazyauth.User{ID: "alice"},
		Scope:     []string{"offline_access"},
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	server := newRefreshTestServer(t, store, store)
	response, migrated := exchangeRefreshToken(t, server, "refresh_legacy")
	if response.Code != http.StatusOK {
		t.Fatalf("legacy refresh status = %d body=%s", response.Code, response.Body.String())
	}
	if strings.HasPrefix(migrated.RefreshToken, "refresh_") {
		t.Fatalf("migrated refresh token remains opaque: %q", migrated.RefreshToken)
	}
	if _, err := server.validateRefreshToken(migrated.RefreshToken); err != nil {
		t.Fatalf("migrated refresh token: %v", err)
	}
}

func TestDiskStorePersistsRefreshTokenFamilyState(t *testing.T) {
	path := t.TempDir() + "/oauth.json"
	store, err := NewDiskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	family := RefreshTokenFamily{
		ID:             "family",
		CurrentTokenID: "token",
		ExpiresAt:      time.Now().Add(time.Hour),
	}
	if err := store.CreateFamily(context.Background(), family); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewDiskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Families[family.ID]; got.ID != family.ID || got.CurrentTokenID != family.CurrentTokenID || got.Revoked {
		t.Fatalf("reloaded family = %#v", got)
	}
}
