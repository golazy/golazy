package jsonl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golazy.dev/lazymedia"
)

var _ lazymedia.Repository = (*JSONLRepository)(nil)
var _ lazymedia.VariantLister = (*JSONLRepository)(nil)

// JSONLRepository stores variant metadata in an append-only JSONL log.
type JSONLRepository struct {
	mu       sync.RWMutex
	path     string
	variants map[string]lazymedia.Variant
}

type logEvent struct {
	Op           string            `json:"op"`
	Variant      lazymedia.Variant `json:"variant,omitempty"`
	SourceFileID string            `json:"source_file_id,omitempty"`
	VariantKey   string            `json:"variant_key,omitempty"`
	Time         time.Time         `json:"time,omitempty"`
}

// New opens or creates an append-only JSONL variant repository.
func New(path string) (*JSONLRepository, error) {
	repo := &JSONLRepository{
		path:     path,
		variants: map[string]lazymedia.Variant{},
	}
	if err := repo.replay(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *JSONLRepository) FindVariant(ctx context.Context, sourceFileID, variantKey string, options ...any) (lazymedia.Variant, []any, error) {
	if err := ctxErr(ctx); err != nil {
		return lazymedia.Variant{}, options, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	variant, ok := r.variants[variantID(sourceFileID, variantKey)]
	if !ok {
		return lazymedia.Variant{}, options, fmt.Errorf("lazymedia: variant %q/%q not found: %w", sourceFileID, variantKey, os.ErrNotExist)
	}
	return variant, options, nil
}

func (r *JSONLRepository) ListVariants(ctx context.Context, query lazymedia.VariantListQuery, options ...any) ([]lazymedia.Variant, []any, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, options, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	variants := make([]lazymedia.Variant, 0, len(r.variants))
	for _, variant := range r.variants {
		if !variantMatches(query, variant) {
			continue
		}
		variants = append(variants, variant)
	}
	sort.Slice(variants, func(i, j int) bool {
		if variants[i].SourceFileID != variants[j].SourceFileID {
			return variants[i].SourceFileID < variants[j].SourceFileID
		}
		return variants[i].VariantKey < variants[j].VariantKey
	})
	return variants, options, nil
}

func (r *JSONLRepository) SaveVariant(ctx context.Context, variant lazymedia.Variant, options ...any) (lazymedia.Variant, []any, error) {
	if err := ctxErr(ctx); err != nil {
		return lazymedia.Variant{}, options, err
	}
	if err := validateVariant(variant); err != nil {
		return lazymedia.Variant{}, options, err
	}
	now := time.Now().UTC()
	if variant.CreatedAt.IsZero() {
		if existing, ok := r.variants[variantID(variant.SourceFileID, variant.VariantKey)]; ok && !existing.CreatedAt.IsZero() {
			variant.CreatedAt = existing.CreatedAt
		} else {
			variant.CreatedAt = now
		}
	}
	variant.UpdatedAt = now
	if variant.Status == "" {
		variant.Status = lazymedia.StatusReady
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.append(logEvent{Op: "save", Variant: variant, Time: now}); err != nil {
		return lazymedia.Variant{}, options, err
	}
	r.variants[variantID(variant.SourceFileID, variant.VariantKey)] = variant
	return variant, options, nil
}

func variantMatches(query lazymedia.VariantListQuery, variant lazymedia.Variant) bool {
	if query.SourceFileID != "" && variant.SourceFileID != query.SourceFileID {
		return false
	}
	if query.VariantKey != "" && variant.VariantKey != query.VariantKey {
		return false
	}
	if strings.TrimSpace(query.Status) != "" && variant.Status != query.Status {
		return false
	}
	return true
}

func (r *JSONLRepository) DeleteVariant(ctx context.Context, sourceFileID, variantKey string, options ...any) ([]any, error) {
	if err := ctxErr(ctx); err != nil {
		return options, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.append(logEvent{Op: "delete", SourceFileID: sourceFileID, VariantKey: variantKey, Time: time.Now().UTC()}); err != nil {
		return options, err
	}
	delete(r.variants, variantID(sourceFileID, variantKey))
	return options, nil
}

func (r *JSONLRepository) replay() error {
	file, err := os.Open(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event logEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return err
		}
		switch event.Op {
		case "save":
			r.variants[variantID(event.Variant.SourceFileID, event.Variant.VariantKey)] = event.Variant
		case "delete":
			delete(r.variants, variantID(event.SourceFileID, event.VariantKey))
		}
	}
	return scanner.Err()
}

func (r *JSONLRepository) append(event logEvent) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func variantID(sourceFileID, variantKey string) string {
	return sourceFileID + "\x00" + variantKey
}

func validateVariant(variant lazymedia.Variant) error {
	if variant.SourceFileID == "" {
		return fmt.Errorf("lazymedia: source file id is required")
	}
	if variant.VariantKey == "" {
		return fmt.Errorf("lazymedia: variant key is required")
	}
	return nil
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
