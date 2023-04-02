package generator

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golazy.dev/lazysupport"
)

const tmplExt = ".tmpl"

type FSGenerator struct {
	Base
	WithVars
	FS       embed.FS
	FuncMap  template.FuncMap
	Logger   Logger
	Resolver Resolver

	dest       string
	data       any
	TrimPrefix string
}

func (t *FSGenerator) Generate(path string, data map[string]string) error {
	t.dest = path
	t.data = data
	if err := t.ValidateVars(data); err != nil {
		return err
	}

	if t.Logger == nil {
		t.Logger = Log
	}
	if t.Resolver == nil {
		t.Resolver = CompareContent(UserResolver)
	}
	return t.installDir(".")
}

func (t *FSGenerator) installDir(dir string) error {
	var errs error
	entries, err := t.FS.ReadDir(path.Join(t.TrimPrefix, dir))
	if err != nil {
		return err
	}

	for _, entry := range entries {
		p := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			err := t.installDir(p)
			errs = errors.Join(errs, err)
		} else {
			err := t.installFile(p)
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (g *FSGenerator) pathTemplate(filePath string) string {
	p, err := g.renderString(filePath, filePath)
	if err != nil {
		panic(err)
	}

	return p
}
func (g *FSGenerator) installFile(filePath string) error {

	dest := filepath.Join(g.dest, filePath)
	if filepath.Ext(dest) == tmplExt {
		dest = dest[:len(dest)-len(tmplExt)]
	}

	dest = g.pathTemplate(dest)
	// Create directory if it does not exists
	dir := filepath.Dir(dest)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	f, err := os.CreateTemp(dir, filepath.Base(dest))
	if err != nil {
		return err
	}
	temp := f.Name()
	defer f.Close()
	defer os.Remove(temp)

	origin, err := g.FS.Open(path.Join(g.TrimPrefix, filePath))
	if err != nil {
		return (err)
	}
	if filepath.Ext(filePath) == tmplExt {
		err = g.render(filePath, origin, f)
		if err != nil {
			panic(err)
		}
	}

	defer origin.Close()

	_, err = io.Copy(f, origin)
	if err != nil {
		return err
	}

	f.Close()

	return os.Rename(temp, dest)

}

func (g *FSGenerator) renderString(name, tmpl string) (string, error) {
	in := strings.NewReader(tmpl)
	out := &bytes.Buffer{}
	err := g.render(name, in, out)
	return out.String(), err
}

func (g *FSGenerator) render(name string, in io.Reader, out io.Writer) error {
	t := template.New(name)
	if g.FuncMap != nil {
		t = t.Funcs(g.FuncMap)
	}

	t.Funcs(g.funcs())

	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	t, err = t.Parse(string(data))
	if err != nil {
		panic(fmt.Errorf("error parsing template %s: %w", name, err))
	}
	return t.Execute(out, g.data)
}

func (g *FSGenerator) funcs() template.FuncMap {
	return template.FuncMap{
		"underscorize": lazysupport.Underscorize,
	}
}
