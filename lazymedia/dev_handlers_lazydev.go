//go:build lazydev

package lazymedia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"

	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazystorage"
)

const (
	lazyDevMediaObjectLimit  = 500
	lazyDevMediaUploadMemory = 32 << 20
)

// RegisterLazyDevHandlers registers media, file, and storage inspector
// endpoints for the development panel.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, inspector *LazyDevInspector) {
	if controlPlane == nil {
		return
	}
	if inspector == nil {
		inspector = &LazyDevInspector{}
	}
	controlPlane.Handle("GET "+LazyDevMediaPath, http.HandlerFunc(inspector.handleSnapshot))
	controlPlane.Handle("GET "+LazyDevMediaDownloadPath, http.HandlerFunc(inspector.handleDownload))
	controlPlane.Handle("POST "+LazyDevMediaStorageUploadPath, http.HandlerFunc(inspector.handleStorageUpload))
	controlPlane.Handle("POST "+LazyDevMediaStorageDeletePath, http.HandlerFunc(inspector.handleStorageDelete))
	controlPlane.Handle("POST "+LazyDevMediaFileUploadPath, http.HandlerFunc(inspector.handleFileUpload))
	controlPlane.Handle("POST "+LazyDevMediaFileDeletePath, http.HandlerFunc(inspector.handleFileDelete))
	controlPlane.Handle("POST "+LazyDevMediaVariantDeletePath, http.HandlerFunc(inspector.handleVariantDelete))
}

func (i *LazyDevInspector) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	writeLazyDevMediaJSON(w, i.snapshot(r))
}

func (i *LazyDevInspector) snapshot(r *http.Request) LazyDevSnapshot {
	storages := i.storageSnapshots()
	selectedStorage := strings.TrimSpace(r.URL.Query().Get("storage"))
	if selectedStorage == "" {
		selectedStorage = i.defaultStorageName(storages)
	}
	prefix := strings.Trim(strings.TrimSpace(r.URL.Query().Get("prefix")), "/")

	snapshot := LazyDevSnapshot{
		Storages:        storages,
		SelectedStorage: selectedStorage,
		StoragePrefix:   prefix,
	}
	if selectedStorage != "" {
		objects, truncated, err := i.listStorageObjects(r.Context(), selectedStorage, prefix)
		snapshot.StorageObjects = objects
		snapshot.StorageObjectsTruncated = truncated
		if err != nil {
			snapshot.StorageObjectsError = err.Error()
		}
	}

	files, err := i.listFiles(r.Context())
	if err != nil {
		snapshot.FilesError = err.Error()
	}
	variants, err := i.listVariants(r.Context())
	if err != nil {
		snapshot.VariantsError = err.Error()
	}
	snapshot.Variants = variants
	snapshot.Files = attachLazyDevVariants(files, variants)
	return snapshot
}

func (i *LazyDevInspector) storageSnapshots() []LazyDevStorageSnapshot {
	names := make([]string, 0, len(i.Storages))
	for name := range i.Storages {
		names = append(names, name)
	}
	sort.Strings(names)
	storages := make([]LazyDevStorageSnapshot, 0, len(names))
	for _, name := range names {
		storage := i.Storages[name]
		_, writable := storage.(lazystorage.Writer)
		_, deletable := storage.(lazystorage.Deleter)
		_, listable := storage.(lazystorage.Lister)
		_, urlable := storage.(lazystorage.URLer)
		storages = append(storages, LazyDevStorageSnapshot{
			Name:      name,
			Default:   name == i.DefaultStorage,
			Readable:  storage != nil,
			Writable:  storage != nil && writable,
			Deletable: storage != nil && deletable,
			Listable:  storage != nil && listable,
			URLable:   storage != nil && urlable,
		})
	}
	return storages
}

func (i *LazyDevInspector) defaultStorageName(storages []LazyDevStorageSnapshot) string {
	if strings.TrimSpace(i.DefaultStorage) != "" {
		if _, ok := i.Storages[i.DefaultStorage]; ok {
			return i.DefaultStorage
		}
	}
	if len(storages) == 0 {
		return ""
	}
	return storages[0].Name
}

func (i *LazyDevInspector) listStorageObjects(ctx context.Context, storageName string, prefix string) ([]LazyDevStorageObject, bool, error) {
	storage, ok := i.Storages[storageName]
	if !ok || storage == nil {
		return nil, false, fmt.Errorf("storage %q is not configured", storageName)
	}
	lister, ok := storage.(lazystorage.Lister)
	if !ok {
		return nil, false, fmt.Errorf("storage %q cannot list objects", storageName)
	}
	if prefix != "" {
		if err := lazystorage.ValidateKey(prefix); err != nil {
			return nil, false, err
		}
	}
	iterator, _, err := lister.List(ctx, prefix)
	if err != nil {
		return nil, false, err
	}
	defer iterator.Close()

	objects := []LazyDevStorageObject{}
	for {
		info, err := iterator.Next()
		if errors.Is(err, io.EOF) {
			return objects, false, nil
		}
		if err != nil {
			return objects, false, err
		}
		object := LazyDevStorageObject{
			Key:         info.Key,
			ContentType: info.ContentType,
			Size:        info.Size,
			Checksum:    info.Checksum,
			ModifiedAt:  info.ModifiedAt,
		}
		if urler, ok := storage.(lazystorage.URLer); ok {
			resolved, _, err := urler.URL(ctx, info.Key)
			if err == nil {
				object.URL = resolved.String
			} else {
				object.URLError = err.Error()
			}
		}
		objects = append(objects, object)
		if len(objects) >= lazyDevMediaObjectLimit {
			return objects, true, nil
		}
	}
}

