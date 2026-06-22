package s3

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazystorage"
)

func TestStoragePutOpenListDeleteAndURL(t *testing.T) {
	var objects = map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Errorf("Authorization header is empty")
		}
		key := strings.TrimPrefix(r.URL.Path, "/bucket/")
		switch r.Method {
		case http.MethodPut:
			if r.URL.Path == "/bucket" {
				w.WriteHeader(http.StatusOK)
				return
			}
			data, _ := io.ReadAll(r.Body)
			objects[key] = string(data)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if r.URL.Query().Get("list-type") == "2" {
				w.Header().Set("Content-Type", "application/xml")
				_ = xml.NewEncoder(w).Encode(listBucketResult{Contents: []listContent{{
					Key:          "assets/app.js",
					ETag:         `"etag"`,
					Size:         18,
					LastModified: time.Now().UTC().Format(time.RFC3339),
				}}})
				return
			}
			value, ok := objects[key]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/javascript")
			w.Header().Set("Content-Length", "18")
			w.Header().Set("ETag", `"etag"`)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			_, _ = w.Write([]byte(value))
		case http.MethodDelete:
			delete(objects, key)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	storage := New(
		WithEndpoint(server.URL),
		WithBucket("bucket"),
		WithCredentials("access", "secret"),
		WithPublicBaseURL("https://example.test/assets"),
	)
	ctx := context.Background()
	if err := storage.EnsureBucket(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.Put(ctx, "assets/app.js", strings.NewReader("console.log('ok');"), lazystorage.ContentType{Value: "text/javascript"}); err != nil {
		t.Fatal(err)
	}
	opened, _, err := storage.Open(ctx, "assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	data, err := io.ReadAll(opened)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "console.log('ok');" {
		t.Fatalf("opened = %q", data)
	}
	iterator, _, err := storage.List(ctx, "assets")
	if err != nil {
		t.Fatal(err)
	}
	info, err := iterator.Next()
	if err != nil {
		t.Fatal(err)
	}
	if info.Key != "assets/app.js" {
		t.Fatalf("listed key = %q", info.Key)
	}
	resolved, _, err := storage.URL(ctx, "assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.String != "https://example.test/assets/assets/app.js" || !resolved.Public {
		t.Fatalf("URL = %#v", resolved)
	}
	if _, err := storage.Delete(ctx, "assets/app.js"); err != nil {
		t.Fatal(err)
	}
}
