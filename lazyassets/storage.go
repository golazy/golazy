// Package lazyassets is a package that provides a simple way to manage assets in a web application.
package lazyassets

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/golazy/golazy/router"
)

type Storage struct {
	paths router.Matcher[file]
	Files map[string][]Route
}

type Route struct {
	Path      string
	Permalink string
	Loc       string
}

func (m *Storage) AddFS(fileSys fs.FS, prefix ...string) *Storage {
	var err error
	if len(prefix) > 0 {
		fileSys, err = fs.Sub(fileSys, path.Join(prefix...))
		if err != nil {
			panic(err)
		}
	}
	m.addFS(fileSys, loc())
	return m
}

var linkRegExp = regexp.MustCompile(`url\([\s'"]*([^'"\)]+)[ '"]*\)`)

// formatCSSPermalink replaces the URLs in the CSS with the permalinks
func formatCSSPermalink(m *Server, css string) string {
	result := []byte{}
	last := 0
	for _, submatches := range linkRegExp.FindAllStringSubmatchIndex(css, -1) {
		result = append(result, css[last:submatches[2]]...)

		url := css[submatches[2]:submatches[3]]
		path := url
		if f := m.Find(url); f != nil &&
			(m.PermalinkFiles == nil || m.PermalinkFiles(f)) {
			path = f.Permalink()
		}
		result = append(result, []byte(path)...)
		last = submatches[3]
	}
	result = append(result, css[last:]...)
	return string(result)

}

func (m *Storage) addFS(files fs.FS, loc string) error {
	if m.Files == nil {
		m.Files = map[string][]Route{}
	}
	m.Files[loc] = []Route{}
	err := fs.WalkDir(files, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}
		if d.IsDir() {
			return nil
		}
		f := &file{
			openFn: func() (io.ReadCloser, error) {
				return files.Open(filePath)
			},
			path: filePath,
			mime: mime.TypeByExtension(path.Ext(filePath)),
			loc:  loc,
		}
		m.Files[loc] = append(m.Files[loc], Route{
			Path: filePath,
			Loc:  loc,
		})

		m.addFile("/"+filePath, f)
		return nil
	})
	if err != nil {
		return err
	}
	return nil

}

func (m *Storage) addFile(filepath string, f *file) {
	if m.paths == nil {
		m.paths = router.NewPathMatcher[file]()
	}
	u, err := url.Parse(filepath)
	if err != nil {
		panic(err)
	}

	m.paths.Add(router.NewRouteDefinition(u.String()), f)
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

func (m *Storage) AddFile(path string, content []byte) *Storage {
	if len(path) == 0 {
		panic("path can't be empty")
	}
	if path[0] != '/' {
		path = "/" + path
	}

	f := newStaticFile(path, loc(), content)
	m.addFile(path, f)
	return m
}

//	func (m *Assets) Get(src string) string {
//		p, f := m.Permalink(src)
//		if f == nil {
//			return src
//		}
//		return p
//	}

// func (m *Assets) Integrity(src string) string {
// 	_, f := m.Permalink(src)
// 	if f == nil {
// 		return ""
// 	}
// 	return f.Integrity()
// }

// Find returns the file that matches the path
func (m *Storage) Find(p string) (f File) {

	if len(p) == 0 || m.paths == nil {
		return nil
	}
	if p[0] != '/' {
		p = "/" + p
	}
	req := &http.Request{URL: &url.URL{Path: p}}

	route := m.paths.Find(req)
	if route != nil {
		route.init()
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
	//f = newFile(p, route.Loc, route.f)

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
