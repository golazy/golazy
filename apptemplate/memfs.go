package apptemplate

import (
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"
)

type MemFS map[string]string

func (m MemFS) Open(n string) (fs.File, error) {
	if !fs.ValidPath(n) {
		return nil, &fs.PathError{Op: "open", Path: n, Err: fs.ErrInvalid}
	}
	if n == "." {
		return &memFile{isDir: true, path: "", fs: m}, nil
	}
	_, ok := m[n]
	if ok {
		return &memFile{path: n, fs: m}, nil
	}
	// Try with dirs
	for k := range m {
		current := getdir(k)
		if strings.HasPrefix(current+"/", n+"/") {
			return &memFile{path: n, isDir: true, fs: m}, nil
		}
	}

	return nil, fs.ErrNotExist
}

func getdir(path string) string {
	n := strings.LastIndexAny(path, "/")
	if n == -1 {
		return ""
	}
	return path[:n]
}
func getFirstDir(path string) string {
	n := strings.IndexAny(path, "/")
	if n == -1 {
		return ""
	}
	return path[:n]
}

var _ fs.DirEntry = &memFile{}
var _ fs.ReadDirFile = &memFile{}
var _ fs.FileInfo = &memFile{}
var _ fs.File = &memFile{}

type memFile struct {
	pos   uint32
	path  string
	isDir bool
	fs    MemFS
}

func (m *memFile) Read(p []byte) (n int, err error) {
	data := []byte(m.fs[m.path])
	//given the amount of data read already in m.pos, the size of data, copy up to the len of p or the len of data. save the pos.
	n = copy(p, data[m.pos:])
	m.pos += uint32(n)
	if m.pos >= uint32(len(data)) {
		err = io.EOF
	}
	return
}

func (m *memFile) Close() error { return nil }

func (m *memFile) Stat() (fs.FileInfo, error) {
	return m, nil
}

func (m *memFile) IsDir() bool {
	return m.isDir
}

func (m *memFile) Name() string {
	return path.Base(m.path)
}
func (m *memFile) ModTime() time.Time {
	return fTime
}

var fTime = time.Now()

func (m *memFile) Mode() fs.FileMode {
	if m.isDir {
		return fs.ModeDir
	}
	return 0
}
func (m *memFile) Size() int64 {
	s := int64(len(m.fs[m.path]))
	return s
}
func (m *memFile) Sys() interface{} {
	return nil
}

func (m *memFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if n == 0 {
		return nil, nil
	}
	// find all the paths with the name prefix of m.path
	entries := map[string]*memFile{}
	for k := range m.fs {
		if m.path != "" && !strings.HasPrefix(k, m.path+"/") {
			continue
		}

		if m.path != "" {
			k = strings.TrimPrefix(k, m.path+"/")
		}
		dir, file := path.Split(k)
		if dir == "" {
			entries[file] = &memFile{path: path.Join(m.path, k), isDir: false, fs: m.fs}
			continue
		}
		dir = getFirstDir(dir)
		if _, ok := entries[dir]; !ok {
			entries[dir] = &memFile{isDir: true, path: dir, fs: m.fs}
		}
	}

	dirEntries := make([]fs.DirEntry, 0, len(entries))
	for _, v := range entries {
		dirEntries = append(dirEntries, v)
	}

	// Ensure it is sorted for susequent calls
	slices.SortFunc(dirEntries, func(i, j fs.DirEntry) int {
		return strings.Compare(i.(*memFile).path, j.(*memFile).path)
	})

	start := m.pos
	end := uint32(len(dirEntries))
	if end-start == 0 {
		if n < 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n > 0 {
		if end-start > uint32(n) {
			end = start + uint32(n)
			m.pos += uint32(n)
			return dirEntries[start:end], nil
		}
		m.pos += end
		return dirEntries[start:end], io.EOF
	}
	m.pos += end
	return dirEntries[start:end], nil

}

var maxInt = int(^uint(0) >> 1)

func (m *memFile) Info() (fs.FileInfo, error) {
	return m, nil
}

func (m *memFile) Type() fs.FileMode {
	return m.Mode()
}
