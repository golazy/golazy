package lazymedia

import (
	"encoding/json"
	"time"
)

const (
	StatusReady      = "ready"
	StatusGenerating = "generating"
	StatusFailed     = "failed"
)

// File is the minimal file metadata lazymedia needs from a file service.
type File struct {
	ID          string
	Filename    string
	ContentType string
	Size        int64
	Metadata    json.RawMessage
}

// Variant identifies a generated representation of a source file.
type Variant struct {
	SourceFileID string          `json:"source_file_id"`
	VariantKey   string          `json:"variant_key"`
	Spec         json.RawMessage `json:"spec,omitempty"`
	OutputFileID string          `json:"output_file_id,omitempty"`
	Status       string          `json:"status,omitempty"`
	Error        string          `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"created_at,omitempty"`
	UpdatedAt    time.Time       `json:"updated_at,omitempty"`
}

// Request asks for a generated representation.
type Request struct {
	SourceFileID string
	VariantKey   string
	Spec         json.RawMessage
}

// VariantKey selects a named representation.
type VariantKey struct {
	Key string
}

// Spec supplies opaque app-defined JSON for the requested representation.
type Spec struct {
	JSON json.RawMessage
}

// Regenerate bypasses an existing ready variant and generates it again.
type Regenerate struct{}

// OutputFilename requests a filename for the generated file.
type OutputFilename struct {
	Name string
}
