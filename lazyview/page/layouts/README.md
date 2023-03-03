# layouts

## Variables

```golang
var BasicLayout = &page.Page{
    Lang:     "en",
    Title:    "lazyview",
    Viewport: "width=device-width",
    Styles:   []string{SimpleCSS(), PageStyle()},
    Head: []any{
        Script(Async(), nodes.NewAttr("nomodule"), Src("https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js"), Crossorigin(("anonymous"))),
        Script(Type("module"),
            nodes.Raw(`import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';`),
        ),
    },
}
```

## Functions

### func [LayoutBody](/basic_layout.go#L94)

`func LayoutBody(l *page.Page, content ...any) io.WriterTo`

### func [PageStyle](/basic_layout.go#L24)

`func PageStyle() string`

### func [SimpleCSS](/basic_layout.go#L91)

`func SimpleCSS() string`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
