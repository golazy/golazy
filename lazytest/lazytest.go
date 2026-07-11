package lazytest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"golazy.dev/lazyapp"
	"golazy.dev/lazyroutes"
)

type Option func(*App)

type App struct {
	t       testing.TB
	Handler http.Handler
	Router  *lazyroutes.Scope
}

type Client struct {
	app     *App
	cookies []*http.Cookie
}

type Response struct {
	tb       testing.TB
	Recorder *httptest.ResponseRecorder
	Result   *http.Response
	Request  *http.Request
}

type Case struct {
	Name        string
	Method      string
	Path        string
	Headers     http.Header
	Status      int
	Contains    []string
	NotContains []string
	ContentType string
	Allow       []string
}

type RequestOption func(*http.Request)

func WithRouter(router *lazyroutes.Scope) Option {
	return func(app *App) {
		app.Router = router
	}
}

func New(t testing.TB, app *lazyapp.App, opts ...Option) *App {
	if t == nil {
		panic("lazytest: testing.TB is required")
	}
	if app == nil {
		t.Fatal("lazytest: app is required")
	}
	lazyApp := &App{t: t, Handler: app, Router: app.Router}
	for _, option := range opts {
		option(lazyApp)
	}
	return lazyApp
}

func FromHandler(t testing.TB, handler http.Handler, opts ...Option) *App {
	if t == nil {
		panic("lazytest: testing.TB is required")
	}
	if handler == nil {
		t.Fatal("lazytest: handler is required")
	}
	lazyApp := &App{t: t, Handler: handler}
	for _, option := range opts {
		option(lazyApp)
	}
	return lazyApp
}

func (app *App) Routes() lazyroutes.RouteTable {
	app.requireT()
	if app.Router == nil {
		return nil
	}
	routes := make(lazyroutes.RouteTable, len(app.Router.Routes))
	copy(routes, app.Router.Routes)
	return routes
}

func (app *App) PathFor(name string, values ...any) string {
	app.requireT()
	if app.Router == nil {
		app.t.Fatalf("lazytest: PathFor requires a router but none is configured")
	}
	path, err := app.Router.PathFor(name, values...)
	if err != nil {
		app.t.Fatalf("lazytest: route %q: %v", name, err)
	}
	return path
}

func (app *App) Client() *Client {
	app.requireT()
	return &Client{app: app}
}

func (app *App) Do(method, target string, body io.Reader, opts ...RequestOption) *Response {
	return app.do(app.t, method, target, body, opts...)
}

func (app *App) Get(target string, opts ...RequestOption) *Response {
	return app.Do(http.MethodGet, target, nil, opts...)
}

func (app *App) Post(target string, body io.Reader, opts ...RequestOption) *Response {
	return app.Do(http.MethodPost, target, body, opts...)
}

func (app *App) PostForm(target string, values url.Values, opts ...RequestOption) *Response {
	opts = append([]RequestOption{Header("Content-Type", "application/x-www-form-urlencoded")}, opts...)
	if values == nil {
		return app.Post(target, strings.NewReader(""), opts...)
	}
	return app.Post(target, strings.NewReader(values.Encode()), opts...)
}

func (app *App) GetPath(name string, values ...any) *Response {
	return app.Get(app.PathFor(name, values...))
}

func (app *App) Check(cases ...Case) {
	app.requireT()
	tester, hasRun := app.t.(interface {
		testing.TB
		Run(string, func(*testing.T)) bool
	})
	for _, testCase := range cases {
		if hasRun {
			tester.Run(caseName(testCase), func(t *testing.T) {
				app.runCase(testCase, t)
			})
			continue
		}
		app.runCase(testCase, app.t)
	}
}

func (app *App) do(tb testing.TB, method, target string, body io.Reader, opts ...RequestOption) *Response {
	app.requireT()
	if tb == nil {
		tb = app.t
	}
	tb.Helper()
	request := httptest.NewRequest(method, target, body)
	for _, option := range opts {
		option(request)
	}
	recorder := httptest.NewRecorder()
	app.Handler.ServeHTTP(recorder, request)
	return &Response{
		tb:       tb,
		Request:  request,
		Recorder: recorder,
		Result:   recorder.Result(),
	}
}

