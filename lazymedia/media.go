package lazymedia

import (
	"context"
	"fmt"

	"golazy.dev/lazystorage"
)

// Media resolves and generates file representations.
type Media struct {
	Files      FileStore
	Repository Repository
	Processor  Processor
}

// Variant returns a ready variant, generating it when missing or requested.
func (m *Media) Variant(ctx context.Context, request Request, options ...any) (File, []any, error) {
	if err := m.validate(); err != nil {
		return File{}, options, err
	}
	request, options = applyRequestOptions(request, options)
	if err := validateRequest(request); err != nil {
		return File{}, options, err
	}

	_, options, regenerate := lazystorage.Take[Regenerate](options)
	if !regenerate {
		existing, remaining, err := m.Repository.FindVariant(ctx, request.SourceFileID, request.VariantKey, options...)
		options = remaining
		if err == nil && existing.Status == StatusReady && existing.OutputFileID != "" {
			file, options, err := m.findFile(ctx, existing.OutputFileID, options...)
			return file, options, err
		}
	}
	return m.generate(ctx, request, options...)
}

// URL returns the URL for a ready or generated variant.
func (m *Media) URL(ctx context.Context, request Request, options ...any) (string, []any, error) {
	file, options, err := m.Variant(ctx, request, options...)
	if err != nil {
		return "", options, err
	}
	return m.Files.URL(ctx, file.ID, options...)
}

// ListVariants returns repository variants when the repository supports
// enumeration.
func (m *Media) ListVariants(ctx context.Context, query VariantListQuery, options ...any) ([]Variant, []any, error) {
	if m == nil {
		return nil, options, fmt.Errorf("lazymedia: media service is nil")
	}
	if m.Repository == nil {
		return nil, options, fmt.Errorf("lazymedia: repository is nil")
	}
	lister, ok := m.Repository.(VariantLister)
	if !ok {
		return nil, options, fmt.Errorf("lazymedia: repository cannot list variants")
	}
	return lister.ListVariants(ctx, query, options...)
}

func (m *Media) generate(ctx context.Context, request Request, options ...any) (File, []any, error) {
	body, sourceFile, options, err := m.Files.Open(ctx, request.SourceFileID, options...)
	if err != nil {
		return File{}, options, err
	}
	defer body.Close()

	result, options, err := m.Processor.Process(ctx, Source{File: sourceFile, Body: body}, request, options...)
	if err != nil {
		_, _, _ = m.Repository.SaveVariant(ctx, Variant{
			SourceFileID: request.SourceFileID,
			VariantKey:   request.VariantKey,
			Spec:         request.Spec,
			Status:       StatusFailed,
			Error:        err.Error(),
		})
		return File{}, options, err
	}
	if result.Body == nil {
		return File{}, options, fmt.Errorf("lazymedia: processor returned nil body")
	}
	putOptions := append([]any{}, options...)
	putOptions = append(putOptions, result.Options...)
	if result.ContentType != "" {
		putOptions = append(putOptions, lazystorage.ContentType{Value: result.ContentType})
	}
	if result.Filename != "" {
		putOptions = append(putOptions, OutputFilename{Name: result.Filename})
	}
	file, remaining, err := m.Files.Put(ctx, result.Body, putOptions...)
	if err != nil {
		return File{}, remaining, err
	}
	_, remaining, err = m.Repository.SaveVariant(ctx, Variant{
		SourceFileID: request.SourceFileID,
		VariantKey:   request.VariantKey,
		Spec:         request.Spec,
		OutputFileID: file.ID,
		Status:       StatusReady,
	}, remaining...)
	if err != nil {
		return File{}, remaining, err
	}
	return file, remaining, nil
}

func (m *Media) findFile(ctx context.Context, id string, options ...any) (File, []any, error) {
	body, file, options, err := m.Files.Open(ctx, id, options...)
	if err != nil {
		return File{}, options, err
	}
	_ = body.Close()
	return file, options, nil
}

func (m *Media) validate() error {
	if m == nil {
		return fmt.Errorf("lazymedia: media service is nil")
	}
	if m.Files == nil {
		return fmt.Errorf("lazymedia: file store is nil")
	}
	if m.Repository == nil {
		return fmt.Errorf("lazymedia: repository is nil")
	}
	if m.Processor == nil {
		return fmt.Errorf("lazymedia: processor is nil")
	}
	return nil
}

func applyRequestOptions(request Request, options []any) (Request, []any) {
	if request.VariantKey == "" {
		if key, remaining, ok := lazystorage.Take[VariantKey](options); ok {
			request.VariantKey = key.Key
			options = remaining
		}
	}
	if len(request.Spec) == 0 {
		if spec, remaining, ok := lazystorage.Take[Spec](options); ok {
			request.Spec = spec.JSON
			options = remaining
		}
	}
	return request, options
}
