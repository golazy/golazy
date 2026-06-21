package lazyfiles

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golazy.dev/lazystorage"
)

// Files coordinates a repository with named storages.
type Files struct {
	Repository     Repository
	Storages       map[string]lazystorage.Storage
	DefaultStorage string
	RoutePrefix    string
	SigningKey     []byte
}

// Put writes body to storage and records the file catalog entry.
func (f *Files) Put(ctx context.Context, body io.Reader, options ...any) (File, []any, error) {
	if f == nil {
		return File{}, options, fmt.Errorf("lazyfiles: files service is nil")
	}
	if f.Repository == nil {
		return File{}, options, fmt.Errorf("lazyfiles: repository is nil")
	}
	if body == nil {
		return File{}, options, fmt.Errorf("lazyfiles: nil body")
	}

	storageOption, options, hasStorage := lazystorage.Take[StorageName](options)
	storageName := f.DefaultStorage
	if hasStorage {
		storageName = storageOption.Name
	}
	if storageName == "" {
		return File{}, options, fmt.Errorf("lazyfiles: storage name is required")
	}
	storage, ok := f.Storages[storageName]
	if !ok {
		return File{}, options, fmt.Errorf("lazyfiles: storage %q is not configured", storageName)
	}
	writer, ok := storage.(lazystorage.Writer)
	if !ok {
		return File{}, options, fmt.Errorf("lazyfiles: storage %q cannot write", storageName)
	}

	id := newID()
	keyOption, options, hasKey := lazystorage.Take[ObjectKey](options)
	key := keyOption.Key
	if !hasKey || key == "" {
		key = id
	}
	info, options, err := writer.Put(ctx, key, body, options...)
	if err != nil {
		return File{}, options, err
	}

	filename, options, _ := lazystorage.Take[Filename](options)
	metadata, options, _ := lazystorage.Take[Metadata](options)
	file := File{
		ID:          id,
		Filename:    strings.TrimSpace(filename.Name),
		ContentType: info.ContentType,
		Size:        info.Size,
		Checksum:    info.Checksum,
		Metadata:    metadata.JSON,
	}
	location := Location{
		FileID:   id,
		Storage:  storageName,
		Key:      info.Key,
		Role:     RolePrimary,
		Status:   StatusActive,
		Checksum: info.Checksum,
	}
	return f.Repository.Put(ctx, file, location, options...)
}

// Find returns a file and its active location.
func (f *Files) Find(ctx context.Context, id string, options ...any) (StoredFile, []any, error) {
	file, locations, options, err := f.Repository.Find(ctx, Query{ID: id}, options...)
	if err != nil {
		return StoredFile{}, options, err
	}
	location, ok := activeLocation(locations)
	if !ok {
		return StoredFile{}, options, fmt.Errorf("lazyfiles: file %q has no active location", id)
	}
	return StoredFile{File: file, Location: location, Locations: locations}, options, nil
}

// Open opens a file by catalog id.
func (f *Files) Open(ctx context.Context, id string, options ...any) (lazystorage.File, File, []any, error) {
	stored, options, err := f.Find(ctx, id, options...)
	if err != nil {
		return nil, File{}, options, err
	}
	storage, ok := f.Storages[stored.Location.Storage]
	if !ok {
		return nil, File{}, options, fmt.Errorf("lazyfiles: storage %q is not configured", stored.Location.Storage)
	}
	opened, options, err := storage.Open(ctx, stored.Location.Key, options...)
	if err != nil {
		return nil, File{}, options, err
	}
	return opened, stored.File, options, nil
}

func newID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(data[:])
}
