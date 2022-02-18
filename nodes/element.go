package nodes

import (
	"fmt"
	"io"
)

type Element struct {
	tag        string
	children   []io.WriterTo
	attributes []Attr // Attr List of element attributes
}

func (r *Element) add(something interface{}) {
	switch v := something.(type) {
	case *Element:
		r.children = append(r.children, v)
	case string:
		r.children = append(r.children, Text(v))
	case Attr:
		r.attributes = append(r.attributes, v)
	case []interface{}:
		for _, arg := range v {
			r.add(arg)
		}
	default:
		panic(fmt.Errorf("don't recognize that: %v", v))
	}
}

type writeSession struct {
	io.Writer
	n   int64
	err error
}

func (w *writeSession) WriteS(s string) (n int, err error) {
	return w.Write([]byte(s))
}
func (w *writeSession) Write(data []byte) (n int, err error) {
	if w.err != nil {
		return
	}
	n, err = w.Writer.Write(data)
	w.n += int64(n)
	w.err = err
	return
}

// WriteTo writes the current string to the writer w
func (r Element) WriteTo(w io.Writer) (n64 int64, err error) {

	session := &writeSession{Writer: w}

	isEmptyElement := false
	if len(r.children) == 0 ||
		r.tag == "area" ||
		r.tag == "base" ||
		r.tag == "br" ||
		r.tag == "col" ||
		r.tag == "embed" ||
		r.tag == "hr" ||
		r.tag == "img" ||
		r.tag == "input" ||
		r.tag == "keygen" ||
		r.tag == "link" ||
		r.tag == "meta" ||
		r.tag == "param" ||
		r.tag == "source" ||
		r.tag == "track" ||
		r.tag == "wbr" {
		isEmptyElement = true
	}

	// Open tag
	session.WriteS("<" + r.tag)

	// Process atributes
	nAttr := len(r.attributes)
	if nAttr != 0 {
		session.WriteS(" ")

		// Write the list of tags
		for i, attr := range r.attributes {
			// Write a space before
			if i != 0 {
				// Put space before except for the first
				session.WriteS(" ")
			}
			attr.WriteTo(session)
		}

	}
	// Is an empty element
	// TODO
	if isEmptyElement {
		session.WriteS("/>")
		return session.n, session.err
	}

	session.WriteS(">")

	// Content
	for _, c := range r.children {
		c.WriteTo(session)
	}
	// Close
	session.WriteS("</" + r.tag + ">")
	return
}

// NewElement creates a new element with the provided tagname and the provided options
// The options can be:
//
// * An Attr that will be render
// * A string or Text
// * Another Element
// * Any WriterTo interface
// Attributes are output in order
// The rest is output in the same order as received
func NewElement(tagname string, options ...interface{}) *Element {
	r := &Element{
		tag: tagname,
	}
	r.add(options)
	return r
}
