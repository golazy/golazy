package nodes

import "io"

// Text represents a TextNode
type Text string

// WriteTo writes the current string to the writer w
func (t Text) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(t))
	return int64(n), err
}
