package lazyml

import (
	"bytes"
	"html/template"
	"io"
)

type ContentNode []io.WriterTo

func (c ContentNode) String() string {

	buf := &bytes.Buffer{}
	for _, node := range c {
		node.WriteTo(buf)
	}

	return buf.String()

}

func (c ContentNode) HTML() template.HTML {
	return template.HTML(c.String())
}
func (c ContentNode) WriteTo(w io.Writer) (n64 int64, err error) {
	for _, node := range c {
		n, err := node.WriteTo(w)
		n64 += n
		if err != nil {
			return n64, err
		}
	}

	return
}

func NewContentNode(nodes ...io.WriterTo) ContentNode {
	return nodes
}

var Collection = NewContentNode
