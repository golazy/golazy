package lazystorage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFilesystemPutOpenListAndURL(t *testing.T) {
	ctx := context.Background()
	storage := NewFilesystem(t.TempDir(), WithBaseURL("https://cdn.example.test/assets"))

	info, remaining, err := storage.Put(ctx, "images/card.txt", strings.NewReader("hello"), ContentType{Value: "text/plain"})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining options = %d, want 0", len(remaining))
	}
	if info.Key != "images/card.txt" {
		t.Fatalf("Key = %q, want images/card.txt", info.Key)
	}
	if info.ContentType != "text/plain" {
		t.Fatalf("ContentType = %q, want text/plain", info.ContentType)
	}
	if info.Size != 5 {
		t.Fatalf("Size = %d, want 5", info.Size)
	}
	if !strings.HasPrefix(info.Checksum, "sha256:") {
		t.Fatalf("Checksum = %q, want sha256 prefix", info.Checksum)
	}

	file, remaining, err := storage.Open(ctx, "images/card.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if len(remaining) != 0 {
		t.Fatalf("open remaining options = %d, want 0", len(remaining))
	}
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q, want hello", data)
	}

	iterator, _, err := storage.List(ctx, "images")
	if err != nil {
		t.Fatal(err)
	}
	listed, err := iterator.Next()
	if err != nil {
		t.Fatal(err)
	}
	if listed.Key != "images/card.txt" {
		t.Fatalf("listed key = %q, want images/card.txt", listed.Key)
	}
	if _, err := iterator.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("second Next error = %v, want EOF", err)
	}

	resolved, _, err := storage.URL(ctx, "images/card.txt")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.String != "https://cdn.example.test/assets/images/card.txt" {
		t.Fatalf("URL = %q, want CDN URL", resolved.String)
	}
	if !resolved.Public {
		t.Fatal("URL Public = false, want true")
	}
}

func TestFilesystemRejectsInvalidKeys(t *testing.T) {
	storage := NewFilesystem(t.TempDir())
	if _, _, err := storage.Put(context.Background(), "../secret.txt", strings.NewReader("no")); err == nil {
		t.Fatal("Put invalid key error = nil, want error")
	}
	if _, _, err := storage.Open(context.Background(), "/absolute.txt"); err == nil {
		t.Fatal("Open invalid key error = nil, want error")
	}
}
