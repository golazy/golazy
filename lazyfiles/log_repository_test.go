package lazyfiles

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLogRepositoryReplaysEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "files.log.jsonl")
	repo, err := NewLogRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	file := File{ID: "file-1", Filename: "card.txt"}
	location := Location{FileID: "file-1", Storage: "local", Key: "card.txt", Role: RolePrimary, Status: StatusActive}
	if _, _, err := repo.Put(context.Background(), file, location); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewLogRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	got, locations, _, err := reopened.Find(context.Background(), Query{ID: "file-1"})
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
