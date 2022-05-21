# layout

Package layout provides helpers to generate an html document

## Functions

### func [AddComponent](/layout.go#L29)

`func AddComponent(c Component)`

### func [Layout](/layout.go#L25)

`func Layout(content ...interface{}) io.WriterTo`

```golang
nodes.Beautify = true
defer (func() {
    nodes.Beautify = false
})()

template := &LayoutTemplate{}

template.With("hola mundo").WriteTo(os.Stdout)
```

 Output:

```
<html>
<head/>
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

template := &LayoutTemplate{
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
    LayoutBody: func(l *LayoutTemplate, content ...interface{}) io.WriterTo {
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

### func [LayoutBody](/layout.go#L96)

`func LayoutBody(l *LayoutTemplate, content ...interface{}) io.WriterTo`

### func [PageStyle](/layout.go#L111)

`func PageStyle() string`

### func [SimpleCSS](/layout.go#L178)

`func SimpleCSS() string`

## Sub Packages

* [lazylayout](./lazylayout)

## Examples

### Component

```golang
nodes.Beautify = true
defer (func() {
    nodes.Beautify = false
})()

template := &LayoutTemplate{}
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

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
