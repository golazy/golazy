package lazyassets

import (
	"context"
	"io"
	"strings"
	"testing"

	"golazy.dev/lazystorage"
)

func TestRegistryUploadsPermanentAssetsAndManifest(t *testing.T) {
	registry := newBasicRegistry(t)
	storage := &memoryWriter{objects: map[string]string{}, cache: map[string]string{}}

	if err := registry.Upload(context.Background(), storage); err != nil {
		t.Fatal(err)
	}

	permanent, err := registry.Path("/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimPrefix(permanent, "/")
	if _, ok := storage.objects[key]; !ok {
		t.Fatalf("uploaded objects = %#v, want %s", storage.objects, key)
	}
	if _, ok := storage.objects["styles.css"]; ok {
		t.Fatalf("logical stylesheet was uploaded in permanent-only mode")
	}
	if !strings.Contains(storage.objects["manifest.json"], `"permanent"`) {
		t.Fatalf("manifest = %s, want permanent entries", storage.objects["manifest.json"])
	}
	if storage.cache[key] != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", storage.cache[key])
	}
}

type memoryWriter struct {
	objects map[string]string
	cache   map[string]string
}

func (w *memoryWriter) Put(ctx context.Context, key string, body io.Reader, options ...any) (lazystorage.Info, []any, error) {
	if err := ctx.Err(); err != nil {
		return lazystorage.Info{}, options, err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return lazystorage.Info{}, options, err
	}
	cache, remaining, _ := lazystorage.Take[lazystorage.CacheControl](options)
	w.objects[key] = string(data)
	w.cache[key] = cache.Value
	return lazystorage.Info{Key: key, Size: int64(len(data))}, remaining, nil
}
