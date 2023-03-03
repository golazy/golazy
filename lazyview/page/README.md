# page

package document provides helpers to generate an html document

## Types

### type [Page](/document.go#L12)

`type Page struct { ... }`

#### func (*Page) [AddComponent](/document.go#L25)

`func (p *Page) AddComponent(c *component.Component)`

#### func (*Page) [AddStyle](/document.go#L29)

`func (p *Page) AddStyle(s string)`

#### func (*Page) [With](/document.go#L33)

`func (p *Page) With(content ...any) io.WriterTo`

## Sub Packages

* [layouts](./layouts)

* [lazylayout](./lazylayout)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
