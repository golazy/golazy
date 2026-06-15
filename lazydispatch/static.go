package lazydispatch

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

type staticFiles struct {
	files  fs.FS
	server http.Handler
}

func Public(files fs.FS) Middleware {
	return Static(files)
}

func Static(files fs.FS) Middleware {
	return staticFiles{
		files:  files,
		server: http.FileServerFS(files),
	}
}

func (s staticFiles) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.exists(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			MethodNotAllowed(http.MethodGet).ServeHTTP(w, r)
			return
		}
		s.server.ServeHTTP(w, r)
	})
}

func (s staticFiles) exists(requestPath string) bool {
	name := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
	if name == "." {
		name = ""
	}
	if name == "" {
		return fs.ValidPath("index.html") && fileExists(s.files, "index.html")
	}
	if !fs.ValidPath(name) {
		return false
	}
	info, err := fs.Stat(s.files, name)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return true
	}
	return fileExists(s.files, path.Join(name, "index.html"))
}

func fileExists(files fs.FS, name string) bool {
	info, err := fs.Stat(files, name)
	return err == nil && !info.IsDir()
}
