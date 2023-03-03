package component

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/style"
)

type URL struct {
	URL        string
	Path       string
	ImportName string
	Script     []script.Script
	Style      []style.Style
}

func (u *URL) Install(opts InstallOptions) error {

	f, err := os.CreateTemp("", "golazy")
	if err != nil {
		return err
	}

	path := f.Name()

	req, err := http.Get(u.URL)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	_, err = io.Copy(f, req.Body)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return os.Rename(path, filepath.Join(opts.Path, u.destPath()))
}

func (u *URL) destPath() string {
	if u.Path != "" {
		return u.Path
	}

	url, err := url.Parse(u.URL)
	if err != nil {
		panic(err)
	}

	return url.Path
}

func (u *URL) Uninstall(opts InstallOptions) error {
	return os.Remove(filepath.Join(opts.Path, u.destPath()))
}

func (u *URL) ImportMap() ImportMap {
	if u.ImportName == "" {
		return nil
	}
	im := make(ImportMap)
	im[u.ImportName] = u.destPath()
	return im
}
