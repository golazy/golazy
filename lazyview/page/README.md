# document

package document provides helpers to generate an html document

## Variables

```golang
var BasicLayout = &Document{
    Lang:     "en",
    Title:    "lazyview",
    Viewport: "width=device-width",
    Styles:   []string{SimpleCSS(), PageStyle()},
    Head: []interface{}{
        Script(Async(), Src("https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js"), Crossorigin(("anonymous"))),
        Script(Type("module"),
            nodes.Raw(`import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';`),
        ),
    },
    LayoutBody: LayoutBody,
}
```

```golang
var DefaultLayout = &Document{}
```

## Functions

### func [AddComponent](/document.go#L28)

`func AddComponent(c Component)`

### func [Layout](/document.go#L24)

`func Layout(content ...interface{}) io.WriterTo`

```golang
nodes.Beautify = true
defer (func() {
    nodes.Beautify = false
})()

template := &Document{}

template.With("hola mundo").WriteTo(os.Stdout)
```

 Output:

```
<html>
<head>
</head>
<body>
hola mundo</body>
</html>
```

### Complete

```golang
nodes.Beautify = true
defer (func() {
    nodes.Beautify = false
})()

template := &Document{
    Lang:     "en",
    Title:    "lazyview",
    Viewport: "width=device-width",
    Styles:   []string{"body{margin:0;padding:0;box-sizing: border-box;}"},
    Head: []interface{}{
        Script(Async(), Src("https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js"), Crossorigin(("anonymous"))),
        Script(Type("module"),
            nodes.Raw(`import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';`),
        ),
    },
    Scripts: []string{
        `document.write("hello");`,
    },
    LayoutBody: func(l *Document, content ...interface{}) io.WriterTo {
        return Body(Main(content...))
    },
}

template.With("hello").WriteTo(os.Stdout)
```

 Output:

```
<html lang="en">
<head>
<title>lazyview</title>
<meta name="viewport" content="width=device-width"/>
<style>body{margin:0;padding:0;box-sizing: border-box;}</style>
<script>document.write("hello");</script>
<script async src="https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js" crossorigin="anonymous"/>
<script type="module">import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';</script>
</head>
<body>
<main>hello</main>
</body>
</html>
```

### func [LayoutBody](/document.go#L95)

`func LayoutBody(l *Document, content ...interface{}) io.WriterTo`

### func [PageStyle](/document.go#L110)

`func PageStyle() string`

### func [SimpleCSS](/document.go#L177)

`func SimpleCSS() string`

## Types

### type [Component](/component.go#L3)

`type Component struct { ... }`

```golang
nodes.Beautify = true
defer (func() {
    nodes.Beautify = false
})()

template := &Document{}
template.AddComponent(Component{
    Scripts: []string{`document.Write("hello world");`},
    Styles:  []string{`body{background: red;}`},
    Head: []interface{}{
        Script(Type("module"), Src("https://google.com/s.rs")),
    },
})

template.With("hola mundo").WriteTo(os.Stdout)
```

 Output:

```
<html>
<head>
<style>body{background: red;}</style>
<script>document.Write("hello world");</script>
<script type="module" src="https://google.com/s.rs"/>
</head>
<body>
hola mundo</body>
</html>
```

### type [Document](/document.go#L11)

`type Document struct { ... }`

#### func (*Document) [AddComponent](/document.go#L32)

`func (l *Document) AddComponent(c Component)`

#### func (*Document) [With](/document.go#L36)

`func (l *Document) With(content ...interface{ ... }) io.WriterTo`

## Sub Packages

* [lazylayout](./lazylayout)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
