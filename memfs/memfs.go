package memfs

import (
	"bytes"
	"io/fs"
	"strings"
	"time"
)

func New() *FS {
	d := newDir()
	return &FS{d}
}

type FS struct {
	*dir
}

func newDir() *dir {
	return &dir{
		files: make([]*file, 0),
		dirs:  make([]*dir, 0),
	}
}

type dir struct {
	name  string
	files []*file
	dirs  []*dir
}

// Implement fs.DirEntry
func (d *dir) Name() string {
	return d.name
}
func (d *dir) IsDir() bool {
	return true
}
func (d *dir) Type() fs.FileMode {
	return fs.ModeDir
}

func (d *dir) Info() (fs.FileInfo, error) {
	return d, nil
}

func (d *dir) Size() int64 {
	return 0
}
func (d *dir) ModTime() time.Time {
	return time.Time{}
}
func (d *dir) Mode() fs.FileMode {
	return fs.ModeDir
}
func (d *dir) Sys() any {
	return nil
}

func (d *dir) ReadDir(components ...string) ([]fs.DirEntry, error) {
	if len(components) == 0 {
		entries := []fs.DirEntry{}
		for _, f := range d.files {
			entries = append(entries, f)
		}
		for _, d := range d.dirs {
			entries = append(entries, d)
		}
		return entries, nil

	}
	for _, d := range d.dirs {
		if d.name == components[0] {
			return d.ReadDir(components[1:]...)
		}
	}
	return nil, fs.ErrNotExist

}

func (d *dir) Open(components ...string) (fs.File, error) {
	if len(components) == 0 {
		return nil, fs.ErrNotExist
	}
	if len(components) == 1 {
		for _, f := range d.files {
			if f.path == components[0] {
				return openfile{
					Reader: bytes.NewReader(f.data),
					file:   f,
				}, nil
			}
		}
		return nil, fs.ErrNotExist
	}
	for _, d := range d.dirs {
		if d.name == components[0] {
			return d.Open(components[1:]...)
		}
	}
	return nil, fs.ErrNotExist

}

func (m FS) Open(name string) (fs.File, error) {
	if name == "" {
		return nil, fs.ErrNotExist
	}
	if name[0] == '/' {
		name = name[1:]
	}
	components := strings.Split(name, "/")
	return m.dir.Open(components...)
}

func (d *dir) add(components []string, data []byte) {
	if len(components) == 0 {
		return
	}
	if len(components) == 1 {
		d.files = append(d.files, &file{
			path:      components[0],
			createdAt: time.Now(),
			data:      data,
		})
		return
	}
	var sub *dir

	for _, sub = range d.dirs {
		if sub.name == components[0] {
			break
		}
	}
	if sub == nil {
		sub = newDir()
		sub.name = components[0]
		d.dirs = append(d.dirs, sub)
	}
	sub.add(components[1:], data)

}

func (m *FS) AddMap(files map[string]string) *FS {
	for file, content := range files {
		m.Add(file, []byte(content))
	}

	return m
}
func (m *FS) Add(name string, data []byte) *FS {
	if name == "" {
		panic("name can't be empty")
	}
	if name[0] == '/' {
		name = name[1:]
	}
	components := strings.Split(name, "/")
	m.dir.add(components, data)
	return m
}

type file struct {
	path      string
	createdAt time.Time
	data      []byte
}

// file implements fs.DirEntry
func (f *file) Name() string {
	return f.path
}
func (f *file) IsDir() bool {
	return false
}
func (f *file) Type() fs.FileMode {
	return fs.ModePerm
}
func (f *file) Info() (fs.FileInfo, error) {
	return f, nil
}

func (f *file) Size() int64 {
	return int64(len(f.data))
}
func (f *file) ModTime() time.Time {
	return f.createdAt
}
func (f *file) Mode() fs.FileMode {
	return fs.ModePerm
}
func (f *file) Sys() any {
	return nil
}

type openfile struct {
	*bytes.Reader
	*file
}

func (f openfile) Stat() (fs.FileInfo, error) {
	return f.file, nil
}

func (f openfile) Close() error {
	return nil
}

func (m FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "/" || name == "." || name == "" {
		return m.dir.ReadDir()
	}
	return m.dir.ReadDir(strings.Split(name, "/")...)
}
