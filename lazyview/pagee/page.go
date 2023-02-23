package page

import (
	"io"

	. "golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

type Page struct {
	Styles      []string
	Description string
	Charset     string
	Scripts     []string
	Head        []interface{}
	Lang        string
	Title       string
	Viewport    string
	Content     []interface{}
	Components  []Component
}

func (p *Page) AddStyle(s string) {
	p.Styles = append(p.Styles, s+"\n")
}

func (p *Page) AddScript(s string) {
}

func (p *Page) Merge(other *Page) {
	if other == nil {
		return
	}
	p.Styles = append(p.Styles, other.Styles...)
	p.Scripts = append(p.Scripts, other.Scripts...)
	p.Head = append(p.Head, other.Head...)
	if p.Lang == "" {
		p.Lang = other.Lang
	}
	if p.Title == "" {
		p.Title = other.Title
	}
	if p.Charset == "" {
		p.Charset = other.Charset
	}
	if p.Description == "" {
		p.Description = other.Description
	}
	if p.Viewport == "" {
		p.Viewport = other.Viewport
	}
}

func (p *Page) Render(content ...interface{}) io.WriterTo {

	styles := []nodes.Element{}
	for _, s := range p.Styles {
		styles = append(styles, Style(nodes.Raw(s)))
	}

	var scripts []interface{}
	if len(p.Scripts) > 0 {
		for _, s := range p.Scripts {
			scripts = append(scripts, Script(nodes.Raw(s)))
		}
	}

	head := p.Head

	// Append components
	for _, c := range p.Components {
		for _, s := range c.Scripts {
			scripts = append(scripts, Script(nodes.Raw(s)))
		}
		head = append(head, c.Head...)
		for _, s := range c.Styles {
			styles = append(styles, Style(s))
		}
	}

	var body interface{}
	if p.Content != nil {
		body = p.Content
	}

	var lang interface{}
	if p.Lang != "" {
		lang = Lang(p.Lang)
	}
	var title interface{}
	if p.Title != "" {
		title = Title(p.Title)
	}

	var viewport interface{}
	if p.Viewport != "" {
		viewport = Meta(Name("viewport"), ContentAttr(p.Viewport))
	}

	if p.Charset != "" {
		head = append(head, Meta(Charset(p.Charset)))
	}
	if p.Description != "" {
		head = append(head, Meta(Name("description"), ContentAttr(p.Description)))
	}

	return Html(
		lang,
		Head(
			title,
			viewport,
			styles,
			scripts,
			head,
		),
		body,
	)
}
