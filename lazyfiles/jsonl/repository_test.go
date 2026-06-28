package jsonl

import (
	"context"
	"path/filepath"
	"testing"

	"golazy.dev/lazyfiles"
)

func TestJSONLRepositoryReplaysEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "files.log.jsonl")
	repo, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	file := lazyfiles.File{ID: "file-1", Filename: "card.txt"}
	location := lazyfiles.Location{FileID: "file-1", Storage: "local", Key: "card.txt", Role: lazyfiles.RolePrimary, Status: lazyfiles.StatusActive}
	if _, _, err := repo.Put(context.Background(), file, location); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	got, locations, _, err := reopened.Find(context.Background(), lazyfiles.Query{ID: "file-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Filename != "card.txt" {
		t.Fatalf("Filename = %q, want card.txt", got.Filename)
	}
	if len(locations) != 1 || locations[0].Key != "card.txt" {
		t.Fatalf("locations = %#v, want card.txt", locations)
	}
}
