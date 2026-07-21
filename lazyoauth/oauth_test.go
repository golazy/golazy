package lazyoauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
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
	if issuer := location.Query().Get("iss"); issuer != "http://auth.example" {
		t.Fatalf("authorization response issuer = %q", issuer)
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

func TestAuthorizationServerMetadataCanDisableIssuerRequirementForBrokenClients(t *testing.T) {
	server, err := New(Config{
		Issuer: "https://auth.example",
		DisableAuthorizationResponseIssuerMetadata: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"authorization_response_iss_parameter_supported":false`) {
		t.Fatalf("metadata = %s", response.Body.String())
	}
}

func TestAuthorizationServerMetadataAdvertisesConfiguredClientRegistrationMethods(t *testing.T) {
	server, err := New(Config{
		Issuer:                         "https://auth.example",
		AllowClientIDMetadataDocuments: true,
		ClientIDMetadataDocumentHosts:  []string{"chatgpt.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	var metadata map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if supported, ok := metadata["client_id_metadata_document_supported"].(bool); !ok || !supported {
		t.Fatalf("CIMD support = %#v", metadata["client_id_metadata_document_supported"])
	}
	if _, ok := metadata["registration_endpoint"]; ok {
		t.Fatalf("registration endpoint advertised while DCR is disabled: %#v", metadata["registration_endpoint"])
	}
}

func TestAuthorizationCodeFlowWithClientIDMetadataDocument(t *testing.T) {
	const redirectURI = "https://chatgpt.com/connector/oauth/paperless"
	var metadataURL string
	var requests atomic.Int32
	metadataServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/oauth/client.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"client_id":     metadataURL,
			"client_name":   "ChatGPT",
			"redirect_uris": []string{redirectURI},
		})
	}))
	t.Cleanup(metadataServer.Close)
	metadataURL = metadataServer.URL + "/oauth/client.json"
	parsedMetadataURL, err := url.Parse(metadataURL)
	if err != nil {
		t.Fatal(err)
	}

	server, err := New(Config{
		Issuer:                             "https://auth.example",
		Resource:                           "https://app.example/mcp",
		Auth:                               lazyauth.Config{Authenticator: testAuth{}},
		Signer:                             lazyjwt.Signer{KeyID: "main", Key: []byte("secret")},
		AllowClientIDMetadataDocuments:     true,
		ClientIDMetadataDocumentHosts:      []string{parsedMetadataURL.Host},
		ClientIDMetadataDocumentHTTPClient: metadataServer.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	values := url.Values{
		"response_type":         {"code"},
		"client_id":             {metadataURL},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {"verifier"},
		"code_challenge_method": {"plain"},
		"username":              {"alice"},
		"password":              {"secret"},
	}
	authorize := httptest.NewRecorder()
	server.ServeHTTP(authorize, httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+values.Encode(), nil))
	if authorize.Code != http.StatusFound {
		t.Fatalf("authorize status = %d body=%s", authorize.Code, authorize.Body.String())
	}
	location, err := url.Parse(authorize.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(location.Query()["iss"], []string{"https://auth.example"}) {
		t.Fatalf("authorization redirect = %s", location)
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {location.Query().Get("code")},
		"code_verifier": {"verifier"},
	}
	tokenRequest := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	token := httptest.NewRecorder()
	server.ServeHTTP(token, tokenRequest)
	if token.Code != http.StatusOK {
		t.Fatalf("token status = %d body=%s", token.Code, token.Body.String())
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("metadata requests = %d, want 2", got)
	}
}

func TestClientIDMetadataDocumentRejectsUnlistedRedirectURI(t *testing.T) {
	var metadataURL string
	metadataServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"client_id":     metadataURL,
			"client_name":   "ChatGPT",
			"redirect_uris": []string{"https://chatgpt.com/connector/oauth/expected"},
		})
	}))
	t.Cleanup(metadataServer.Close)
	metadataURL = metadataServer.URL + "/oauth/client.json"
	parsedMetadataURL, err := url.Parse(metadataURL)
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(Config{
		Issuer:                             "https://auth.example",
		AllowClientIDMetadataDocuments:     true,
		ClientIDMetadataDocumentHosts:      []string{parsedMetadataURL.Host},
		ClientIDMetadataDocumentHTTPClient: metadataServer.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	values := url.Values{
		"client_id":    {metadataURL},
		"redirect_uri": {"https://attacker.example/callback"},
		"username":     {"alice"},
		"password":     {"secret"},
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+values.Encode(), nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
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
