package lazyassets

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
)

type File interface {
	Path() string
	Permalink() string

	Hash() string
	Integrity() string

	MimeType() string
}

type fileWithPrefix struct {
	File
	pathPrefix string
}

func (f *fileWithPrefix) Path() string {
	return path.Join(f.pathPrefix, f.File.Path())
}

func (f *fileWithPrefix) Permalink() string {
	return path.Join(f.pathPrefix, f.File.Permalink())
}

type file struct {
	sync.RWMutex

	path string

	openFn func() (io.ReadCloser, error)

	permalink string
	mime      string
	loc       string

	pathPrefix string

	h hash
}

func (f *file) Path() string {
	if f.pathPrefix != "" {
		return path.Join(f.pathPrefix, f.path)
	}

	return "/" + f.path
}

func (f *file) Permalink() string {
	f.init()
	if f.pathPrefix != "" {
		return path.Join(f.pathPrefix, f.permalink)
	}
	return "/" + f.permalink
}

func (f *file) Hash() string {
	return f.hash().Short()
}

func (f *file) MimeType() string {
	return f.mime
}

func newStaticFile(fpath string, loc string, content []byte) *file {
	return &file{
		openFn: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(content)), nil
		},
		path: strings.TrimPrefix(fpath, "/"),
		mime: mime.TypeByExtension(path.Ext(fpath)),
		loc:  loc,
	}
}

func (f *file) Integrity() string {
	return f.hash().Integrity()
}

func (f *file) RouteHash() string {
	return f.hash().Short()
}
func (f *file) init() error {
	f.Lock()
	defer f.Unlock()
	if f.permalink != "" {
		return nil
	}

	file, err := f.openFn()
	if err != nil {
		return err
	}
	if c, ok := file.(io.Closer); ok {
		defer c.Close()
	}
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	f.h = newHash(data)
	if f.mime == "" {
		f.mime = http.DetectContentType(data)
	}
	f.permalink = withHash(f.path, f.h.Short())
	return nil
}

func (f *file) hash() hash {
	f.init()
	return f.h
}

func (f *file) Etag() string {
	f.init()
	return f.h.Etag()
}
