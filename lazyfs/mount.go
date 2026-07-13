package lazyfs

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"
)

// Mount returns a filesystem that exposes files beneath prefix. Parent
// directories needed to reach prefix are synthesized without copying files.
func Mount(prefix string, files fs.FS) (fs.FS, error) {
	if files == nil {
		return nil, fmt.Errorf("lazyfs: mounted filesystem is nil")
	}
	prefix = strings.TrimSpace(prefix)
	if !fs.ValidPath(prefix) {
		return nil, pathError("mount", prefix, fs.ErrInvalid)
	}
	if prefix == "." {
		return files, nil
	}
	return mountedFS{prefix: prefix, files: files}, nil
}

type mountedFS struct {
	prefix string
	files  fs.FS
}

func (mounted mountedFS) Open(name string) (fs.File, error) {
	kind, target, err := mounted.translate("open", name)
	if err != nil {
		return nil, err
	}
	if kind == mountVirtual {
		info := mounted.virtualInfo(name)
		entries, err := mounted.ReadDir(name)
		if err != nil {
			return nil, err
		}
		return &directory{info: info, entries: entries}, nil
	}
	opened, err := mounted.files.Open(target)
	if err != nil {
		return nil, err
	}
	if target != "." {
		return opened, nil
	}
	info, err := opened.Stat()
	if err != nil {
		_ = opened.Close()
		return nil, err
	}
	info = namedInfo{FileInfo: info, name: path.Base(name)}
	if !info.IsDir() {
		return &namedFile{File: opened, info: info}, nil
	}
	_ = opened.Close()
	entries, err := fs.ReadDir(mounted.files, target)
	if err != nil {
		return nil, err
	}
	return &directory{info: info, entries: entries}, nil
}

func (mounted mountedFS) ReadFile(name string) ([]byte, error) {
	kind, target, err := mounted.translate("read", name)
	if err != nil {
		return nil, err
	}
	if kind == mountVirtual {
		return nil, pathError("read", name, fs.ErrInvalid)
	}
	return fs.ReadFile(mounted.files, target)
}

func (mounted mountedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	kind, target, err := mounted.translate("readdir", name)
	if err != nil {
		return nil, err
	}
	if kind == mountFiles {
		return fs.ReadDir(mounted.files, target)
	}
	child := mounted.virtualChild(name)
	return []fs.DirEntry{virtualDirEntry{name: child}}, nil
}

func (mounted mountedFS) Stat(name string) (fs.FileInfo, error) {
	kind, target, err := mounted.translate("stat", name)
	if err != nil {
		return nil, err
	}
	if kind == mountVirtual {
		return mounted.virtualInfo(name), nil
	}
	info, err := fs.Stat(mounted.files, target)
	if err != nil {
		return nil, err
	}
	if target == "." {
		return namedInfo{FileInfo: info, name: path.Base(name)}, nil
	}
	return info, nil
}

func (mounted mountedFS) Glob(pattern string) ([]string, error) {
	return fs.Glob(openOnly{FS: mounted}, pattern)
}

func (mounted mountedFS) ReadLink(name string) (string, error) {
	kind, target, err := mounted.translate("readlink", name)
	if err != nil {
		return "", err
	}
	if kind == mountVirtual {
		return "", pathError("readlink", name, fs.ErrInvalid)
	}
	return fs.ReadLink(mounted.files, target)
}

func (mounted mountedFS) Lstat(name string) (fs.FileInfo, error) {
	kind, target, err := mounted.translate("lstat", name)
	if err != nil {
		return nil, err
	}
	if kind == mountVirtual {
		return mounted.virtualInfo(name), nil
	}
	info, err := fs.Lstat(mounted.files, target)
	if err != nil {
		return nil, err
	}
	if target == "." {
		return namedInfo{FileInfo: info, name: path.Base(name)}, nil
	}
	return info, nil
}

const (
	mountVirtual = iota
	mountFiles
)

func (mounted mountedFS) translate(operation string, name string) (int, string, error) {
	if err := validPath(operation, name); err != nil {
		return 0, "", err
	}
	if name == mounted.prefix {
		return mountFiles, ".", nil
	}
	if strings.HasPrefix(name, mounted.prefix+"/") {
		return mountFiles, strings.TrimPrefix(name, mounted.prefix+"/"), nil
	}
	if name == "." || strings.HasPrefix(mounted.prefix, name+"/") {
		return mountVirtual, "", nil
	}
	return 0, "", pathError(operation, name, fs.ErrNotExist)
}

func (mounted mountedFS) virtualChild(name string) string {
	remainder := mounted.prefix
	if name != "." {
		remainder = strings.TrimPrefix(mounted.prefix, name+"/")
	}
	if separator := strings.IndexByte(remainder, '/'); separator >= 0 {
		return remainder[:separator]
	}
	return remainder
}

func (mounted mountedFS) virtualInfo(name string) fs.FileInfo {
	entryName := path.Base(name)
	if name == "." {
		entryName = "."
	}
	return virtualDirInfo{name: entryName}
}

type namedInfo struct {
	fs.FileInfo
	name string
}

func (info namedInfo) Name() string {
	return info.name
}

type namedFile struct {
	fs.File
	info fs.FileInfo
}

func (file *namedFile) Stat() (fs.FileInfo, error) {
	return file.info, nil
}

type virtualDirInfo struct {
	name string
}

func (info virtualDirInfo) Name() string  { return info.name }
func (virtualDirInfo) Size() int64        { return 0 }
func (virtualDirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (virtualDirInfo) ModTime() time.Time { return time.Time{} }
func (virtualDirInfo) IsDir() bool        { return true }
func (virtualDirInfo) Sys() any           { return nil }

type virtualDirEntry struct {
	name string
}

func (entry virtualDirEntry) Name() string { return entry.name }
func (virtualDirEntry) IsDir() bool        { return true }
func (virtualDirEntry) Type() fs.FileMode  { return fs.ModeDir }
func (entry virtualDirEntry) Info() (fs.FileInfo, error) {
	return virtualDirInfo{name: entry.name}, nil
}

var (
	_ fs.FS         = mountedFS{}
	_ fs.ReadFileFS = mountedFS{}
	_ fs.ReadDirFS  = mountedFS{}
	_ fs.StatFS     = mountedFS{}
	_ fs.GlobFS     = mountedFS{}
	_ fs.ReadLinkFS = mountedFS{}
)