func (i *LazyDevInspector) listFiles(ctx context.Context) ([]LazyDevFileSnapshot, error) {
	if i.Files == nil {
		return nil, nil
	}
	files, err := i.Files.ListFiles(ctx, LazyDevFileListQuery{})
	if err != nil {
		return nil, err
	}
	for index := range files {
		url, err := i.Files.FileURL(ctx, files[index].ID)
		if err == nil {
			files[index].URL = url
		} else {
			files[index].URLError = err.Error()
		}
	}
	sort.Slice(files, func(left, right int) bool {
		leftTime := files[left].UpdatedAt
		rightTime := files[right].UpdatedAt
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return files[left].ID < files[right].ID
	})
	return files, nil
}

func (i *LazyDevInspector) listVariants(ctx context.Context) ([]LazyDevVariantSnapshot, error) {
	if i.Media == nil {
		return nil, nil
	}
	variants, _, err := i.Media.ListVariants(ctx, VariantListQuery{})
	if err != nil {
		return nil, err
	}
	rows := make([]LazyDevVariantSnapshot, 0, len(variants))
	for _, variant := range variants {
		row := LazyDevVariantSnapshot{
			SourceFileID: variant.SourceFileID,
			VariantKey:   variant.VariantKey,
			Spec:         variant.Spec,
			OutputFileID: variant.OutputFileID,
			Status:       variant.Status,
			Error:        variant.Error,
			CreatedAt:    variant.CreatedAt,
			UpdatedAt:    variant.UpdatedAt,
		}
		if i.Files != nil && row.OutputFileID != "" {
			url, err := i.Files.FileURL(ctx, row.OutputFileID)
			if err == nil {
				row.OutputURL = url
			} else {
				row.OutputURLError = err.Error()
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(left, right int) bool {
		if rows[left].SourceFileID != rows[right].SourceFileID {
			return rows[left].SourceFileID < rows[right].SourceFileID
		}
		return rows[left].VariantKey < rows[right].VariantKey
	})
	return rows, nil
}

func attachLazyDevVariants(files []LazyDevFileSnapshot, variants []LazyDevVariantSnapshot) []LazyDevFileSnapshot {
	if len(files) == 0 || len(variants) == 0 {
		return files
	}
	bySource := map[string][]LazyDevVariantSnapshot{}
	for _, variant := range variants {
		bySource[variant.SourceFileID] = append(bySource[variant.SourceFileID], variant)
	}
	for index := range files {
		files[index].Variants = append([]LazyDevVariantSnapshot(nil), bySource[files[index].ID]...)
	}
	return files
}

func (i *LazyDevInspector) handleDownload(w http.ResponseWriter, r *http.Request) {
	if fileID := strings.TrimSpace(r.URL.Query().Get("file")); fileID != "" {
		i.downloadFile(w, r, fileID)
		return
	}
	i.downloadStorageObject(w, r)
}

func (i *LazyDevInspector) downloadFile(w http.ResponseWriter, r *http.Request, fileID string) {
	if i.Files == nil {
		http.Error(w, "file catalog is not configured", http.StatusNotFound)
		return
	}
	opened, file, err := i.Files.OpenFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, err.Error(), statusForLazyDevMediaError(err))
		return
	}
	defer opened.Close()
	filename := file.Filename
	if filename == "" {
		filename = file.ID
	}
	serveLazyDevMediaDownload(w, r, opened, filename, file.ContentType, file.Size)
}

func (i *LazyDevInspector) downloadStorageObject(w http.ResponseWriter, r *http.Request) {
	storageName := strings.TrimSpace(r.URL.Query().Get("storage"))
	key := strings.Trim(strings.TrimSpace(r.URL.Query().Get("key")), "/")
	if storageName == "" || key == "" {
		http.Error(w, "storage and key are required", http.StatusBadRequest)
		return
	}
	storage, ok := i.Storages[storageName]
	if !ok || storage == nil {
		http.Error(w, fmt.Sprintf("storage %q is not configured", storageName), http.StatusNotFound)
		return
	}
	opened, _, err := storage.Open(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), statusForLazyDevMediaError(err))
		return
	}
	defer opened.Close()
	info, _ := opened.Stat()
	serveLazyDevMediaDownload(w, r, opened, path.Base(key), info.ContentType, info.Size)
}

func serveLazyDevMediaDownload(w http.ResponseWriter, r *http.Request, body io.Reader, filename string, contentType string, size int64) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, body)
}

