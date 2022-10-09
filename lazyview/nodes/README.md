# nodes

Package nodes provides data structures to represents Html ElementNodes,
TextNodes and Attributes.

You are free to use this package, but is probably more comfortable to use the
html package, that already have html elements and common attributes.

## Variables

```golang
var Beautify = true
```

## Types

### type [Attr](/attr.go#L10)

`type Attr struct { ... }`

Attr holds information about an attribute for an Element Node

#### func [NewAttr](/attr.go#L17)

`func NewAttr(key string, value ...string) Attr`

NewAttr creates a new attribute.
If several arguments are given, they are join by a space

#### func (Attr) [String](/attr.go#L66)

`func (a Attr) String() string`

#### func (Attr) [WriteTo](/attr.go#L26)

`func (a Attr) WriteTo(w io.Writer) (n int64, err error)`

WriteTo writes the current string to the writer w

### type [ContentNode](/content_node.go#L5)

`type ContentNode []io.WriterTo`

#### func (ContentNode) [WriteTo](/content_node.go#L7)

`func (c ContentNode) WriteTo(w io.Writer) (n64 int64, err error)`

### type [Element](/element.go#L13)

`type Element struct { ... }`

#### func [NewElement](/element.go#L289)

`func NewElement(tagname string, options ...interface{ ... }) Element`

NewElement creates a new element with the provided tagname and the provided options
The options can be:

* An Attr that will be render
* A string or Text
* Another Element
* Any WriterTo interface
Attributes are output in order
The rest is output in the same order as received

```golang
Beautify = false
content := NewElement("html", NewAttr("lang", "en"),
    NewElement("head",
        NewElement("title", "Mi pagina")),
    NewElement("body",
        NewElement("h1", "This is my page")),
)

content.WriteTo(os.Stdout)
```

 Output:

```
<!DOCTYPE html><html lang=en><head><title>Mi pagina</title><body><h1>This is my page</h1>
```

#### func (Element) [String](/element.go#L274)

`func (r Element) String() string`

#### func (Element) [WriteTo](/element.go#L226)

`func (r Element) WriteTo(w io.Writer) (n64 int64, err error)`

WriteTo writes the current string to the writer w

### type [Raw](/text.go#L10)

`type Raw string`

#### func (Raw) [WriteTo](/text.go#L13)

`func (t Raw) WriteTo(w io.Writer) (int64, error)`

WriteTo writes the current string to the writer w without any escape

### type [Text](/text.go#L9)

`type Text string`

Text represents a TextNode

#### func (Text) [WriteTo](/text.go#L19)

`func (t Text) WriteTo(w io.Writer) (int64, error)`

WriteTo writes the current string to the writer w while escapeing html with [https://godocs.io/html#EscapeString](https://godocs.io/html#EscapeString)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
