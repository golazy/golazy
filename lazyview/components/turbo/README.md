# turbo

## Variables

```golang
var TurboComponent = &component.Component{
    Sources: []component.Source{
        component.Npm{
            Name:    "@hotwired/turbo",
            Imports: map[string]string{turbo: "dist/turbo.es2017-esm.js"},
            Prefix:  "/",
        },
        GoogleFonts("Inter", "400,500,600,700"),
    },
    Script: `
		import { Turbo } from "@hotwired/turbo";
		Turbo.start();
	`,
    Style: `body{font-family: "Inter", sans-serif;}`,
    Head:  ``,
}
```

## Functions

### func [Action](/turbo.go#L53)

`func Action(value ...string) nodes.Attr`

### func [Autoscroll](/turbo.go#L50)

`func Autoscroll(value ...string) nodes.Attr`

### func [Busy](/turbo.go#L35)

`func Busy(value ...string) nodes.Attr`

### func [Complete](/turbo.go#L47)

`func Complete(value ...string) nodes.Attr`

### func [DataTurbo](/turbo.go#L61)

`func DataTurbo(value ...string) nodes.Attr`

### func [DataTurboAction](/turbo.go#L70)

`func DataTurboAction(value ...string) nodes.Attr`

### func [DataTurboCache](/turbo.go#L76)

`func DataTurboCache(value ...string) nodes.Attr`

### func [DataTurboConfirm](/turbo.go#L88)

`func DataTurboConfirm(value ...string) nodes.Attr`

### func [DataTurboEval](/turbo.go#L79)

`func DataTurboEval(value ...string) nodes.Attr`

### func [DataTurboFrame](/turbo.go#L67)

`func DataTurboFrame(value ...string) nodes.Attr`

### func [DataTurboMethod](/turbo.go#L82)

`func DataTurboMethod(value ...string) nodes.Attr`

### func [DataTurboPermanent](/turbo.go#L73)

`func DataTurboPermanent(value ...string) nodes.Attr`

### func [DataTurboPreload](/turbo.go#L91)

`func DataTurboPreload(value ...string) nodes.Attr`

### func [DataTurboStream](/turbo.go#L85)

`func DataTurboStream(value ...string) nodes.Attr`

### func [DataTurboTrack](/turbo.go#L64)

`func DataTurboTrack(value ...string) nodes.Attr`

### func [Disabled](/turbo.go#L38)

`func Disabled(value ...string) nodes.Attr`

### func [Loading](/turbo.go#L32)

`func Loading(value ...string) nodes.Attr`

### func [Src](/turbo.go#L29)

`func Src(value ...string) nodes.Attr`

### func [Target](/turbo.go#L44)

`func Target(value ...string) nodes.Attr`

### func [Targets](/turbo.go#L41)

`func Targets(value ...string) nodes.Attr`

### func [TurboFrame](/turbo.go#L25)

`func TurboFrame(options ...any) nodes.Element`

### func [TurboStream](/turbo.go#L57)

`func TurboStream(options ...any) nodes.Element`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
