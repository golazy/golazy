package layout_manager

import "io"

type Header struct {
	Lang    string
	Title   string
	Content []any
}

func (h *Header) Add(data io.Writer) {
	h.Content = append(h.Content, data)
}

func (h *Header) Merge(other *Header) {
	if h.Lang == "" {
		h.Lang = other.Lang
	}
	if h.Title == "" {
		h.Title = other.Title
	}
	h.Content = append(h.Content, other.Content...)
}
