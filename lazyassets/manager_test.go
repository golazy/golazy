package lazyassets

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed test_assets/*
var TestAssetsFS embed.FS

func TestManager_ByPath(t *testing.T) {
	m := New()
	m.AddFS(TestAssetsFS, "test_assets")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/@test/hello.world", nil)

	m.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected 200. Got %d", rec.Code)
	}
	if rec.Body.String() != "hi" {
		t.Errorf("Expected hi. Got: %q", rec.Body.String())
	}

}

func TestManager_Permalink(t *testing.T) {
	m := New()
	m.AddFS(TestAssetsFS, "test_assets")
	p, f := m.Permalink("@test/hello.world")
	if f == nil {
		t.Fatal(f)
	}

	expected := "/@test/hello-0791006df812.world"
	if p != expected {
		t.Errorf("Expected %q. Got: %q", expected, p)
	}

	p, f = m.Permalink("asdf")
	if f != nil || p != "" {
		t.Errorf("Expected ErrNotFound. Got: %v", f)
	}

}

func TestManager_ByPermalink(t *testing.T) {
	m := New()
	m.AddFS(TestAssetsFS, "test_assets")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/@test/hello-0791006df812.world", nil)

	// First request should fill the cache
	m.ServeHTTP(rec, req)

	// The second should use the cache
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Expected 200. Got %d", rec.Code)
	}
}

func TestManager_CacheControl(t *testing.T) {
	m := New()
	m.AddFS(TestAssetsFS, "test_assets")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/@test/hello.world", nil)

	m.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "" {
		t.Error("Expected Cache-Control header to be empty", rec.Header().Get("Cache-Control"))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/@test/hello-0791006df812.world", nil)
	m.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "public, max-age=31536000" {
		t.Error("Expected Cache-Control header", rec.Header().Get("Cache-Control"))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/@test/hello-0791006df812.world", nil)
	m.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "public, max-age=31536000" {
		t.Error("Expected Cache-Control header", rec.Header().Get("Cache-Control"))
	}
}

func TestManager_ETag(t *testing.T) {
	m := New()
	m.AddFS(TestAssetsFS, "test_assets")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/@test/hello.world", nil)

	m.ServeHTTP(rec, req)

	etag := `"0791006df8128477244f53d0fdce210db81f55757510e26acee35c18a6bceaa28dcdbbfd6dc041b9b4dc7b1b54e37f52"`

	if rec.Header().Get("ETag") != etag {
		t.Errorf("Expected ETag header %q. Got: %q", etag, rec.Header().Get("ETag"))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/@test/hello-0791006df812.world", nil)
	m.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != etag {
		t.Errorf("Expected ETag header %q. Got: %q", etag, rec.Header().Get("ETag"))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/@test/hello-0791006df812.world", nil)
	m.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 200 {
		t.Errorf("Expected 200. Got: %d", rec.Result().StatusCode)
	}

	if rec.Header().Get("ETag") != etag {
		t.Errorf("Expected ETag header %q. Got: %q", etag, rec.Header().Get("ETag"))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/@test/hello-0791006df812.world", nil)
	req.Header.Set("If-None-Match", etag)
	m.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusNotModified {
		t.Errorf("Expected 304. Got: %d", rec.Result().StatusCode)
	}

}

func TestWithoutHash(t *testing.T) {

	_, _, err := withoutHash("hello.world")
	if err != errNoHash {
		t.Errorf("Expected %q. Got: %q", errNoHash, err)
	}

	p, hash, err := withoutHash("asdf-123asdf.js")
	if err != nil {
		t.Error(err)
	}
	if hash != "123asdf" {
		t.Errorf("Expected %q. Got: %q", "123asdf", hash)
	}

	if p != "asdf.js" {
		t.Errorf("Expected %q. Got: %q", "asdf.js", p)
	}

	// With path
	p, hash, err = withoutHash("/js/asdf-zxcv.js")
	if err != nil {
		t.Error(err)
	}
	if hash != "zxcv" {
		t.Errorf("Expected %q. Got: %q", "zxcv", hash)
	}
	if p != "/js/asdf.js" {
		t.Errorf("Expected %q. Got: %q", "/js/asdf.js", p)
	}

	// No extension
	p, hash, err = withoutHash("/data/asdf-123")
	if err != nil {
		t.Error(err)
	}
	if p != "/data/asdf" {
		t.Errorf("Expected %q. Got: %q", "/data/asdf", p)
	}
	if hash != "123" {
		t.Errorf("Expected %q. Got: %q", "123", hash)
	}

	// Ending in underscore
	_, _, err = withoutHash("/data/asdf-")
	if err != errNoHash {
		t.Error(err)
	}

}

func TestManager_Find(t *testing.T) {

	m := New()
	m.AddFS(TestAssetsFS, "test_assets")

	f := m.Find("missing")
	if f != nil {
		t.Error("Expected nil. Got:", f)
	}

	f = m.Find("/@test/hello-0791006df812.world")
	if f == nil {
		t.Error("Expected file. Got:", f)
	}
	if !f.Permalink {
		t.Error("Expected file to be a permalink.")
	}

	f = m.Find("/@test/hello-0791006df812.world")
	if f == nil {
		t.Error("Expected file. Got:", f)
	}
	if !f.Permalink {
		t.Error("Expected file to be a permalink. Got:")
	}

	f = m.Find("/@test/hello.world")
	if f == nil {
		t.Error("Expected file. Got:", f)
	}
	if f.Permalink {
		t.Error("Expected file to not be a permalink. Got:")
	}

}
