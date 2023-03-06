package lazyassets

import (
	"io"
	"mime"
	"net/http"
	"path"
	"sync"
)

type File struct {
	sync.RWMutex
	Path      string
	Permalink string
	Hash      Hash
	Mime      string
}

func (f *File) Init(file io.Reader) {
	content, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	f.Hash = NewHash(content)
	f.Permalink = addHash(f.Path, f.Hash.Short())
	f.Mime = mime.TypeByExtension(path.Ext(f.Path))
	if f.Mime == "" {
		f.Mime = http.DetectContentType(content)

	}
}
