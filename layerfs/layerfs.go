package layerfs

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Layer struct {
	fs.FS
	Name string
}

type FS struct {
	Layers []Layer
}

func New() *FS {
	return &FS{}
}

// Add add a layer to the FS
// Later added entries have more priority and can shadow previous entries
func (f *FS) Add(dir fs.FS) {
	var file string
	var line int
	var ok bool
	for i := range 6 {
		_, file, line, ok = runtime.Caller(i + 1)
		if !ok {
			break
		}

		if !strings.Contains(file, "golazy/lazyview") && !strings.Contains(file, "golazy/lazyapp") {
			break
		}
	}
	if file == "" {
		file = "UNKNOWN"
		line = 0
	}

	l := Layer{
		FS:   dir,
		Name: fmt.Sprintf("%s:%d", file, line),
	}

	f.Layers = append([]Layer{l}, f.Layers...)
}

// Open calls Open on the layers in order and return the first one that does not return an error
func (filesystem *FS) Open(name string) (f fs.File, err error) {
	if name == "/" {
		return nil, fs.ErrPermission
	}
	if name == "." || name == "" {
		return &directory{FS: *filesystem, path: name}, nil
	}
	for _, layer := range filesystem.Layers {
		f, err = layer.Open(name)
		if err == nil {
			stat, err := f.Stat()
			if err != nil {
				return f, nil
			}
			if stat.IsDir() {
				return &directory{FS: *filesystem, file: f, path: name}, nil
			}
			return f, nil
		}

	}
	return
}

// Dir returns a new FS that is a subdirectory of the given FS
func Dir(f fs.FS, dir ...string) (fs.FS, error) {
	return fs.Sub(f, filepath.Join(dir...))
}

// MustDir returns a new FS that is a subdirectory of the given FS.
// It panics if the subdirectory can't be found
func MustDir(f fs.FS, dir ...string) fs.FS {
	f, err := Dir(f, dir...)
	if err != nil {
		panic(fmt.Errorf("can't find subdirectory %s: %w", dir, err))
	}
	return f
}

type directory struct {
	entries []fs.DirEntry
	FS      FS
	path    string
	file    fs.File
}

func (d *directory) Stat() (fs.FileInfo, error) {
	if d.file == nil {
		return fakeFileInfo{path: d.path}, nil
	}
	return d.file.Stat()
}

func (d *directory) Read(data []byte) (int, error) {
	if d.file == nil {
		return 0, fs.ErrInvalid
	}
	return d.file.Read(data)
}

func (d *directory) Close() error {
	d.entries = nil
	if d.file == nil {
		return nil
	}
	return d.file.Close()
}

// ReadDir reads the contents of the directory and returns
// a slice of up to n DirEntry values in directory order.
// Subsequent calls on the same file will yield further DirEntry values.
//
// If n > 0, ReadDir returns at most n DirEntry structures.
// In this case, if ReadDir returns an empty slice, it will return
// a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
// (ReadDir must return io.EOF itself, not an error wrapping io.EOF.)
//
// If n <= 0, ReadDir returns all the DirEntry values from the directory
// in a single slice. In this case, if ReadDir succeeds (reads all the way
// to the end of the directory), it returns the slice and a nil error.
// If it encounters an error before the end of the directory,
// ReadDir returns the DirEntry list read until that point and a non-nil error.
func (d *directory) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.entries == nil {

		// Read all the layers and flatten
		results := map[string]fs.DirEntry{}
		for _, layer := range d.FS.Layers {
			entries, err := fs.ReadDir(layer.FS, d.path)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				name := entry.Name()
				if results[name] == nil {
					results[name] = entry
				}
			}
		}
		for _, entry := range results {
			d.entries = append(d.entries, entry)
		}
	}

	if n > 0 && len(d.entries) == 0 {
		return nil, io.EOF
	}

	if len(d.entries) == 0 {
		return nil, nil
	}

	end := len(d.entries)
	if n > 0 && end > n {
		end = n
	}
	entries := make([]fs.DirEntry, end)
	copy(entries, d.entries[:end])

	d.entries = d.entries[end:]

	return entries, nil
}

type fakeFileInfo struct {
	path string
}

func (f fakeFileInfo) Name() string {
	return f.path
}

func (f fakeFileInfo) Size() int64 {
	return 0
}

func (f fakeFileInfo) Mode() fs.FileMode {
	return fs.ModeDir | 0755
}

func (f fakeFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (f fakeFileInfo) IsDir() bool {
	return true
}

func (f fakeFileInfo) Sys() any {
	return nil
}
