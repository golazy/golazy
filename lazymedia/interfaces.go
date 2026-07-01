package lazymedia

import (
	"context"
	"io"
)

// FileStore is the minimal file service lazymedia needs.
type FileStore interface {
	Open(context.Context, string, ...any) (io.ReadCloser, File, []any, error)
	Put(context.Context, io.Reader, ...any) (File, []any, error)
	URL(context.Context, string, ...any) (string, []any, error)
}

// Repository stores variant relationships.
type Repository interface {
	FindVariant(context.Context, string, string, ...any) (Variant, []any, error)
	SaveVariant(context.Context, Variant, ...any) (Variant, []any, error)
	DeleteVariant(context.Context, string, string, ...any) ([]any, error)
}

// VariantLister is implemented by repositories that can enumerate variants.
type VariantLister interface {
	ListVariants(context.Context, VariantListQuery, ...any) ([]Variant, []any, error)
}

// Processor generates a representation for a source file.
type Processor interface {
	Process(context.Context, Source, Request, ...any) (Result, []any, error)
}

// ProcessorFunc adapts a function to Processor.
type ProcessorFunc func(context.Context, Source, Request, ...any) (Result, []any, error)

func (fn ProcessorFunc) Process(ctx context.Context, source Source, request Request, options ...any) (Result, []any, error) {
	return fn(ctx, source, request, options...)
}

// Source is an opened source file passed to processors.
type Source struct {
	File File
	Body io.ReadCloser
}

// Result is the generated file body and metadata returned by processors.
type Result struct {
	Body        io.Reader
	ContentType string
	Filename    string
	Options     []any
}
