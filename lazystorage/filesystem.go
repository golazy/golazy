package lazystorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Filesystem stores objects under a local root directory.
type Filesystem struct {
	root    string
	baseURL string
}

// FilesystemOption configures a filesystem storage.
type FilesystemOption func(*Filesystem)

// WithBaseURL lets Filesystem satisfy URLer. Leave it empty when lazyfiles
// should serve fallback application URLs instead.
func WithBaseURL(baseURL string) FilesystemOption {
	return func(storage *Filesystem) {
		storage.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// NewFilesystem creates a filesystem storage rooted at root.
func NewFilesystem(root string, options ...FilesystemOption) *Filesystem {
	storage := &Filesystem{root: root}
	for _, option := range options {
		option(storage)
	}
	return storage
}

// Open opens key for reading.
func (s *Filesystem) Open(ctx context.Context, key string, options ...any) (File, []any, error) {
	if err := ctx.Err(); err != nil {
		return nil, options, err
	}
	if err := ValidateKey(key); err != nil {
		return nil, options, err
	}
	file, err := os.Open(s.localPath(key))
	if err != nil {
		return nil, options, err
	}
	return &filesystemFile{File: file, key: key}, options, nil
}

// Put writes key atomically where the host filesystem supports rename.
func (s *Filesystem) Put(ctx context.Context, key string, body io.Reader, options ...any) (Info, []any, error) {
	if err := ctx.Err(); err != nil {
		return Info{}, options, err
	}
	if err := ValidateKey(key); err != nil {
		return Info{}, options, err
	}
	if body == nil {
		return Info{}, options, fmt.Errorf("lazystorage: nil body")
	}
	contentType, remaining, _ := Take[ContentType](options)

	target := s.localPath(key)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Info{}, remaining, err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".lazystorage-*")
	if err != nil {
		return Info{}, remaining, err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	hash := sha256.New()
	var sniff bytes.Buffer
	writer := io.MultiWriter(temp, hash)
	size, err := copyWithSniff(writer, body, &sniff)
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Info{}, remaining, err
	}
	if err := os.Rename(tempName, target); err != nil {
		return Info{}, remaining, err
	}

	info, err := os.Stat(target)
	if err != nil {
		return Info{}, remaining, err
	}
	detected := strings.TrimSpace(contentType.Value)
	if detected == "" {
		detected = contentTypeForKey(key, sniff.Bytes())
	}
	return Info{
		Key:         key,
		ContentType: detected,
		Size:        size,
		Checksum:    "sha256:" + hex.EncodeToString(hash.Sum(nil)),
		ModifiedAt:  info.ModTime(),
	}, remaining, nil
}

// Delete removes key.
func (s *Filesystem) Delete(ctx context.Context, key string, options ...any) ([]any, error) {
	if err := ctx.Err(); err != nil {
		return options, err
	}
	if err := ValidateKey(key); err != nil {
		return options, err
	}
	return options, os.Remove(s.localPath(key))
}

// List lists object metadata below prefix.
func (s *Filesystem) List(ctx context.Context, prefix string, options ...any) (Iterator, []any, error) {
	if err := ctx.Err(); err != nil {
		return nil, options, err
	}
	if prefix != "" {
		if err := ValidateKey(prefix); err != nil {
			return nil, options, err
		}
	}
	var infos []Info
	root := s.root
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		key, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		key = filepath.ToSlash(key)
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			return nil
		}
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		infos = append(infos, Info{
			Key:         key,
			ContentType: contentTypeForKey(key, nil),
			Size:        stat.Size(),
			ModifiedAt:  stat.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, options, err
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Key < infos[j].Key
	})
	return &sliceIterator{infos: infos}, options, nil
}

// URL returns a public URL only when Filesystem was configured with a base URL.
func (s *Filesystem) URL(ctx context.Context, key string, options ...any) (URL, []any, error) {
	if err := ctx.Err(); err != nil {
		return URL{}, options, err
	}
	if err := ValidateKey(key); err != nil {
		return URL{}, options, err
	}
	if s.baseURL == "" {
		return URL{}, options, fmt.Errorf("lazystorage: filesystem storage has no base URL")
	}
	return URL{String: s.baseURL + "/" + path.Clean(key), Public: true}, options, nil
}

func (s *Filesystem) localPath(key string) string {
	return filepath.Join(s.root, filepath.FromSlash(key))
}

type filesystemFile struct {
	*os.File
	key string
}

func (f *filesystemFile) Stat() (Info, error) {
	info, err := f.File.Stat()
	if err != nil {
		return Info{}, err
	}
	return Info{
		Key:         f.key,
		ContentType: contentTypeForKey(f.key, nil),
		Size:        info.Size(),
		ModifiedAt:  info.ModTime(),
	}, nil
}

type sliceIterator struct {
	infos []Info
	index int
}

func (i *sliceIterator) Next() (Info, error) {
	if i.index >= len(i.infos) {
		return Info{}, io.EOF
	}
	info := i.infos[i.index]
	i.index++
	return info, nil
}

func (i *sliceIterator) Close() error {
	return nil
}

func copyWithSniff(dst io.Writer, src io.Reader, sniff *bytes.Buffer) (int64, error) {
	var size int64
	buffer := make([]byte, 32*1024)
	for {
		n, readErr := src.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if sniff.Len() < 512 {
				remaining := min(512-sniff.Len(), len(chunk))
				_, _ = sniff.Write(chunk[:remaining])
			}
			written, writeErr := dst.Write(chunk)
			size += int64(written)
			if writeErr != nil {
				return size, writeErr
			}
			if written != n {
				return size, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return size, nil
		}
		if readErr != nil {
			return size, readErr
		}
	}
}

func contentTypeForKey(key string, data []byte) string {
	if contentType := mime.TypeByExtension(path.Ext(key)); contentType != "" {
		return contentType
	}
	if len(data) > 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
}
