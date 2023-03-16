package component

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"golazy.dev/lazysupport"
	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/style"
)

type URL struct {
	URL        string
	Name       string
	Path       string
	ImportName string
	Scripts    []script.Script
	Styles     []style.Style
	Head       []io.WriterTo
}

func (u *URL) String() string {
	if u.Name != "" {
		return u.Name
	}
	file := path.Base(u.URL)
	ext := path.Ext(file)
	return lazysupport.Camelize(file[:len(file)-len(ext)])
}
func (u *URL) Install(opts InstallOptions) error {

	f, err := os.OpenFile(filepath.Join(opts.Path, u.destPath()), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

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
	return nil

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

func (u *URL) Installed(opts InstallOptions) bool {
	_, err := os.Stat(filepath.Join(opts.Path, u.destPath()))
	return err == nil
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

func (u *URL) PageScripts() []script.Script {
	return u.Scripts
}
