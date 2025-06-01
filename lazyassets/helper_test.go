package lazyassets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type tester struct {
	*testing.T
	*Server
}
type response struct {
	*testing.T
	code  int
	cache string
	etag  string
	body  string
}

func NewServerTest(t *testing.T) *tester {
	return &tester{t, &Server{Storage: &Storage{}}}
}

func (t *tester) fetch(path string) *response {
	return t.fetchWithReq(path, nil)
}

func (t *tester) fetchWithReq(path string, reqFn func(req *http.Request)) *response {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", path, nil)
	if reqFn != nil {
		reqFn(req)
	}
	t.ServeHTTP(rec, req)
	return &response{
		T:     t.T,
		code:  rec.Code,
		cache: rec.Header().Get("Cache-Control"),
		etag:  rec.Header().Get("ETag"),
		body:  rec.Body.String(),
	}
}

func (t *tester) assertPermalink(path string) string {
	t.Helper()
	perm := t.permalink(path)
	if perm == path {
		t.Errorf("Expected permalink. Got: %q", perm)
	}
	return perm
}

func (t *tester) permalinkIs(path, expected string) {
	t.Helper()
	perm := t.permalink(path)
	if perm != expected {
		t.Errorf("Expected %q. Got: %q", expected, perm)
	}
}

func (t *tester) permalink(path string) string {
	t.Helper()
	f := t.Find(path)
	if f == nil {
		return path
	}
	return f.Permalink()
}

func (r *response) BodyContains(body string) {
	r.Helper()
	if !strings.Contains(r.body, body) {
		r.Errorf("Expected %q. Got: %q", body, r.body)
	}
}

func (r *response) BodyIs(body string) {
	r.Helper()
	if r.body != body {
		r.Errorf("Expected %q. Got: %q", body, r.body)
	}
}

func (r *response) StatusIs(code int) {
	r.Helper()
	if r.code != code {
		r.Error("Expected", code, "Got", r.code)
	}
}

func (r *response) EtagIs(etag string) {
	r.Helper()
	if r.etag != etag {
		r.Errorf("Expected %q. Got: %q", etag, r.etag)
	}
}

func (r *response) CacheIs(cache string) {
	r.Helper()
	if r.cache != cache {
		r.Error("Expected", cache, "Got", r.cache)
	}
}