func (app *App) requireT() {
	if app == nil || app.t == nil {
		panic("lazytest: testing.TB is required")
	}
	if app.Handler == nil {
		app.t.Fatal("lazytest: handler is required")
	}
}

func (app *App) runCase(testCase Case, tb testing.TB) {
	app.ensureHandler(tb)

	method := strings.TrimSpace(testCase.Method)
	if method == "" {
		method = http.MethodGet
	}
	if testCase.Path == "" {
		tb.Fatalf("lazytest case %q missing path", caseName(testCase))
	}
	response := app.do(tb, method, testCase.Path, nil, requestOptionsFromHeaders(testCase.Headers)...)

	if testCase.Status > 0 {
		response.Status(testCase.Status)
	}
	for _, expected := range testCase.Contains {
		response.Contains(expected)
	}
	for _, unexpected := range testCase.NotContains {
		response.NotContains(unexpected)
	}
	if testCase.ContentType != "" {
		response.ContentType(testCase.ContentType)
	}
	for _, expected := range testCase.Allow {
		response.HeaderContains("Allow", expected)
	}
}

func (app *App) ensureHandler(tb testing.TB) {
	app.requireT()
	if tb == nil {
		tb = app.t
	}
	if app.Handler == nil {
		tb.Fatal("lazytest: handler is required")
	}
}

func caseName(testCase Case) string {
	if name := strings.TrimSpace(testCase.Name); name != "" {
		return name
	}
	return testCase.Method + " " + testCase.Path
}

func requestOptionsFromHeaders(headers http.Header) []RequestOption {
	if len(headers) == 0 {
		return nil
	}
	options := make([]RequestOption, 0, len(headers))
	for name, values := range headers {
		for _, value := range values {
			options = append(options, Header(name, value))
		}
	}
	return options
}

func Header(name, value string) RequestOption {
	return func(request *http.Request) {
		request.Header.Set(name, value)
	}
}

func Accept(value string) RequestOption {
	return Header("Accept", value)
}

func Cookie(cookie *http.Cookie) RequestOption {
	return func(request *http.Request) {
		if cookie != nil {
			request.AddCookie(cookie)
		}
	}
}

func Cookies(cookies ...*http.Cookie) RequestOption {
	return func(request *http.Request) {
		for _, cookie := range cookies {
			if cookie != nil {
				request.AddCookie(cookie)
			}
		}
	}
}

func BasicAuth(username, password string) RequestOption {
	return func(request *http.Request) {
		request.SetBasicAuth(username, password)
	}
}

func (response *Response) Status(code int) *Response {
	response.tb.Helper()
	if got, want := response.Result.StatusCode, code; got != want {
		response.tb.Fatalf("status = %d, want %d; body: %s", got, want, response.BodyString())
	}
	return response
}

func (response *Response) OK() *Response {
	return response.Status(http.StatusOK)
}

func (response *Response) NotFound() *Response {
	return response.Status(http.StatusNotFound)
}

func (response *Response) MethodNotAllowed() *Response {
	return response.Status(http.StatusMethodNotAllowed)
}

func (response *Response) BodyString() string {
	return response.Recorder.Body.String()
}

func (response *Response) BodyBytes() []byte {
	bytes := response.Recorder.Body.Bytes()
	copied := make([]byte, len(bytes))
	copy(copied, bytes)
	return copied
}

func (response *Response) Contains(value string) *Response {
	response.tb.Helper()
	if !strings.Contains(response.BodyString(), value) {
		response.tb.Fatalf("body %q does not contain %q", response.BodyString(), value)
	}
	return response
}

func (response *Response) NotContains(value string) *Response {
	response.tb.Helper()
	if strings.Contains(response.BodyString(), value) {
		response.tb.Fatalf("body unexpectedly contains %q", value)
	}
	return response
}

func (response *Response) Match(pattern string) []string {
	response.tb.Helper()
	regex, err := regexp.Compile(pattern)
	if err != nil {
		response.tb.Fatalf("parse regex %q: %v", pattern, err)
	}
	matches := regex.FindStringSubmatch(response.BodyString())
	if matches == nil {
		response.tb.Fatalf("body does not match %q", pattern)
	}
	return matches
}

func (response *Response) Header(name string) string {
	return response.Result.Header.Get(name)
}

