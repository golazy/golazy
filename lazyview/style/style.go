package style

import (
	"io"

	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

type Style struct {
	Href    string
	Content string
	Media   string
}

func (s *Style) Element() nodes.Element {
	opts := []nodes.Attr{}
	if s.Media != "" {
		opts = append(opts, html.Media(s.Media))
	}

	if s.Content != "" {
		return html.Style(s.Content, opts)
	}
	return html.Link(html.Href(s.Href), html.Rel("stylesheet"), opts)

}

func (s *Style) WriteTo(w io.Writer) (int64, error) {
	return s.Element().WriteTo(w)
}
