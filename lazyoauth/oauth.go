package lazyoauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"golazy.dev/lazyauth"
	"golazy.dev/lazyjwt"
)

var (
	ErrInvalidClient   = errors.New("lazyoauth: invalid client")
	ErrInvalidRequest  = errors.New("lazyoauth: invalid request")
	ErrInvalidGrant    = errors.New("lazyoauth: invalid grant")
	ErrUnauthorized    = errors.New("lazyoauth: unauthorized")
	ErrUnsupportedFlow = errors.New("lazyoauth: unsupported flow")
)

const (
	defaultAuthorizePath = "/oauth/authorize"
	defaultTokenPath     = "/oauth/token"
	defaultRegisterPath  = "/oauth/register"
	defaultJWKSPath      = "/oauth/jwks"
)

// Config configures an OAuth authorization server and resource server.
type Config struct {
	Issuer   string
	Resource string
	Auth     lazyauth.Config
	Store    Store
	Signer   lazyjwt.Signer
	// Validator overrides the resource-server JWT validator. When empty, a
	// validator is derived from Signer, Issuer, and Resource.
	Validator lazyjwt.ValidatorConfig

	ClaimsMapper ClaimsMapper

	AuthorizePath string
	TokenPath     string
	RegisterPath  string
	JWKSPath      string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	AllowDynamicClients bool
}

// ClaimsMapper maps an authenticated user and OAuth client to JWT claims.
type ClaimsMapper interface {
	ClaimsFor(context.Context, lazyauth.User, Client) (lazyjwt.Claims, error)
}

// ClaimsMapperFunc adapts a function to ClaimsMapper.
type ClaimsMapperFunc func(context.Context, lazyauth.User, Client) (lazyjwt.Claims, error)

func (fn ClaimsMapperFunc) ClaimsFor(ctx context.Context, user lazyauth.User, client Client) (lazyjwt.Claims, error) {
	return fn(ctx, user, client)
}

// Server serves OAuth endpoints and validates bearer tokens.
type Server struct {
	config Config
}

// New creates an OAuth server.
func New(config Config) (*Server, error) {
	if config.Issuer == "" {
		config.Issuer = "http://localhost"
	}
	config.Issuer = strings.TrimRight(config.Issuer, "/")
	if config.Resource == "" {
		config.Resource = config.Issuer + "/mcp"
	}
	if config.AuthorizePath == "" {
		config.AuthorizePath = defaultAuthorizePath
	}
	if config.TokenPath == "" {
		config.TokenPath = defaultTokenPath
	}
	if config.RegisterPath == "" {
		config.RegisterPath = defaultRegisterPath
	}
	if config.JWKSPath == "" {
		config.JWKSPath = defaultJWKSPath
	}
	if config.Store == nil {
		config.Store = NewMemoryStore()
	}
	if len(config.Signer.Key) == 0 {
		config.Signer.Key = randomBytes(32)
	}
	if config.Signer.KeyID == "" {
		config.Signer.KeyID = "default"
	}
	if config.AccessTokenTTL == 0 {
		config.AccessTokenTTL = 15 * time.Minute
	}
	if config.RefreshTokenTTL == 0 {
		config.RefreshTokenTTL = 24 * time.Hour
	}
	if config.ClaimsMapper == nil {
		config.ClaimsMapper = ClaimsMapperFunc(defaultClaims)
	}
	return &Server{config: config}, nil
}

// Protect validates bearer tokens before calling next.
func (server *Server) Protect(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := server.validateRequest(r)
		if err != nil {
			server.unauthorized(w, err)
			return
		}
		ctx := lazyjwt.WithClaims(r.Context(), claims)
		ctx = lazyauth.WithUser(ctx, userFromClaims(claims))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Handler serves OAuth endpoints and protects non-OAuth requests with bearer
// token validation.
func (server *Server) Handler(next http.Handler) http.Handler {
	protected := server.Protect(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if server.serveOAuth(w, r) {
			return
		}
		protected.ServeHTTP(w, r)
	})
}

// ServeHTTP serves OAuth metadata and endpoint requests.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if server.serveOAuth(w, r) {
		return
	}
	http.NotFound(w, r)
}

