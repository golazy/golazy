package lazycontroller

import (
	"mime/multipart"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"golazy.dev/lazysupport"
)

type Params struct {
	r *http.Request
}

var MaxFormSize int64 = 10 * 1024 * 1024

func (c *Controller) Gen_Params() *Params {
	if c.params.r == nil {
		c.Request.ParseMultipartForm(MaxFormSize)
		c.params.r = c.Request
	}
	return &c.params
}

func (p *Params) With(name string, fn func(string)) {
	if p.IsPresent(name) {
		fn(p.Get(name))
	}
}

func (p *Params) IsPresent(name string) bool {
	_, ok := p.r.Form[name]
	return ok
}

type File struct {
	Name        string
	ContentType string
	Size        int64
	multipart.File
}

func (p *Params) WithFile(name string, fn func(*File)) {
	file := p.File(name)
	if file != nil {
		fn(file)
	}
}

func (p *Params) File(name string) *File {

	formFile, header, err := p.r.FormFile(name)
	if err != nil {
		return nil
	}
	return &File{
		File:        formFile,
		Name:        header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
	}

}
func (p *Params) Get(name string) string {
	return p.r.Form.Get(name)
}

// GetString is similar to get but it strips tags and remove spaces
func (p *Params) GetString(name string) string {
	return strings.TrimSpace(lazysupport.StripTags(p.Get(name)))
}

func (p *Params) GetInt(name string) int {
	i, err := strconv.ParseInt(p.Get(name), 10, 64)
	if err != nil {
		return 0
	}
	return int(i)
}
func (p *Params) GetUint(name string) uint {
	i, err := strconv.ParseUint(p.Get(name), 10, 64)
	if err != nil {
		return 0
	}
	return uint(i)
}

func (p *Params) GetBool(name string) (value bool, ok bool) {
	v := p.Get(name)
	if v == "" {
		return false, false
	}
	if slices.Contains(trueValues, v) {
		return true, true
	}
	if slices.Contains(falseValues, v) {
		return false, true
	}
	return false, false
}
