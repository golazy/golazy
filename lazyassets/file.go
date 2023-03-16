package lazyassets

import (
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
)

type File struct {
	sync.RWMutex
	F         func() io.Reader
	Permalink bool
	filled    bool

	Mime string
	h    Hash
	Loc  string
}

func newFile(filepath, loc string, c func() io.Reader, cache bool) *File {
	return &File{
		F:         c,
		Mime:      mime.TypeByExtension(path.Ext(filepath)),
		Permalink: cache,
		Loc:       loc,
	}
}

func (f *File) Integrity() string {
	return f.Hash().Integrity()
}

func (f *File) RouteHash() string {
	return f.Hash().Short()
}
func (f *File) init() {
	f.Lock()
	defer f.Unlock()
	if f.filled {
		return
	}

	file := f.F()
	if c, ok := file.(io.Closer); ok {
		defer c.Close()
	}
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	f.h = NewHash(data)
	if f.Mime == "" {
		f.Mime = http.DetectContentType(data)
	}
}

func (f *File) Hash() Hash {
	f.init()
	return f.h
}

func (f *File) Etag() string {
	f.init()
	return f.h.Etag()
}

func (f *File) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.init()

	noMatch := r.Header.Get("If-None-Match")
	if strings.Contains(noMatch, f.h.Etag()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", f.Mime)

	file := f.F()
	if c, ok := file.(io.Closer); ok {
		defer c.Close()
	}

	if f.Permalink {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	}

	w.Header().Set("ETag", `"`+f.h.Etag()+`"`)

	io.Copy(w, file)
}