// HandlesPath reports whether path belongs to this OAuth server.
func (server *Server) HandlesPath(path string) bool {
	switch path {
	case "/.well-known/oauth-protected-resource",
		"/.well-known/oauth-authorization-server",
		"/.well-known/openid-configuration",
		server.config.JWKSPath,
		server.config.RegisterPath,
		server.config.AuthorizePath,
		server.config.TokenPath:
		return true
	default:
		return false
	}
}

func (server *Server) serveOAuth(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/.well-known/oauth-protected-resource":
		server.writeJSON(w, server.protectedResourceMetadata(r))
	case "/.well-known/oauth-authorization-server", "/.well-known/openid-configuration":
		server.writeJSON(w, server.authorizationServerMetadata(r))
	case server.config.JWKSPath:
		server.writeJSON(w, map[string]any{"keys": []any{}})
	case server.config.RegisterPath:
		if !server.config.AllowDynamicClients {
			http.NotFound(w, r)
			return true
		}
		server.registerClient(w, r)
	case server.config.AuthorizePath:
		server.authorize(w, r)
	case server.config.TokenPath:
		server.token(w, r)
	default:
		return false
	}
	return true
}

func (server *Server) validateRequest(r *http.Request) (lazyjwt.Claims, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return lazyjwt.Claims{}, ErrUnauthorized
	}
	token := strings.TrimSpace(auth[len("Bearer "):])
	validator := server.config.Validator
	if len(validator.Keys) == 0 {
		validator.Keys = map[string][]byte{server.config.Signer.KeyID: server.config.Signer.Key}
	}
	if validator.Issuer == "" {
		validator.Issuer = server.config.Issuer
	}
	if len(validator.Audience) == 0 {
		validator.Audience = []string{server.config.Resource}
	}
	return lazyjwt.Verify(token, validator)
}

func (server *Server) unauthorized(w http.ResponseWriter, err error) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource", error="invalid_token"`, server.config.Issuer))
	http.Error(w, err.Error(), http.StatusUnauthorized)
}

func (server *Server) protectedResourceMetadata(r *http.Request) map[string]any {
	resource := server.config.Resource
	if resource == "" {
		resource = origin(r) + "/mcp"
	}
	return map[string]any{
		"resource":              resource,
		"authorization_servers": []string{server.config.Issuer},
		"bearer_methods_supported": []string{
			"header",
		},
	}
}

func (server *Server) authorizationServerMetadata(r *http.Request) map[string]any {
	base := origin(r)
	if server.config.Issuer != "" {
		base = server.config.Issuer
	}
	return map[string]any{
		"issuer":                                                   server.config.Issuer,
		"authorization_endpoint":                                   base + server.config.AuthorizePath,
		"token_endpoint":                                           base + server.config.TokenPath,
		"registration_endpoint":                                    base + server.config.RegisterPath,
		"jwks_uri":                                                 base + server.config.JWKSPath,
		"response_types_supported":                                 []string{"code"},
		"grant_types_supported":                                    []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":                         []string{"S256", "plain"},
		"token_endpoint_auth_methods_supported":                    []string{"none"},
		"scopes_supported":                                         []string{"openid", "profile", "offline_access"},
		"subject_types_supported":                                  []string{"public"},
		"id_token_signing_alg_values_supported":                    []string{"HS256"},
		"authorization_response_iss_parameter_supported":           true,
		"authorization_details_types_supported":                    []string{},
		"pushed_authorization_request_endpoint":                    nil,
		"require_pushed_authorization_requests":                    false,
		"dpop_signing_alg_values_supported":                        []string{},
		"revocation_endpoint_auth_methods_supported":               []string{"none"},
		"introspection_endpoint_auth_methods_supported":            []string{"none"},
		"claims_parameter_supported":                               false,
		"request_parameter_supported":                              false,
		"request_uri_parameter_supported":                          false,
		"require_request_uri_registration":                         false,
		"tls_client_certificate_bound_access_tokens":               false,
		"mtls_endpoint_aliases":                                    map[string]any{},
		"backchannel_logout_supported":                             false,
		"frontchannel_logout_supported":                            false,
		"claims_supported":                                         []string{"sub", "iss", "aud", "exp", "iat", "client_id"},
		"ui_locales_supported":                                     []string{},
		"op_policy_uri":                                            nil,
		"op_tos_uri":                                               nil,
		"service_documentation":                                    nil,
		"display_values_supported":                                 []string{"page"},
		"claim_types_supported":                                    []string{"normal"},
		"request_object_signing_alg_values_supported":              []string{},
		"request_object_encryption_alg_values_supported":           []string{},
		"request_object_encryption_enc_values_supported":           []string{},
		"userinfo_signing_alg_values_supported":                    []string{},
		"userinfo_encryption_alg_values_supported":                 []string{},
		"userinfo_encryption_enc_values_supported":                 []string{},
		"id_token_encryption_alg_values_supported":                 []string{},
		"id_token_encryption_enc_values_supported":                 []string{},
		"authorization_signing_alg_values_supported":               []string{},
		"authorization_encryption_alg_values_supported":            []string{},
		"authorization_encryption_enc_values_supported":            []string{},
		"token_endpoint_auth_signing_alg_values_supported":         []string{},
		"revocation_endpoint_auth_signing_alg_values_supported":    []string{},
		"introspection_endpoint_auth_signing_alg_values_supported": []string{},
	}
}

