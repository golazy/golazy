package lazyfiles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazystorage"
)

func TestFilesPutReturnsFallbackURLAndServesFile(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewLogRepository(filepath.Join(dir, "files.log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	files := &Files{
		Repository:     repo,
		Storages:       map[string]lazystorage.Storage{"local": lazystorage.NewFilesystem(filepath.Join(dir, "objects"))},
		DefaultStorage: "local",
		RoutePrefix:    "/files",
		SigningKey:     []byte("test-key"),
	}

	file, remaining, err := files.Put(
		context.Background(),
		strings.NewReader("hello files"),
		Filename{Name: "hello.txt"},
		lazystorage.ContentType{Value: "text/plain"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining options = %d, want 0", len(remaining))
	}

	resolved, remaining, err := files.URL(context.Background(), file.ID, lazystorage.ExpiresIn{Duration: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("URL remaining options = %d, want 0", len(remaining))
	}
	if !strings.HasPrefix(resolved, "/files/") {
		t.Fatalf("URL = %q, want /files prefix", resolved)
	}
	if strings.Contains(resolved, file.ID) {
		t.Fatalf("URL = %q, want signed token instead of raw id", resolved)
	}

	request := httptest.NewRequest(http.MethodGet, resolved, nil)
	response := httptest.NewRecorder()
	files.Handler(nil).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if response.Body.String() != "hello files" {
		t.Fatalf("body = %q, want hello files", response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
}

func TestFilesUsesStorageURLWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewLogRepository(filepath.Join(dir, "files.log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	files := &Files{
		Repository:     repo,
		Storages:       map[string]lazystorage.Storage{"local": lazystorage.NewFilesystem(filepath.Join(dir, "objects"), lazystorage.WithBaseURL("https://cdn.example.test"))},
		DefaultStorage: "local",
	}
	file, _, err := files.Put(context.Background(), strings.NewReader("cdn"), ObjectKey{Key: "cdn/card.txt"})
	if err != nil {
		t.Fatal(err)
	}
	resolved, _, err := files.URL(context.Background(), file.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "https://cdn.example.test/cdn/card.txt" {
		t.Fatalf("URL = %q, want storage URL", resolved)
	}
}
