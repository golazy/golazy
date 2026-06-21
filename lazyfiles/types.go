package lazyfiles

import (
	"encoding/json"
	"time"
)

const (
	RolePrimary  = "primary"
	RoleMirror   = "mirror"
	RoleLegacy   = "legacy"
	StatusActive = "active"
)

// File is the catalog record for one logical stored file.
type File struct {
	ID          string          `json:"id"`
	Filename    string          `json:"filename,omitempty"`
	ContentType string          `json:"content_type,omitempty"`
	Size        int64           `json:"size,omitempty"`
	Checksum    string          `json:"checksum,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at,omitempty"`
	DeletedAt   time.Time       `json:"deleted_at,omitempty"`
}

// Location tells lazyfiles where a file's bytes live.
type Location struct {
	FileID   string `json:"file_id"`
	Storage  string `json:"storage"`
	Key      string `json:"key"`
	Role     string `json:"role,omitempty"`
	Status   string `json:"status,omitempty"`
	Checksum string `json:"checksum,omitempty"`
}

// Query identifies a file.
type Query struct {
	ID string
}

// StoredFile combines a file record with its chosen location.
type StoredFile struct {
	File      File
	Location  Location
	Locations []Location
}

// StorageName selects a named lazystorage backend.
type StorageName struct {
	Name string
}

// ObjectKey selects the storage key for a write.
type ObjectKey struct {
	Key string
}

// Filename sets the file's display filename.
type Filename struct {
	Name string
}

// Metadata sets opaque file metadata.
type Metadata struct {
	JSON json.RawMessage
}