func (server *Server) registerClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
		ClientURI    string   `json:"client_uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body.RedirectURIs) == 0 {
		http.Error(w, "redirect_uris is required", http.StatusBadRequest)
		return
	}
	client := Client{
		ID:           "client_" + randomToken(),
		Name:         body.ClientName,
		RedirectURIs: append([]string(nil), body.RedirectURIs...),
		Domain:       clientDomain(body.ClientURI, body.RedirectURIs[0]),
	}
	if err := server.config.Store.SaveClient(r.Context(), client); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	server.writeJSON(w, map[string]any{
		"client_id":                  client.ID,
		"client_name":                client.Name,
		"redirect_uris":              client.RedirectURIs,
		"token_endpoint_auth_method": "none",
	})
}

func (server *Server) authorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("username") == "" {
			server.loginForm(w, r)
			return
		}
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	values := r.URL.Query()
	if r.Method == http.MethodPost {
		values = url.Values(r.PostForm)
	}
	client, err := server.client(r.Context(), values.Get("client_id"), values.Get("redirect_uri"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, err := lazyauth.Authenticate(r.Context(), server.config.Auth, lazyauth.Credential{
		Kind:       "password",
		Identifier: values.Get("username"),
		Secret:     values.Get("password"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	code := "code_" + randomToken()
	authCode := AuthCode{
		Code:                code,
		ClientID:            client.ID,
		RedirectURI:         values.Get("redirect_uri"),
		CodeChallenge:       values.Get("code_challenge"),
		CodeChallengeMethod: firstNonEmpty(values.Get("code_challenge_method"), "plain"),
		User:                user,
		Scope:               strings.Fields(values.Get("scope")),
		ExpiresAt:           time.Now().Add(5 * time.Minute),
	}
	if err := server.config.Store.SaveAuthCode(r.Context(), authCode); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirect, _ := url.Parse(authCode.RedirectURI)
	q := redirect.Query()
	q.Set("code", code)
	if state := values.Get("state"); state != "" {
		q.Set("state", state)
	}
	q.Set("iss", server.config.Issuer)
	redirect.RawQuery = q.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (server *Server) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		server.authorizationCodeToken(w, r)
	case "refresh_token":
		server.refreshToken(w, r)
	default:
		http.Error(w, ErrUnsupportedFlow.Error(), http.StatusBadRequest)
	}
}

func (server *Server) authorizationCodeToken(w http.ResponseWriter, r *http.Request) {
	code, err := server.config.Store.TakeAuthCode(r.Context(), r.Form.Get("code"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !time.Now().Before(code.ExpiresAt) {
		http.Error(w, ErrInvalidGrant.Error(), http.StatusBadRequest)
		return
	}
	client, err := server.client(r.Context(), code.ClientID, code.RedirectURI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !verifyPKCE(code.CodeChallenge, code.CodeChallengeMethod, r.Form.Get("code_verifier")) {
		http.Error(w, ErrInvalidGrant.Error(), http.StatusBadRequest)
		return
	}
	server.issueTokens(w, r, client, code.User, code.Scope)
}

func (server *Server) refreshToken(w http.ResponseWriter, r *http.Request) {
	refresh, err := server.config.Store.GetRefreshToken(r.Context(), r.Form.Get("refresh_token"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !time.Now().Before(refresh.ExpiresAt) {
		http.Error(w, ErrInvalidGrant.Error(), http.StatusBadRequest)
		return
	}
	client, err := server.client(r.Context(), refresh.ClientID, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server.issueTokens(w, r, client, refresh.User, refresh.Scope)
}

func (server *Server) issueTokens(w http.ResponseWriter, r *http.Request, client Client, user lazyauth.User, scope []string) {
	claims, err := server.config.ClaimsMapper.ClaimsFor(r.Context(), user, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	claims.Issuer = firstNonEmpty(claims.Issuer, server.config.Issuer)
	claims.Subject = firstNonEmpty(claims.Subject, user.ID)
	if len(claims.Audience) == 0 {
		claims.Audience = []string{server.config.Resource}
	}
	claims.IssuedAt = now
	claims.ExpiresAt = now.Add(server.config.AccessTokenTTL)
	claims.ClientID = firstNonEmpty(claims.ClientID, client.ID)
	claims.ClientDomain = firstNonEmpty(claims.ClientDomain, client.Domain)
	claims.Scope = mergeStrings(claims.Scope, scope)
	access, err := server.config.Signer.Sign(claims)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	refresh := RefreshToken{
		Token:     "refresh_" + randomToken(),
		ClientID:  client.ID,
		User:      user,
		Scope:     scope,
		ExpiresAt: now.Add(server.config.RefreshTokenTTL),
	}
	if err := server.config.Store.SaveRefreshToken(r.Context(), refresh); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	server.writeJSON(w, map[string]any{
		"access_token":  access,
		"token_type":    "Bearer",
		"expires_in":    int(server.config.AccessTokenTTL / time.Second),
		"refresh_token": refresh.Token,
		"scope":         strings.Join(scope, " "),
	})
}

func (server *Server) client(ctx context.Context, id string, redirectURI string) (Client, error) {
	client, err := server.config.Store.GetClient(ctx, id)
	if err != nil {
		return Client{}, err
	}
	if redirectURI != "" && !slices.Contains(client.RedirectURIs, redirectURI) {
		return Client{}, ErrInvalidClient
	}
	return client, nil
}

func (server *Server) loginForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = loginTemplate.Execute(w, r.URL.Query())
}

var loginTemplate = template.Must(template.New("login").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Sign in</title></head>
<body>
<form method="post">
{{range $key, $values := .}}{{range $values}}<input type="hidden" name="{{$key}}" value="{{.}}">{{end}}{{end}}
<label>User <input name="username" autocomplete="username"></label>
<label>Password <input name="password" type="password" autocomplete="current-password"></label>
<button type="submit">Sign in</button>
</form>
</body></html>`))

