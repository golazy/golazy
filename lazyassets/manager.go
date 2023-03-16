package lazyassets

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golazy.dev/lazyaction/router"
)

type Assets struct {
	paths router.Matcher[File]

	files  []File
	Prefix string
}

func New() *Assets {
	m := &Assets{
		paths: router.NewPathMatcher[File](),
		files: make([]File, 0, 100),
	}

	return m
}

type Route struct {
	Path string
	Loc  string
}

func (m *Assets) Routes() []Route {
	path := m.paths.All()
	routes := make([]Route, len(path))
	for i, route := range path {
		routes[i] = Route{
			route.Req.URL.String(),
			route.T.Loc,
		}
	}
	return routes
}
func (m *Assets) AddFS(fs fs.ReadDirFS, prefix string) *Assets {
	m.addFS(fs, loc(), prefix)
	return m
}

type Stylesheet struct {
	content [][]byte
}

func (s *Stylesheet) Add(content []byte) {
	// TODO: Add log info about the caller
	s.content = append(s.content, content)
}

func (s *Stylesheet) newReader() io.Reader {
	return bytes.NewReader(bytes.Join(s.content, nil))
}

func (m *Assets) NewStylesheet(path string) *Stylesheet {
	if len(path) == 0 {
		panic("path can't be empty")
	}
	if path[0] != '/' {
		path = "/" + path
	}

	s := &Stylesheet{}
	m.addFile(path, &File{
		F:    s.newReader,
		Mime: "text/css",
		Loc:  loc(),
	})
	return s
}

func (m *Assets) addFS(fs fs.ReadDirFS, loc, prefix string, fpath ...string) {
	filePath := path.Join(fpath...)
	fullPath := path.Join(prefix, filePath)
	dir, err := fs.ReadDir(fullPath)
	if err != nil {
		panic(err)
	}

	for _, entry := range dir {
		p := path.Join(filePath, entry.Name())
		if entry.IsDir() {
			m.addFS(fs, loc, prefix, p)
			continue
		}

		f := newFile(p, loc, func() io.Reader {
			file, err := fs.Open(path.Join(prefix, p))
			if err != nil {
				panic(err)
			}
			return file
		}, false)

		m.addFile("/"+p, f)

	}
}

func (m *Assets) addFile(filepath string, f *File) {
	m.paths.Add(router.NewRouteDefinition(filepath), f)
}

func loc() string {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		wd, _ := os.Getwd()
		f, err := filepath.Rel(wd, file)
		if err != nil {
			return file + ":" + strconv.Itoa(line)
		}
		return f + ":" + strconv.Itoa(line)
	}
	return ""

}

func (m *Assets) AddFile(path string, content []byte) *Assets {
	if len(path) == 0 {
		panic("path can't be empty")
	}
	if path[0] != '/' {
		path = "/" + path
	}

	f := newFile(path, loc(), func() io.Reader {
		return bytes.NewReader(content)
	}, false)
	m.addFile(path, f)
	return m
}

func (m *Assets) NewMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := m.paths.Find(r)
		if f == nil {
			h.ServeHTTP(w, r)
			return
		}
		f.ServeHTTP(w, r)
	})
}

func (m *Assets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file := m.Find(r.URL.Path)
	if file == nil {
		http.NotFound(w, r)
		return
	}
	file.ServeHTTP(w, r)
}

func (m *Assets) Get(src string) string {
	p, f := m.Permalink(src)
	if f == nil {
		panic("File not found: " + src)
	}
	return p

}

func (m *Assets) Permalink(p string) (string, *File) {
	if len(p) == 0 {
		return "", nil
	}
	if p[0] != '/' {
		p = "/" + p
	}

	f := m.Find(p)
	if f == nil {
		return "", nil
	}
	if f.Permalink {
		return p, f
	}

	path := withHash(p, f.RouteHash())
	f = m.Find(path)

	return path, f
}

func (m *Assets) PermalinkFile(p string) *File {
	f := m.Find(p)
	if f == nil {
		return nil
	}
	if f.Permalink {
		return f
	}

	f = m.Find(withHash(p, f.RouteHash()))
	return f

}

func (m *Assets) Find(p string) (f *File) {
	if len(p) == 0 || m.paths == nil {
		return nil
	}
	if p[0] != '/' {
		p = "/" + p
	}
	req := &http.Request{URL: &url.URL{Path: p}}

	route := m.paths.Find(req)
	if route != nil {
		return route
	}

	clean, sha, err := withoutHash(p)
	if err != nil {
		return nil
	}
	route = m.paths.Find(&http.Request{URL: &url.URL{Path: clean}})
	if route == nil {
		return nil
	}
	if sha != route.RouteHash() {
		return nil
	}
	// Add the route
	f = newFile(p, route.Loc, route.F, true)
	m.addFile(p, f)

	return f

}

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

func withHash(p, hash string) string {
	ext := path.Ext(p)
	return p[:len(p)-len(ext)] + "-" + hash + ext
}