func (i *LazyDevInspector) handleStorageUpload(w http.ResponseWriter, r *http.Request) {
	storageName := strings.TrimSpace(r.FormValue("storage"))
	storage, ok := i.Storages[storageName]
	if !ok || storage == nil {
		http.Error(w, fmt.Sprintf("storage %q is not configured", storageName), http.StatusNotFound)
		return
	}
	writer, ok := storage.(lazystorage.Writer)
	if !ok {
		http.Error(w, fmt.Sprintf("storage %q cannot write", storageName), http.StatusBadRequest)
		return
	}
	file, header, err := lazyDevMediaUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	key := strings.Trim(strings.TrimSpace(r.FormValue("key")), "/")
	if key == "" {
		key = strings.Trim(strings.TrimSpace(header.Filename), "/")
	}
	if key == "" {
		http.Error(w, "storage key is required", http.StatusBadRequest)
		return
	}
	_, _, err = writer.Put(r.Context(), key, file, lazystorage.ContentType{Value: uploadContentType(header)})
	if err != nil {
		http.Error(w, err.Error(), statusForLazyDevMediaError(err))
		return
	}
	writeLazyDevMediaJSON(w, map[string]string{"status": "ok"})
}

func (i *LazyDevInspector) handleStorageDelete(w http.ResponseWriter, r *http.Request) {
	storageName := strings.TrimSpace(r.FormValue("storage"))
	key := strings.Trim(strings.TrimSpace(r.FormValue("key")), "/")
	if storageName == "" || key == "" {
		http.Error(w, "storage and key are required", http.StatusBadRequest)
		return
	}
	storage, ok := i.Storages[storageName]
	if !ok || storage == nil {
		http.Error(w, fmt.Sprintf("storage %q is not configured", storageName), http.StatusNotFound)
		return
	}
	deleter, ok := storage.(lazystorage.Deleter)
	if !ok {
		http.Error(w, fmt.Sprintf("storage %q cannot delete", storageName), http.StatusBadRequest)
		return
	}
	if _, err := deleter.Delete(r.Context(), key); err != nil {
		http.Error(w, err.Error(), statusForLazyDevMediaError(err))
		return
	}
	writeLazyDevMediaJSON(w, map[string]string{"status": "ok"})
}

func (i *LazyDevInspector) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if i.Files == nil {
		http.Error(w, "file catalog is not configured", http.StatusNotFound)
		return
	}
	file, header, err := lazyDevMediaUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	options := LazyDevPutFileOptions{
		Storage:     strings.TrimSpace(r.FormValue("storage")),
		Key:         strings.Trim(strings.TrimSpace(r.FormValue("key")), "/"),
		Filename:    strings.TrimSpace(r.FormValue("filename")),
		ContentType: uploadContentType(header),
	}
	if options.Filename == "" {
		options.Filename = header.Filename
	}
	saved, err := i.Files.PutFile(r.Context(), file, options)
	if err != nil {
		http.Error(w, err.Error(), statusForLazyDevMediaError(err))
		return
	}
	writeLazyDevMediaJSON(w, saved)
}

func (i *LazyDevInspector) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if i.Files == nil {
		http.Error(w, "file catalog is not configured", http.StatusNotFound)
		return
	}
	fileID := strings.TrimSpace(r.FormValue("file"))
	if fileID == "" {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	if err := i.Files.DeleteFile(r.Context(), fileID); err != nil {
		http.Error(w, err.Error(), statusForLazyDevMediaError(err))
		return
	}
	writeLazyDevMediaJSON(w, map[string]string{"status": "ok"})
}

func (i *LazyDevInspector) handleVariantDelete(w http.ResponseWriter, r *http.Request) {
	if i.Media == nil || i.Media.Repository == nil {
		http.Error(w, "media repository is not configured", http.StatusNotFound)
		return
	}
	sourceFileID := strings.TrimSpace(r.FormValue("source_file"))
	variantKey := strings.TrimSpace(r.FormValue("variant"))
	if sourceFileID == "" || variantKey == "" {
		http.Error(w, "source_file and variant are required", http.StatusBadRequest)
		return
	}
	if _, err := i.Media.Repository.DeleteVariant(r.Context(), sourceFileID, variantKey); err != nil {
		http.Error(w, err.Error(), statusForLazyDevMediaError(err))
		return
	}
	writeLazyDevMediaJSON(w, map[string]string{"status": "ok"})
}

func lazyDevMediaUpload(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	if err := r.ParseMultipartForm(lazyDevMediaUploadMemory); err != nil {
		return nil, nil, err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, nil, err
	}
	return file, header, nil
}

func uploadContentType(header *multipart.FileHeader) string {
	if header == nil {
		return ""
	}
	return strings.TrimSpace(header.Header.Get("Content-Type"))
}

func statusForLazyDevMediaError(err error) int {
	if errors.Is(err, os.ErrNotExist) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

func writeLazyDevMediaJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, fmt.Sprintf("media: %v", err), http.StatusInternalServerError)
	}
}
