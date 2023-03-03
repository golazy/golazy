# lazylayout

## Variables

```golang
var Layout = &document.Document{
    Lang:     "en",
    Title:    "golazy",
    Viewport: "width=device-width",
    Styles:   []string{style},
    Head:     []any{Script(Type("module"), Src("https://cdn.skypack.dev/@hotwired/turbo"))},
}
```

## Functions

### func [PageHeader](/layout.go#L22)

`func PageHeader() io.WriterTo`

### func [PageNav](/layout.go#L26)

`func PageNav() io.WriterTo`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
