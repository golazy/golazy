package nodes

import (
	"fmt"
	"io"
	"strings"

	"golazy.dev/lazysupport"
)

var Beautify = true

type Element struct {
	tag        string
	children   []io.WriterTo
	attributes []Attr // Attr List of element attributes
}

func (r *Element) add(something interface{}) {
	switch v := something.(type) {
	case Element:
		r.children = append(r.children, v)
	case Raw:
		r.children = append(r.children, v)
	case string:
		r.children = append(r.children, Text(v))
	case Attr:
		r.attributes = append(r.attributes, v)
	case []Element:
		for _, arg := range v {
			r.add(arg)
		}
	case []io.WriterTo:
		for _, arg := range v {
			r.add(arg)
		}
	case []interface{}:
		for _, arg := range v {
			r.add(arg)
		}
	case io.WriterTo:
		r.children = append(r.children, v)
	case nil:

	default:
		panic(fmt.Errorf("when processing elements of a view, found an unexpected type: %T inside %v", v, r.tag))
	}
}

type writeSession struct {
	io.Writer
	n     int64
	err   error
	level int
}

func (w *writeSession) NewLine() {
	if Beautify {
		w.WriteS("\n")
	}
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

// voidElements don't require a closing tag neither need to be self close
var voidElements = lazysupport.NewSet(
	"area",
	"base",
	"br",
	"col",
	"embed",
	"hr",
	"img",
	"input",
	"keygen",
	"link",
	"meta",
	"param",
	"source",
	"track",
	"wbr",
)

// html elements that don't require a closing tag
var skipCloseTag = lazysupport.NewSet(
	"html",
	"head",
	"body",
	"p",
	"li",
	"dt",
	"dd",
	"option",
	"thead",
	"th",
	"tbody",
	"tr",
	"td",
	"tfoot",
	"colgroup",
)

// https://developer.mozilla.org/en-US/docs/Web/HTML/Inline_elements
var inlineElements = lazysupport.NewSet(
	"a",
	"abbr",
	"acronym",
	"audio",
	"b",
	"bdi",
	"bdo",
	"big",
	"br",
	"button",
	"canvas",
	"cite",
	"code",
	"data",
	"datalist",
	"del",
	"dfn",
	"em",
	"embed",
	"i",
	"iframe",
	"img",
	"input",
	"ins",
	"kbd",
	"label",
	"map",
	"mark",
	"meter",
	"noscript",
	"object",
	"output",
	"picture",
	"progress",
	"q",
	"ruby",
	"s",
	"samp",
	"script",
	"select",
	"slot",
	"small",
	"span",
	"strong",
	"sub",
	"sup",
	"svg",
	"template",
	"textarea",
	"time",
	"u",
	"tt",
	"var",
	"video",
	"wbr",
	// Plus some that are not styled like the ones in head
	"title",
	"meta",
	// Plus some that are rendered as block by usually formated as onelines
	"h1",
	"h2",
	"h3",
	"h4",
	"h5",
	"h6",
	"h7",
)

func (r Element) isInline() bool {
	for _, e := range r.children {
		switch child := e.(type) {
		case Element:
			if !child.isInline() {
				return false
			}
		case Text:
		default:
			return false
		}
	}
	return inlineElements.Has(r.tag)
}

// Rule to render a the content of a tag inline
// The tag is title, p, b, strong, i, em,li or there are no Elements inside

func (r Element) writeOpenTag(session *writeSession) {
	if Beautify {
		for i := 0; i < session.level; i++ {
			session.WriteS("  ")
		}
	}
	if r.tag == "html" {
		session.WriteS("<!DOCTYPE html>")
		session.NewLine()
	}

	// Open tag
	session.WriteS("<" + r.tag)

	// Process atributes
	for _, attr := range r.attributes {
		session.WriteS(" ")
		attr.WriteTo(session)
	}
	session.WriteS(">")
}

// WriteTo writes the current string to the writer w
func (r Element) WriteTo(w io.Writer) (n64 int64, err error) {

	var session *writeSession

	if s, ok := w.(*writeSession); ok {
		session = s
	} else {
		session = &writeSession{Writer: w, level: 0}
	}

	r.writeOpenTag(session)

	if voidElements.Has(r.tag) {
		return session.n, session.err
	}

	// Content
	isInline := r.isInline()
	if !isInline {
		session.level = session.level + 1
	}
	for _, c := range r.children {
		if r.tag == "html" {
			session.NewLine()
			c.WriteTo(session)
			continue
		}
		if !isInline {
			session.NewLine()
		}
		c.WriteTo(session)
	}
	if !isInline {
		session.NewLine()
		session.level = session.level - 1
	}

	// Some elements
	if skipCloseTag.Has(r.tag) {
		//session.WriteS("\n")
		return session.n, session.err
	}

	// Close tag
	session.WriteS("</" + r.tag + ">")
	return session.n, session.err
}

func (r Element) String() string {
	buf := &strings.Builder{}
	r.WriteTo(buf)
	return buf.String()
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
func NewElement(tagname string, options ...interface{}) Element {
	r := Element{
		tag: tagname,
	}
	r.add(options)
	return r
}
