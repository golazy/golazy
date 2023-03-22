package generator

import (
	"embed"
	"errors"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
)

const tmplExt = ".tmpl"

type Project struct {
	FS         embed.FS
	Dest       string
	FuncMap    template.FuncMap
	Data       any
	TrimPrefix string
}

func (t *Project) Install() error {
	return t.installDir(".")
}

func (t *Project) installDir(dir string) error {
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

func (i *Project) installFile(filePath string) error {

	dest := filepath.Join(i.Dest, filePath)
	if filepath.Ext(dest) == tmplExt {
		dest = dest[:len(dest)-len(tmplExt)]
	}
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

	origin, err := i.FS.Open(path.Join(i.TrimPrefix, filePath))
	if err != nil {
		return (err)
	}
	if filepath.Ext(filePath) == tmplExt {

		data, err := io.ReadAll(origin)
		if err != nil {
			panic(err)
		}

		t := template.New(filePath)
		if i.FuncMap != nil {
			t = t.Funcs(i.FuncMap)
		}
		t, err = t.Parse(string(data))
		if err != nil {
			panic(err)
		}

		err = t.Execute(f, i.Data)
		if err != nil {
			return err
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
