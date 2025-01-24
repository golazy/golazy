package head

import (
	"io"
	"slices"
)

//go:generate stringer -type=Category -trimprefix=Head
type Category uint8

const (
	HeadTitle Category = iota
	HeadBase
	HeadLink
	HeadMeta
	HeadStyle
	HeadScript
	HeadNoscript
	HeadTemplate
)

type Component interface {
	io.WriterTo
	Category() Category
}

// Head is a collection of components that make up the header of an HTML document.
type Head struct {
	Components []Component
}

func (h *Head) Add(c Component) {
	h.Components = append(h.Components, c)
}

// WriteTo write the header to the writer.
func (h Head) WriteTo(w io.Writer) (int64, error) {

	slices.SortFunc(h.Components, func(a, b Component) int {
		return int(a.Category()) - int(b.Category())
	})
	var n int64
	for _, c := range h.Components {
		n1, err := c.WriteTo(w)
		n += n1
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
