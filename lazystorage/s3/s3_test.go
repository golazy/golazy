package s3

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestStorageConditionalPutAndMissingObject(t *testing.T) {
	type object struct {
		body string
		etag string
	}
	objects := map[string]object{}
	etagIndex := 0
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
			current, exists := objects[key]
			if r.Header.Get("If-None-Match") == "*" && exists {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
			if match := r.Header.Get("If-Match"); match != "" && (!exists || match != `"`+current.etag+`"`) {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
			data, _ := io.ReadAll(r.Body)
			etagIndex++
			etag := "etag-" + strconv.Itoa(etagIndex)
			objects[key] = object{body: string(data), etag: etag}
			w.Header().Set("ETag", `"`+etag+`"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			current, ok := objects[key]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("ETag", `"`+current.etag+`"`)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			_, _ = w.Write([]byte(current.body))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	storage := New(
		WithEndpoint(server.URL),
		WithBucket("bucket"),
		WithCredentials("access", "secret"),
	)
	ctx := context.Background()

	info, remaining, err := storage.Put(ctx, "assets/app.js", strings.NewReader("one"), lazystorage.IfAbsent{})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining options = %#v, want none", remaining)
	}
	if info.Checksum != "etag-1" {
		t.Fatalf("checksum = %q, want response ETag", info.Checksum)
	}

	_, _, err = storage.Put(ctx, "assets/app.js", strings.NewReader("again"), lazystorage.IfAbsent{})
	if !errors.Is(err, lazystorage.ErrPreconditionFailed) {
		t.Fatalf("IfAbsent error = %v, want ErrPreconditionFailed", err)
	}
	_, _, err = storage.Put(ctx, "assets/app.js", strings.NewReader("wrong"), lazystorage.IfETag{Value: "other"})
	if !errors.Is(err, lazystorage.ErrPreconditionFailed) {
		t.Fatalf("IfETag mismatch error = %v, want ErrPreconditionFailed", err)
	}
	info, _, err = storage.Put(ctx, "assets/app.js", strings.NewReader("two"), lazystorage.IfETag{Value: "etag-1"})
	if err != nil {
		t.Fatal(err)
	}
	if info.Checksum != "etag-2" || objects["assets/app.js"].body != "two" {
		t.Fatalf("conditional overwrite info = %#v object = %#v", info, objects["assets/app.js"])
	}

	_, _, err = storage.Open(ctx, "missing.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open missing error = %v, want fs.ErrNotExist", err)
	}
}
