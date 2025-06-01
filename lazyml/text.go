package lazyml

import (
	"html"
	"io"
)

// Text represents a TextNode
type Text string
type Raw string

// WriteTo writes the current string to the writer w without any escape
func (t Raw) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(t))
	return int64(n), err
}

// WriteTo writes the current string to the writer w while escapeing html with https://godocs.io/html#EscapeString
func (t Text) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(html.EscapeString(string(t))))
	return int64(n), err
}
