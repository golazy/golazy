package head

import (
	"io"

	"golazy.dev/lazyml/html"
)

// Title represents a title component.
type Title string

// Category returns HeadTitle
func (t Title) Category() Category {
	return HeadTitle
}

// WriteTo writes the title to the writer.
func (t Title) WriteTo(w io.Writer) (int64, error) {
	return html.Title(string(t)).WriteTo(w)
}