func (server *Server) writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func defaultClaims(_ context.Context, user lazyauth.User, _ Client) (lazyjwt.Claims, error) {
	extra := map[string]any{"data": user.Data}
	maps.Copy(extra, user.Data)
	return lazyjwt.Claims{Subject: user.ID, Extra: extra}, nil
}

func userFromClaims(claims lazyjwt.Claims) lazyauth.User {
	data := map[string]any{}
	if raw, ok := claims.Extra["data"].(map[string]any); ok {
		maps.Copy(data, raw)
	}
	return lazyauth.User{ID: claims.Subject, Data: data}
}

func verifyPKCE(challenge string, method string, verifier string) bool {
	if challenge == "" {
		return true
	}
	switch method {
	case "", "plain":
		return challenge == verifier
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		return challenge == base64.RawURLEncoding.EncodeToString(sum[:])
	default:
		return false
	}
}

func origin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}
	return scheme + "://" + r.Host
}

func randomToken() string {
	return base64.RawURLEncoding.EncodeToString(randomBytes(32))
}

func randomBytes(size int) []byte {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return data
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func mergeStrings(a []string, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, values := range [][]string{a, b} {
		for _, value := range values {
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func clientDomain(clientURI string, redirectURI string) string {
	for _, value := range []string{clientURI, redirectURI} {
		parsed, err := url.Parse(value)
		if err == nil && parsed.Hostname() != "" {
			return parsed.Hostname()
		}
	}
	return ""
}
