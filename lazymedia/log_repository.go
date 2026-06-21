package lazymedia

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogRepository stores variant metadata in an append-only JSONL log.
type LogRepository struct {
	mu       sync.RWMutex
	path     string
	variants map[string]Variant
}

type logEvent struct {
	Op           string    `json:"op"`
	Variant      Variant   `json:"variant,omitempty"`
	SourceFileID string    `json:"source_file_id,omitempty"`
	VariantKey   string    `json:"variant_key,omitempty"`
	Time         time.Time `json:"time,omitempty"`
}

// NewLogRepository opens or creates an append-only JSONL variant repository.
func NewLogRepository(path string) (*LogRepository, error) {
	repo := &LogRepository{
		path:     path,
		variants: map[string]Variant{},
	}
	if err := repo.replay(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *LogRepository) FindVariant(ctx context.Context, sourceFileID, variantKey string, options ...any) (Variant, []any, error) {
	if err := ctxErr(ctx); err != nil {
		return Variant{}, options, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	variant, ok := r.variants[variantID(sourceFileID, variantKey)]
	if !ok {
		return Variant{}, options, fmt.Errorf("lazymedia: variant %q/%q not found: %w", sourceFileID, variantKey, os.ErrNotExist)
	}
	return variant, options, nil
}

func (r *LogRepository) SaveVariant(ctx context.Context, variant Variant, options ...any) (Variant, []any, error) {
	if err := ctxErr(ctx); err != nil {
		return Variant{}, options, err
	}
	if err := validateVariant(variant); err != nil {
		return Variant{}, options, err
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
		variant.Status = StatusReady
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.append(logEvent{Op: "save", Variant: variant, Time: now}); err != nil {
		return Variant{}, options, err
	}
	r.variants[variantID(variant.SourceFileID, variant.VariantKey)] = variant
	return variant, options, nil
}

func (r *LogRepository) DeleteVariant(ctx context.Context, sourceFileID, variantKey string, options ...any) ([]any, error) {
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

func (r *LogRepository) replay() error {
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

func (r *LogRepository) append(event logEvent) error {
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
