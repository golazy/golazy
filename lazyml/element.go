package lazyml

import (
	"fmt"
	"html/template"
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

var Nil = Element{}

// NewElement creates a new element with the provided tagname and the provided options
// The options can be:
//
// * An Attr that will be render
// * A string or Text
// * Another Element
// * Any WriterTo interface
// Attributes are output in order
// The rest is output in the same order as received
func NewElement(tagname string, options ...any) Element {
	r := Element{
		tag: tagname,
	}
	r.add(options)
	return r
}
func (r Element) IsNil() bool {
	return r.tag == "" && len(r.children) == 0 && len(r.attributes) == 0
}

// WriteTo writes the current string to the writer w
func (r Element) WriteTo(w io.Writer) (n64 int64, err error) {
	if r.IsNil() {
		return 0, nil
	}

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

func (r Element) HTML() template.HTML {
	return template.HTML(r.String())
}
func (r *Element) add(something any) {
	switch v := something.(type) {
	case Element:
		r.children = append(r.children, v)
	case Raw:
		r.children = append(r.children, v)
	case Attr:
		r.attributes = append(r.attributes, v)
	case []Attr:
		r.attributes = append(r.attributes, v...)
	case []Element:
		for _, arg := range v {
			r.add(arg)
		}
	case io.WriterTo:
		r.children = append(r.children, v)
	case []io.WriterTo:
		for _, arg := range v {
			r.add(arg)
		}
	case []any:
		for _, arg := range v {
			r.add(arg)
		}

	case int64, int32, int16, int8, int:
		r.children = append(r.children, Text(fmt.Sprintf("%d", v)))
	case uint64, uint32, uint16, uint8, uint:
		r.children = append(r.children, Text(fmt.Sprintf("%d", v)))
	case []byte:
		r.children = append(r.children, Raw(v))
	case string:
		r.children = append(r.children, Text(v))
	case bool:
		r.children = append(r.children, Text(fmt.Sprint(v)))
	case nil:

	default:
		panic(fmt.Errorf("when processing elements of a view, found an unexpected type: %T inside %v", v, r.tag))
	}
}

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
	//	"p",
	"li",
	"dt",
	"dd",
	//	"option",
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