func (response *Response) HeaderEquals(name, value string) *Response {
	response.tb.Helper()
	if got := response.Header(name); got != value {
		response.tb.Fatalf("header %q = %q, want %q", name, got, value)
	}
	return response
}

func (response *Response) HeaderContains(name, value string) *Response {
	response.tb.Helper()
	if got := response.Header(name); !strings.Contains(got, value) {
		response.tb.Fatalf("header %q = %q does not contain %q", name, got, value)
	}
	return response
}

func (response *Response) ContentType(value string) *Response {
	return response.HeaderContains("Content-Type", value)
}

func (response *Response) JSON(target any) *Response {
	response.tb.Helper()
	if target == nil {
		response.tb.Fatal("lazytest: JSON target must not be nil")
	}
	if err := json.Unmarshal(response.BodyBytes(), target); err != nil {
		response.tb.Fatalf("parse JSON response: %v; body: %s", err, response.BodyString())
	}
	return response
}

func (response *Response) Cookies() []*http.Cookie {
	cookies := response.Result.Cookies()
	out := make([]*http.Cookie, len(cookies))
	for i, cookie := range cookies {
		out[i] = cloneCookie(cookie)
	}
	return out
}

func (client *Client) Do(method, target string, body io.Reader, opts ...RequestOption) *Response {
	if client == nil || client.app == nil {
		panic("lazytest: client is not initialized")
	}
	opts = append([]RequestOption{Cookies(client.Cookies()...)}, opts...)
	response := client.app.do(client.app.t, method, target, body, opts...)
	client.mergeCookies(response.Cookies())
	return response
}

func (client *Client) Get(target string, opts ...RequestOption) *Response {
	return client.Do(http.MethodGet, target, nil, opts...)
}

func (client *Client) Post(target string, body io.Reader, opts ...RequestOption) *Response {
	return client.Do(http.MethodPost, target, body, opts...)
}

func (client *Client) PostForm(target string, values url.Values, opts ...RequestOption) *Response {
	opts = append([]RequestOption{Header("Content-Type", "application/x-www-form-urlencoded")}, opts...)
	if values == nil {
		return client.Post(target, strings.NewReader(""), opts...)
	}
	return client.Post(target, strings.NewReader(values.Encode()), opts...)
}

func (client *Client) Cookies() []*http.Cookie {
	client.ensureInitialized()
	cookies := make([]*http.Cookie, len(client.cookies))
	for i, cookie := range client.cookies {
		cookies[i] = cloneCookie(cookie)
	}
	return cookies
}

func (client *Client) SetCookie(cookie *http.Cookie) {
	client.ensureInitialized()
	if cookie == nil {
		return
	}
	client.mergeCookies([]*http.Cookie{cloneCookie(cookie)})
}

func (client *Client) ensureInitialized() {
	if client == nil {
		panic("lazytest: client is not initialized")
	}
	if client.app == nil {
		panic("lazytest: client app is required")
	}
	client.app.requireT()
}

func (client *Client) mergeCookies(newCookies []*http.Cookie) {
	client.ensureInitialized()
	for _, newCookie := range newCookies {
		if newCookie == nil {
			continue
		}
		found := false
		for index, existingCookie := range client.cookies {
			if sameCookie(existingCookie, newCookie) {
				client.cookies[index] = cloneCookie(newCookie)
				found = true
				break
			}
		}
		if !found {
			client.cookies = append(client.cookies, cloneCookie(newCookie))
		}
	}
}

func sameCookie(a, b *http.Cookie) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name && a.Domain == b.Domain && a.Path == b.Path
}

func cloneCookie(cookie *http.Cookie) *http.Cookie {
	if cookie == nil {
		return nil
	}
	return &http.Cookie{
		Name:       cookie.Name,
		Value:      cookie.Value,
		Path:       cookie.Path,
		Domain:     cookie.Domain,
		Expires:    cookie.Expires,
		RawExpires: cookie.RawExpires,
		MaxAge:     cookie.MaxAge,
		Secure:     cookie.Secure,
		HttpOnly:   cookie.HttpOnly,
		SameSite:   cookie.SameSite,
		Raw:        cookie.Raw,
		Unparsed:   append([]string(nil), cookie.Unparsed...),
	}
}
