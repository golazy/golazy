package lazymedia

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestLogRepositoryReplaysVariants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "variants.log.jsonl")
	repo, err := NewLogRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	spec := json.RawMessage(`{"width":1200,"height":630}`)
	if _, _, err := repo.SaveVariant(context.Background(), Variant{
		SourceFileID: "source-1",
		VariantKey:   "og",
		Spec:         spec,
		OutputFileID: "output-1",
		Status:       StatusReady,
	}); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewLogRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	variant, _, err := reopened.FindVariant(context.Background(), "source-1", "og")
	if err != nil {
		t.Fatal(err)
	}
	if variant.OutputFileID != "output-1" {
		t.Fatalf("OutputFileID = %q, want output-1", variant.OutputFileID)
	}
	if string(variant.Spec) != string(spec) {
		t.Fatalf("Spec = %s, want %s", variant.Spec, spec)
	}
}
