package head

import (
	"io"

	"github.com/golazy/golazy/lazyml/html"

	"github.com/golazy/golazy/lazyml"
)

// Style represents a style component.
type Style struct {
	Href    string
	Content string
	Media   string
	Data    map[string]string
}

// Category returns the category of the component.
func (s *Style) Category() Category {
	return HeadStyle
}

func (s *Style) element() lazyml.Element {
	opts := []lazyml.Attr{}
	if s.Media != "" {
		opts = append(opts, html.Media(s.Media))
	}

	if s.Data != nil {
		for k, v := range s.Data {
			opts = append(opts, html.DataAttr(k, v))
		}
	}

	if s.Content != "" {
		return html.Style(s.Content, opts)
	}
	return html.Link(html.Href(s.Href), html.Rel("stylesheet"), opts)

}

// WriteTo writes the style to the writer.
func (s *Style) WriteTo(w io.Writer) (int64, error) {
	return s.element().WriteTo(w)
}
