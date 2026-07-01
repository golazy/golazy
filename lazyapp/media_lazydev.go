//go:build lazydev

package lazyapp

import (
	"context"
	"errors"
	"io"
	"os"

	"golazy.dev/lazyfiles"
	"golazy.dev/lazymedia"
	"golazy.dev/lazystorage"
)

func lazyDevMediaInspector(media mediaServices) *lazymedia.LazyDevInspector {
	inspector := &lazymedia.LazyDevInspector{
		Storages:       media.Storages,
		DefaultStorage: media.DefaultStorage,
		Media:          media.Media,
	}
	if media.Files != nil {
		inspector.Files = lazyDevFileCatalog{files: media.Files}
	}
	return inspector
}

type lazyDevFileCatalog struct {
	files *lazyfiles.Files
}

func (c lazyDevFileCatalog) ListFiles(ctx context.Context, query lazymedia.LazyDevFileListQuery) ([]lazymedia.LazyDevFileSnapshot, error) {
	stored, _, err := c.files.List(ctx, lazyfiles.ListQuery{
		Storage:   query.Storage,
		KeyPrefix: query.KeyPrefix,
	})
	if err != nil {
		return nil, err
	}
	files := make([]lazymedia.LazyDevFileSnapshot, 0, len(stored))
	for _, file := range stored {
		files = append(files, lazyDevFileSnapshot(file))
	}
	return files, nil
}

func (c lazyDevFileCatalog) PutFile(ctx context.Context, body io.Reader, options lazymedia.LazyDevPutFileOptions) (lazymedia.LazyDevFileSnapshot, error) {
	var fileOptions []any
	if options.Storage != "" {
		fileOptions = append(fileOptions, lazyfiles.StorageName{Name: options.Storage})
	}
	if options.Key != "" {
		fileOptions = append(fileOptions, lazyfiles.ObjectKey{Key: options.Key})
	}
	if options.Filename != "" {
		fileOptions = append(fileOptions, lazyfiles.Filename{Name: options.Filename})
	}
	if options.ContentType != "" {
		fileOptions = append(fileOptions, lazystorage.ContentType{Value: options.ContentType})
	}
	file, _, err := c.files.Put(ctx, body, fileOptions...)
	if err != nil {
		return lazymedia.LazyDevFileSnapshot{}, err
	}
	stored, _, err := c.files.Find(ctx, file.ID)
	if err != nil {
		return lazymedia.LazyDevFileSnapshot{
			ID:          file.ID,
			Filename:    file.Filename,
			ContentType: file.ContentType,
			Size:        file.Size,
			Checksum:    file.Checksum,
			Metadata:    file.Metadata,
			CreatedAt:   file.CreatedAt,
			UpdatedAt:   file.UpdatedAt,
			DeletedAt:   file.DeletedAt,
		}, nil
	}
	return lazyDevFileSnapshot(stored), nil
}

func (c lazyDevFileCatalog) OpenFile(ctx context.Context, id string) (io.ReadCloser, lazymedia.LazyDevFileSnapshot, error) {
	opened, file, _, err := c.files.Open(ctx, id)
	if err != nil {
		return nil, lazymedia.LazyDevFileSnapshot{}, err
	}
	return opened, lazymedia.LazyDevFileSnapshot{
		ID:          file.ID,
		Filename:    file.Filename,
		ContentType: file.ContentType,
		Size:        file.Size,
		Checksum:    file.Checksum,
		Metadata:    file.Metadata,
		CreatedAt:   file.CreatedAt,
		UpdatedAt:   file.UpdatedAt,
		DeletedAt:   file.DeletedAt,
	}, nil
}

func (c lazyDevFileCatalog) FileURL(ctx context.Context, id string) (string, error) {
	url, _, err := c.files.URL(ctx, id)
	return url, err
}

func (c lazyDevFileCatalog) DeleteFile(ctx context.Context, id string) error {
	stored, _, err := c.files.Find(ctx, id)
	if err != nil {
		return err
	}
	for _, location := range stored.Locations {
		storage, ok := c.files.Storages[location.Storage]
		if !ok {
			continue
		}
		deleter, ok := storage.(lazystorage.Deleter)
		if !ok {
			continue
		}
		if _, err := deleter.Delete(ctx, location.Key); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	_, err = c.files.Repository.Delete(ctx, id)
	return err
}

func lazyDevFileSnapshot(stored lazyfiles.StoredFile) lazymedia.LazyDevFileSnapshot {
	locations := make([]lazymedia.LazyDevFileLocation, 0, len(stored.Locations))
	for _, location := range stored.Locations {
		locations = append(locations, lazyDevFileLocation(location))
	}
	return lazymedia.LazyDevFileSnapshot{
		ID:          stored.File.ID,
		Filename:    stored.File.Filename,
		ContentType: stored.File.ContentType,
		Size:        stored.File.Size,
		Checksum:    stored.File.Checksum,
		Metadata:    stored.File.Metadata,
		CreatedAt:   stored.File.CreatedAt,
		UpdatedAt:   stored.File.UpdatedAt,
		DeletedAt:   stored.File.DeletedAt,
		Location:    lazyDevFileLocation(stored.Location),
		Locations:   locations,
	}
}

func lazyDevFileLocation(location lazyfiles.Location) lazymedia.LazyDevFileLocation {
	return lazymedia.LazyDevFileLocation{
		FileID:   location.FileID,
		Storage:  location.Storage,
		Key:      location.Key,
		Role:     location.Role,
		Status:   location.Status,
		Checksum: location.Checksum,
	}
}
