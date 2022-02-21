package nodes

import (
	"io"
	"strings"
)

//Attr holds information about an attribute for an Element Node
type Attr struct {
	key  string
	prop *string
}

// NewAttr creates a new attribute.
// If several arguments are given, they are join by a space
func NewAttr(key string, value ...string) Attr {
	if len(value) == 0 {
		return Attr{key: key}
	}
	a := strings.Join(value, " ")
	return Attr{key: key, prop: &a}
}

// WriteTo writes the current string to the writer w
func (a Attr) WriteTo(w io.Writer) (int64, error) {
	n16, err := w.Write([]byte(a.key))
	if err != nil || a.prop == nil {
		return int64(n16), err
	}
	n := int64(n16)
	n16, err = w.Write([]byte("=\""))
	n += int64(n16)
	if err != nil {
		return n, err
	}

	s := strings.ReplaceAll(*a.prop, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "&", "&amp;")

	n16, err = w.Write([]byte(s))
	n += int64(n16)
	if err != nil {
		return n, err
	}

	n16, err = w.Write([]byte(`"`))
	n += int64(n16)
	return n, err
}
