package jsonl

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"golazy.dev/lazymedia"
)

func TestJSONLRepositoryReplaysVariants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "variants.log.jsonl")
	repo, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	spec := json.RawMessage(`{"width":1200,"height":630}`)
	if _, _, err := repo.SaveVariant(context.Background(), lazymedia.Variant{
		SourceFileID: "source-1",
		VariantKey:   "og",
		Spec:         spec,
		OutputFileID: "output-1",
		Status:       lazymedia.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(path)
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

func TestJSONLRepositoryListsVariants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "variants.log.jsonl")
	repo, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []lazymedia.Variant{
		{SourceFileID: "source-1", VariantKey: "thumb", OutputFileID: "thumb-1", Status: lazymedia.StatusReady},
		{SourceFileID: "source-1", VariantKey: "og", OutputFileID: "og-1", Status: lazymedia.StatusReady},
		{SourceFileID: "source-2", VariantKey: "thumb", OutputFileID: "thumb-2", Status: lazymedia.StatusFailed},
	} {
		if _, _, err := repo.SaveVariant(context.Background(), variant); err != nil {
			t.Fatal(err)
		}
	}

	variants, _, err := repo.ListVariants(context.Background(), lazymedia.VariantListQuery{SourceFileID: "source-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 2 || variants[0].VariantKey != "og" || variants[1].VariantKey != "thumb" {
		t.Fatalf("variants = %#v, want sorted source-1 variants", variants)
	}

	variants, _, err = repo.ListVariants(context.Background(), lazymedia.VariantListQuery{Status: lazymedia.StatusFailed})
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 1 || variants[0].SourceFileID != "source-2" {
		t.Fatalf("variants = %#v, want failed source-2 variant", variants)
	}
}
