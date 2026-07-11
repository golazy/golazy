package lazymedia

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"golazy.dev/lazystorage"
)

const LazyDevMediaPath = "/media"
const LazyDevMediaDownloadPath = "/media/download"
const LazyDevMediaStorageUploadPath = "/media/storage/upload"
const LazyDevMediaStorageDeletePath = "/media/storage/delete"
const LazyDevMediaFileUploadPath = "/media/file/upload"
const LazyDevMediaFileDeletePath = "/media/file/delete"
const LazyDevMediaVariantDeletePath = "/media/variant/delete"

// LazyDevInspector describes the media, file, and storage services exposed to
// the development panel.
type LazyDevInspector struct {
	Storages       map[string]lazystorage.Storage
	DefaultStorage string
	Files          LazyDevFileCatalog
	Media          *Media
}

// LazyDevFileCatalog is the file-service shape used by the development panel.
type LazyDevFileCatalog interface {
	ListFiles(context.Context, LazyDevFileListQuery) ([]LazyDevFileSnapshot, error)
	PutFile(context.Context, io.Reader, LazyDevPutFileOptions) (LazyDevFileSnapshot, error)
	OpenFile(context.Context, string) (io.ReadCloser, LazyDevFileSnapshot, error)
	FileURL(context.Context, string) (string, error)
	DeleteFile(context.Context, string) error
}

// LazyDevFileListQuery filters development file catalog listings.
type LazyDevFileListQuery struct {
	Storage   string
	KeyPrefix string
}

// LazyDevPutFileOptions configures development file uploads.
type LazyDevPutFileOptions struct {
	Storage     string
	Key         string
	Filename    string
	ContentType string
}

// LazyDevSnapshot is the JSON payload consumed by the development panel.
type LazyDevSnapshot struct {
	Storages                []LazyDevStorageSnapshot `json:"storages"`
	SelectedStorage         string                   `json:"selected_storage,omitempty"`
	StoragePrefix           string                   `json:"storage_prefix,omitempty"`
	StorageObjects          []LazyDevStorageObject   `json:"storage_objects,omitempty"`
	StorageObjectsError     string                   `json:"storage_objects_error,omitempty"`
	StorageObjectsTruncated bool                     `json:"storage_objects_truncated,omitempty"`
	Files                   []LazyDevFileSnapshot    `json:"files,omitempty"`
	FilesError              string                   `json:"files_error,omitempty"`
	Variants                []LazyDevVariantSnapshot `json:"variants,omitempty"`
	VariantsError           string                   `json:"variants_error,omitempty"`
}

// LazyDevStorageSnapshot describes one configured storage backend.
type LazyDevStorageSnapshot struct {
	Name      string `json:"name"`
	Default   bool   `json:"default,omitempty"`
	Readable  bool   `json:"readable"`
	Writable  bool   `json:"writable"`
	Deletable bool   `json:"deletable"`
	Listable  bool   `json:"listable"`
	URLable   bool   `json:"urlable"`
}

// LazyDevStorageObject is one object listed from a selected storage backend.
type LazyDevStorageObject struct {
	Key         string    `json:"key"`
	ContentType string    `json:"content_type,omitempty"`
	Size        int64     `json:"size,omitempty"`
	Checksum    string    `json:"checksum,omitempty"`
	ModifiedAt  time.Time `json:"modified_at"`
	URL         string    `json:"url,omitempty"`
	URLError    string    `json:"url_error,omitempty"`
}

// LazyDevFileSnapshot describes one cataloged file.
type LazyDevFileSnapshot struct {
	ID          string                   `json:"id"`
	Filename    string                   `json:"filename,omitempty"`
	ContentType string                   `json:"content_type,omitempty"`
	Size        int64                    `json:"size,omitempty"`
	Checksum    string                   `json:"checksum,omitempty"`
	Metadata    json.RawMessage          `json:"metadata,omitempty"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	DeletedAt   time.Time                `json:"deleted_at"`
	Location    LazyDevFileLocation      `json:"location"`
	Locations   []LazyDevFileLocation    `json:"locations,omitempty"`
	URL         string                   `json:"url,omitempty"`
	URLError    string                   `json:"url_error,omitempty"`
	Variants    []LazyDevVariantSnapshot `json:"variants,omitempty"`
}

// LazyDevFileLocation describes where a cataloged file's bytes live.
type LazyDevFileLocation struct {
	FileID   string `json:"file_id,omitempty"`
	Storage  string `json:"storage,omitempty"`
	Key      string `json:"key,omitempty"`
	Role     string `json:"role,omitempty"`
	Status   string `json:"status,omitempty"`
	Checksum string `json:"checksum,omitempty"`
}

// LazyDevVariantSnapshot describes one generated media variant.
type LazyDevVariantSnapshot struct {
	SourceFileID   string          `json:"source_file_id"`
	VariantKey     string          `json:"variant_key"`
	Spec           json.RawMessage `json:"spec,omitempty"`
	OutputFileID   string          `json:"output_file_id,omitempty"`
	OutputURL      string          `json:"output_url,omitempty"`
	OutputURLError string          `json:"output_url_error,omitempty"`
	Status         string          `json:"status,omitempty"`
	Error          string          `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
