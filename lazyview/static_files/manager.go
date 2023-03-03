package static_files

import (
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
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

var ErrNotFound = errors.New("not found")

func (m *Manager) initFile(f *File) error {
	f.Lock()
	defer f.Unlock()
	if !f.Hash.Zero() {
		return nil
	}

	file, err := m.fs.Open(path.Join(m.prefix, f.Path))
	if err != nil {
		return err
	}
	f.Init(file)
	file.Close()
	return nil
}

func (m *Manager) Get(path string) string {
	p, err := m.Permalink(path)
	if err != nil {
		panic(err)
	}
	return p
}

func (m *Manager) NewMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f, shouldCache := m.Find(r.URL.Path); f != nil {
			m.ServeFile(f, shouldCache, w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (m *Manager) Permalink(p string) (string, error) {
	for i := 0; i < len(m.files); i++ {
		f := &m.files[i]
		if f.Path == p {
			m.initFile(f)

			return "/" + f.Permalink, nil
		}
	}
	return "", ErrNotFound
}

type Manager struct {
	files  []File
	fs     fs.ReadDirFS
	prefix string
}

func NewManager(fs fs.ReadDirFS, prefix string) *Manager {
	m := &Manager{
		prefix: prefix,
		fs:     fs,
		files:  make([]File, 0, 100),
	}

	m.readFs()

	return m
}

func (m *Manager) readFs(target ...string) {
	base := path.Join(target...)

	abs := path.Join(m.prefix, base)
	entries, err := m.fs.ReadDir(abs)
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			m.readFs(base, e.Name())
			continue
		}

		fullPath := path.Join(base, e.Name())
		m.files = append(m.files, File{Path: fullPath})
	}
}

var (
	errNoHash = errors.New("path without hash")
)

func withoutHash(permalink string) (cleanPath, hash string, err error) {
	fileName := path.Base(permalink)
	i := strings.LastIndex(fileName, "-")
	if i == -1 || i == len(fileName)-1 { // No hash or path ending in dash
		return "", "", errNoHash
	}
	if i == len(fileName)-1 {
		return "", "", errNoHash
	}

	ext := path.Ext(fileName)
	hash = fileName[i+1 : len(fileName)-len(ext)]
	cleanPath = path.Join(path.Dir(permalink), fileName[:i]+ext)

	return
}

func addHash(p, hash string) string {
	ext := path.Ext(p)
	return p[:len(p)-len(ext)] + "-" + hash + ext
}

func (m *Manager) find(p string) (f *File, shouldCache bool) {
	if len(p) == 0 {
		return nil, false
	}
	if p[0] == '/' {
		p = p[1:]
	}
	clean, sha, err := withoutHash(p)
	for i := 0; i < len(m.files); i++ {
		f := &m.files[i]
		// Fetch the normal path
		if f.Path == p {
			return f, false
		}
		if err != nil {
			continue
		}
		if f.Permalink == p {
			return f, true
		}
		// We found a file that matches. Check the sha
		if f.Path == clean {
			_, err := m.Permalink(clean)
			if err != nil {
				panic(err)
			}
			if sha == f.Hash.Short() {
				return f, true
			}
		}
	}
	return nil, false
}

func (m *Manager) Find(p string) (f *File, shouldCache bool) {
	f, shouldCache = m.find(p)
	if f == nil {
		return
	}
	m.initFile(f)
	return
}

func (m *Manager) ServeFile(f *File, shouldCache bool, w http.ResponseWriter, r *http.Request) {
	m.initFile(f)

	if f == nil {
		http.NotFound(w, r)
		return
	}

	noMatch := r.Header.Get("If-None-Match")
	if strings.Contains(noMatch, f.Hash.String()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	file, err := m.fs.Open(path.Join(m.prefix, f.Path))
	if err != nil {
		panic(err)
	}

	if shouldCache {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	}

	w.Header().Set("Content-Type", f.Mime)
	w.Header().Set("ETag", `"`+f.Hash.String()+`"`)
	_, err = io.Copy(w, file)
	if err != nil {
		panic(err)
	}
}

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	f, perm := m.Find(r.URL.Path)
	if f == nil {
		http.NotFound(w, r)
		return
	}
	m.initFile(f)
	m.ServeFile(f, perm, w, r)
}
