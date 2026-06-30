package lazyfiles

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

const maxLogRecordBytes = 1 << 20

// LogRepository stores file metadata in an append-only JSONL log.
type LogRepository struct {
	mu        sync.RWMutex
	path      string
	files     map[string]File
	locations map[string][]Location
}

type logEvent struct {
	Op       string    `json:"op"`
	File     File      `json:"file,omitempty"`
	Location Location  `json:"location,omitempty"`
	FileID   string    `json:"file_id,omitempty"`
	Time     time.Time `json:"time,omitempty"`
}

// NewLogRepository opens or creates an append-only JSONL repository at path.
func NewLogRepository(path string) (*LogRepository, error) {
	repo := &LogRepository{
		path:      path,
		files:     map[string]File{},
		locations: map[string][]Location{},
	}
	if err := repo.replay(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *LogRepository) Put(ctx context.Context, file File, location Location, options ...any) (File, []any, error) {
	if err := ctx.Err(); err != nil {
		return File{}, options, err
	}
	if location.FileID == "" {
		location.FileID = file.ID
	}
	if location.Role == "" {
		location.Role = RolePrimary
	}
	if location.Status == "" {
		location.Status = StatusActive
	}
	now := time.Now().UTC()
	if file.CreatedAt.IsZero() {
		if existing, ok := r.files[file.ID]; ok && !existing.CreatedAt.IsZero() {
			file.CreatedAt = existing.CreatedAt
		} else {
			file.CreatedAt = now
		}
	}
	file.UpdatedAt = now
	if err := validateFileLocation(file, location); err != nil {
		return File{}, options, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.append(logEvent{Op: "put", File: file, Location: location, Time: now}); err != nil {
		return File{}, options, err
	}
	r.applyPut(file, location)
	return file, options, nil
}

func (r *LogRepository) Find(ctx context.Context, query Query, options ...any) (File, []Location, []any, error) {
	if err := ctx.Err(); err != nil {
		return File{}, nil, options, err
	}
	if query.ID == "" {
		return File{}, nil, options, fmt.Errorf("lazyfiles: file id is required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	file, ok := r.files[query.ID]
	if !ok || !file.DeletedAt.IsZero() {
		return File{}, nil, options, fsErrNotExist(query.ID)
	}
	locations := append([]Location(nil), r.locations[query.ID]...)
	return file, locations, options, nil
}

func (r *LogRepository) Delete(ctx context.Context, fileID string, options ...any) ([]any, error) {
	if err := ctx.Err(); err != nil {
		return options, err
	}
	if fileID == "" {
		return options, fmt.Errorf("lazyfiles: file id is required")
	}
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.append(logEvent{Op: "delete", FileID: fileID, Time: now}); err != nil {
		return options, err
	}
	file := r.files[fileID]
	file.ID = fileID
	file.DeletedAt = now
	file.UpdatedAt = now
	r.files[fileID] = file
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
	scanner.Buffer(make([]byte, 0, 64*1024), maxLogRecordBytes)
	for scanner.Scan() {
		var event logEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return err
		}
		switch event.Op {
		case "put":
			r.applyPut(event.File, event.Location)
		case "delete":
			file := r.files[event.FileID]
			file.ID = event.FileID
			file.DeletedAt = event.Time
			file.UpdatedAt = event.Time
			r.files[event.FileID] = file
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
	if len(data) > maxLogRecordBytes {
		return fmt.Errorf("lazyfiles: log record is %d bytes, exceeds %d byte limit", len(data), maxLogRecordBytes)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return file.Sync()
}

func (r *LogRepository) applyPut(file File, location Location) {
	r.files[file.ID] = file
	locations := r.locations[file.ID]
	for index, existing := range locations {
		if existing.Storage == location.Storage && existing.Key == location.Key {
			locations[index] = location
			r.locations[file.ID] = locations
			return
		}
	}
	r.locations[file.ID] = append(locations, location)
}

func fsErrNotExist(id string) error {
	return fmt.Errorf("lazyfiles: file %q not found: %w", id, os.ErrNotExist)
}
