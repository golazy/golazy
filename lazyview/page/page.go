// package document provides helpers to generate an html document
package page

import (
	"encoding/json"
	"io"
	"strings"

	"golazy.dev/lazyview/component"
	. "golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
	"golazy.dev/lazyview/script"
	"golazy.dev/lazyview/static_files"
	"golazy.dev/lazyview/style"
)

type Page struct {
	Files       *static_files.Manager
	Styles      []style.Style
	Scripts     []script.Script
	Description string
	Keywords    string
	Charset     string
	Head        []io.WriterTo
	Lang        string
	Title       string
	Viewport    string
	Content     any
	ImportMap   component.ImportMap
	Components  []component.Component
}

func (p *Page) AddScript(s script.Script) {
	p.Scripts = append(p.Scripts, s)
}

func (p *Page) Use(c component.Component) {
	p.Components = append(p.Components, c)
}

func (p *Page) AddStyle(s style.Style) {
	p.Styles = append(p.Styles, s)
}

func (p *Page) Element() *nodes.Element {
	var lang io.WriterTo
	if p.Lang != "" {
		lang = Lang(p.Lang)
	}
	var title io.WriterTo
	if p.Title != "" {
		title = Title(p.Title)
	}
	var viewport io.WriterTo
	if p.Viewport != "" {
		viewport = Meta(Name("viewport"), ContentAttr(p.Viewport))
	}
	var charset io.WriterTo
	if p.Charset != "" {
		charset = Meta(Charset(p.Charset))
	}

	var description io.WriterTo
	if p.Description != "" {
		description = Meta(Name("description"), ContentAttr(p.Description))
	}

	var keywords io.WriterTo
	if p.Keywords != "" {
		keywords = Meta(Name("keywords"), ContentAttr(p.Keywords))
	}

	e := Html(
		lang,
		Head(
			charset,
			viewport,
			title,
			description,
			keywords,
			p.styles(),
			p.importMaps(),
			p.scripts(),
			p.head(),
		),
		Body(p.Content),
	)

	return &e

}

func (p *Page) With(content any) io.WriterTo {
	p.Content = content
	return p.Element()
}

func (p *Page) WriteTo(w io.Writer) (int64, error) {
	return p.Element().WriteTo(w)
}

func (p *Page) head() []io.WriterTo {

	head := []io.WriterTo{}
	for _, c := range p.Components {
		if c, ok := (c).(component.ComponentWithHead); ok {
			head = append(head, c.PageHead()...)
		}
	}
	head = append(head, p.Head...)

	return head
}

func (p *Page) scripts() []io.WriterTo {
	scripts := []io.WriterTo{}
	for _, c := range p.Components {
		if c, ok := (c).(component.ComponentWithScripts); ok {
			for _, s := range c.PageScripts() {

				if f := p.findAsset(s.Src); f != nil {
					s.Src = f.Permalink
					s.Integrity = f.Hash.Integrity()
				}
				scripts = append(scripts, s.Element())
			}
		}
	}
	for _, s := range p.Scripts {
		if f := p.findAsset(s.Src); f != nil {
			s.Integrity = f.Hash.Integrity()
			s.Src = f.Permalink
		}
		scripts = append(scripts, s.Element())
	}

	return scripts

}

func (p *Page) toPermalink(path string) string {
	if p.Files == nil {
		return path
	}
	if strings.Contains(path, "://") || p.Files == nil {
		return path
	}
	if len(path) < 1 {
		return path
	}

	link, err := p.Files.Permalink(path[1:])
	if err != nil {
		panic(err)
	}

	return link
}

func (p *Page) findAsset(asset_path string) *static_files.File {
	if p.Files == nil {
		return nil
	}
	f, _ := p.Files.Find(asset_path)
	return f
}

func (p *Page) importMaps() io.WriterTo {

	im := make(component.ImportMap)

	for _, c := range p.Components {
		if c, ok := (c).(component.ComponentWithMaps); ok {
			im.Merge(c.ImportMap())
		}
	}

	im.Merge(p.ImportMap)

	if len(im) == 0 {
		return nil
	}

	// Use permalinks
	if p.Files != nil {
		for k, v := range im {
			im[k] = p.toPermalink(v)
		}
	}
	mapJSON, err := json.Marshal(struct {
		ImportMap map[string]string `json:"imports"`
	}{im})

	if err != nil {
		panic(err)
	}

	return Script(Type("importmap"), mapJSON)

}

func (p *Page) styles() []io.WriterTo {
	// TODO: Get Component Styles
	styles := []io.WriterTo{}
	for _, c := range p.Components {
		if c, ok := (c).(component.ComponentWithStyles); ok {
			for _, s := range c.PageStyles() {
				styles = append(styles, s.Element())
			}
		}
	}
	for _, s := range p.Styles {
		styles = append(styles, s.Element())
	}
	return styles
}

func (p *Page) String() string {
	return p.Element().String()
}
