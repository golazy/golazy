package lazyapp

import (
	"errors"
	"io"
	"io/fs"
	"sort"
)

func overlayViewFS(primary fs.FS, fallback fs.FS) fs.FS {
	if primary == nil {
		return fallback
	}
	if fallback == nil {
		return primary
	}
	return overlayFS{primary: primary, fallback: fallback}
}

type overlayFS struct {
	primary  fs.FS
	fallback fs.FS
}

func (o overlayFS) Open(name string) (fs.File, error) {
	primaryFile, primaryErr := o.primary.Open(name)
	fallbackFile, fallbackErr := o.fallback.Open(name)

	if primaryErr == nil {
		if fallbackErr == nil {
			defer fallbackFile.Close()
		}
		info, err := primaryFile.Stat()
		if err != nil {
			return primaryFile, nil
		}
		if !info.IsDir() {
			return primaryFile, nil
		}
		_ = primaryFile.Close()
		entries, err := overlayReadDir(o.primary, o.fallback, name)
		if err != nil {
			return nil, err
		}
		return &overlayDir{info: info, entries: entries}, nil
	}

	if fallbackErr == nil {
		return fallbackFile, nil
	}
	if !errors.Is(primaryErr, fs.ErrNotExist) {
		return nil, primaryErr
	}
	return nil, fallbackErr
}

func overlayReadDir(primary fs.FS, fallback fs.FS, name string) ([]fs.DirEntry, error) {
	entriesByName := map[string]fs.DirEntry{}
	if err := addDirEntries(entriesByName, fallback, name, false); err != nil {
		return nil, err
	}
	if err := addDirEntries(entriesByName, primary, name, true); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entriesByName))
	for name := range entriesByName {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, entriesByName[name])
	}
	return entries, nil
}

func addDirEntries(target map[string]fs.DirEntry, source fs.FS, name string, override bool) error {
	entries, err := fs.ReadDir(source, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, exists := target[entry.Name()]; !exists || override {
			target[entry.Name()] = entry
		}
	}
	return nil
}

type overlayDir struct {
	info    fs.FileInfo
	entries []fs.DirEntry
	offset  int
}

func (d *overlayDir) Stat() (fs.FileInfo, error) {
	return d.info, nil
}

func (d *overlayDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.info.Name(), Err: fs.ErrInvalid}
}

func (d *overlayDir) Close() error {
	return nil
}

func (d *overlayDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.offset >= len(d.entries) && n > 0 {
		return nil, io.EOF
	}
	if n <= 0 {
		remaining := d.entries[d.offset:]
		d.offset = len(d.entries)
		return remaining, nil
	}

	end := d.offset + n
	if end > len(d.entries) {
		end = len(d.entries)
	}
	entries := d.entries[d.offset:end]
	d.offset = end
	if len(entries) == 0 {
		return nil, io.EOF
	}
	return entries, nil
}
