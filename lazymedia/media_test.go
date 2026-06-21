package lazymedia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"golazy.dev/lazystorage"
)

func TestMediaGeneratesMissingVariantAndReturnsURL(t *testing.T) {
	store := newMemoryFileStore()
	source := store.add("source", "hello", "text/plain")
	repo, err := NewLogRepository(t.TempDir() + "/variants.log.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	media := &Media{
		Files:      store,
		Repository: repo,
		Processor: ProcessorFunc(func(_ context.Context, source Source, request Request, options ...any) (Result, []any, error) {
			calls++
			data, err := io.ReadAll(source.Body)
			if err != nil {
				return Result{}, options, err
			}
			return Result{
				Body:        strings.NewReader(string(data) + "-" + request.VariantKey),
				ContentType: "text/plain",
				Filename:    "hello-" + request.VariantKey + ".txt",
			}, options, nil
		}),
	}

	url, remaining, err := media.URL(context.Background(), Request{
		SourceFileID: source.ID,
		VariantKey:   "og",
		Spec:         json.RawMessage(`{"kind":"og"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining options = %d, want 0", len(remaining))
	}
	if calls != 1 {
		t.Fatalf("processor calls = %d, want 1", calls)
	}
	if !strings.HasPrefix(url, "/files/") {
		t.Fatalf("URL = %q, want /files prefix", url)
	}

	variant, _, err := repo.FindVariant(context.Background(), source.ID, "og")
	if err != nil {
		t.Fatal(err)
	}
	output, ok := store.files[variant.OutputFileID]
	if !ok {
		t.Fatalf("output file %q not stored", variant.OutputFileID)
	}
	if output.body != "hello-og" {
		t.Fatalf("output body = %q, want hello-og", output.body)
	}
}

func TestMediaReusesReadyVariantUnlessRegenerateRequested(t *testing.T) {
	store := newMemoryFileStore()
	source := store.add("source", "hello", "text/plain")
	repo, err := NewLogRepository(t.TempDir() + "/variants.log.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	media := &Media{
		Files:      store,
		Repository: repo,
		Processor: ProcessorFunc(func(_ context.Context, source Source, request Request, options ...any) (Result, []any, error) {
			calls++
			return Result{
				Body:        strings.NewReader(fmt.Sprintf("generated-%d", calls)),
				ContentType: "text/plain",
			}, options, nil
		}),
	}

	first, _, err := media.Variant(context.Background(), Request{SourceFileID: source.ID, VariantKey: "square"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := media.Variant(context.Background(), Request{SourceFileID: source.ID, VariantKey: "square"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("second variant id = %q, want %q", second.ID, first.ID)
	}
	if calls != 1 {
		t.Fatalf("processor calls = %d, want 1", calls)
	}

	third, _, err := media.Variant(context.Background(), Request{SourceFileID: source.ID, VariantKey: "square"}, Regenerate{})
	if err != nil {
		t.Fatal(err)
	}
	if third.ID == first.ID {
		t.Fatalf("regenerated id = %q, want new id", third.ID)
	}
	if calls != 2 {
		t.Fatalf("processor calls = %d, want 2", calls)
	}
}

type memoryFile struct {
	File
	body string
}

type memoryFileStore struct {
	files map[string]memoryFile
	next  int
}

func newMemoryFileStore() *memoryFileStore {
	return &memoryFileStore{files: map[string]memoryFile{}}
}

func (s *memoryFileStore) add(filename, body, contentType string) File {
	s.next++
	file := File{
		ID:          fmt.Sprintf("file-%d", s.next),
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(body)),
	}
	s.files[file.ID] = memoryFile{File: file, body: body}
	return file
}

func (s *memoryFileStore) Open(_ context.Context, id string, options ...any) (io.ReadCloser, File, []any, error) {
	file, ok := s.files[id]
	if !ok {
		return nil, File{}, options, fmt.Errorf("missing file %s", id)
	}
	return io.NopCloser(strings.NewReader(file.body)), file.File, options, nil
}

func (s *memoryFileStore) Put(_ context.Context, body io.Reader, options ...any) (File, []any, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return File{}, options, err
	}
	contentType, options, _ := lazystorage.Take[lazystorage.ContentType](options)
	filename, options, _ := lazystorage.Take[OutputFilename](options)
	s.next++
	file := File{
		ID:          fmt.Sprintf("file-%d", s.next),
		Filename:    filename.Name,
		ContentType: contentType.Value,
		Size:        int64(len(data)),
	}
	s.files[file.ID] = memoryFile{File: file, body: string(bytes.Clone(data))}
	return file, options, nil
}

func (s *memoryFileStore) URL(_ context.Context, id string, options ...any) (string, []any, error) {
	if _, ok := s.files[id]; !ok {
		return "", options, fmt.Errorf("missing file %s", id)
	}
	return "/files/" + id, options, nil
}
