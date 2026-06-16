# GoLazy

![GoLazy](https://golazy.dev/assets/golazy-horizontal.svg)

GoLazy is a convention-first web framework for Go. It keeps application code
close to normal Go while providing the framework pieces that make server-rendered
web applications pleasant to build: application assembly, routing, request-local
controllers, rendering, helpers, and asset serving.

The canonical module path is:

```text
golazy.dev
```

The `golazy.dev` vanity import resolves to
`https://github.com/golazy/golazy`.

## Why GoLazy

GoLazy is for Go developers who want the development speed of a conventional
web framework without leaving Go's normal shape. You can start with a single
file and a normal `main` package, then grow into the conventional application
layout when controllers, views, services, routes, and tests need clearer homes.

The default structure is meant to scale well for larger apps, teams, and coding
agents. It is encouraged, not required. Go packages remain the boundary.

## Goals

- Respect Go conventions: regular modules, regular commands, regular
  `go build`.
- Produce self-contained binaries with embedded production assets and views.
- Keep controller state request-local.
- Keep routing declarative and inspectable.
- Make the framework usable without requiring code generation or a custom build
  tool.

The `lazy` command is a helper for development and project creation. The
framework itself is just Go packages.

## Packages

```text
golazy.dev/lazyapp                 Application composition
golazy.dev/lazyassets              Asset registry, fingerprints, and serving
golazy.dev/lazycontroller          Request-local controllers and render state
golazy.dev/lazycookie              Signed and encrypted secure cookies
golazy.dev/lazydispatch            HTTP dispatch and middleware
golazy.dev/lazyroutes              Route DSL, resources, scopes, and route table
golazy.dev/lazysession             Cookie sessions and session middleware
golazy.dev/lazyview                View rendering and helper registry
golazy.dev/lazyview/gotmpl         html/template engine for lazyview
golazy.dev/lazysupport/inflection  Naming and inflection helpers
```

## Application Shape

A GoLazy app is assembled with `lazyapp.New`:

```go
package appinit

import (
    "os"

    "golazy.dev/lazyapp"
    "golazy.dev/lazysession"
    _ "golazy.dev/lazyview/gotmpl"
    "my_app/app"
)

func App() *lazyapp.App {
    return lazyapp.New(lazyapp.Config{
        Name:    "my_app",
        Drawer:  Draw,
        Public:  app.Public,
        Views:   app.Views,
        Context: Context,
        Sessions: lazysession.Config{
            Key: os.Getenv("SECURE_COOKIE_KEY"),
        },
    })
}
```

When `Sessions.Name` is omitted, `lazyapp` uses the application name followed
by `_session`. `lazysession.Config.Key` is deterministically expanded before it
is passed to the cookie signer, so templates can use a short development key and
production apps can load a stable value from `SECURE_COOKIE_KEY`.

The command entrypoint can then stay small:

```go
func main() {
    if err := appinit.App().ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

Routes are drawn through `lazyroutes.Scope`:

```go
func Draw(router *lazyroutes.Scope) {
    router.Get("/", home.New, (*home.HomeController).Index)
    router.Resources(posts.New)
}
```

Controllers are constructed when routes are drawn. GoLazy keeps a prototype and
uses pooled request instances, binding the current `*http.Request` before the
action runs:

```go
func New(ctx context.Context) (*PostsController, error) {
    base, err := controllers.NewBaseController(ctx)
    if err != nil {
        return nil, err
    }
    return &PostsController{BaseController: base}, nil
}

func (c *PostsController) Index(_ http.ResponseWriter, _ *http.Request) error {
    c.Set("title", "Posts")
    return nil
}
```

If a controller needs request-time setup, implement `BeforeAction` on the
controller or an embedded app base controller.

Actions that return without writing a response render the matching
controller/action view automatically.

Templates can use framework helpers registered by the app, router, and asset
registry:

```html
<a href="{{path_for "posts"}}">Posts</a>
<a href="{{path_for "post" .Post.Param}}">{{.Post.Title}}</a>
<link rel="stylesheet" href="{{asset_path "/styles.css"}}">
```

## Routing

`lazyroutes` provides HTTP verb methods, REST resources, and nested scopes:

```go
func Draw(router *lazyroutes.Scope) {
    router.Namespace("admin", func(admin *lazyroutes.Scope) {
        admin.Resources(posts.New)
    })
}
```

Every registered route is also recorded in a route table with its method, path,
name, controller, action, namespace, and named params. The CLI command
`lazy routes` uses the `printroutes` build tag to print this table without
starting the HTTP server.

## Views And Helpers

`lazyview` owns view rendering and helper registration. Template engines live in
subpackages. Applications opt into Go's `html/template` engine with:

```go
import _ "golazy.dev/lazyview/gotmpl"
```

Helpers are registered as plain Go functions:

```go
func Helpers() map[string]any {
    return map[string]any{
        "page_title": PageTitle,
    }
}
```

The view layer keeps the public helper shape independent from any one template
engine. The Go template engine adapts helpers into `template.FuncMap`
internally.

## Build And Deployment

GoLazy apps build with the normal Go toolchain:

```sh
go build ./cmd/app
```

Production builds embed views and public files, so the resulting binary is
self-contained. Public files are registered as assets with content-hashed
permanent URLs, ETags, integrity values, and cache headers. Development helpers
such as `lazy` may use build tags like `lazydev` to read views from disk while
editing.

## Documentation

Guides are published at [golazy.dev/guides/latest](https://golazy.dev/guides/latest/).
If you want the smallest possible starting point, begin with the
[single-file app guide](https://golazy.dev/guides/latest/single-file-app/).
When that starts to feel crowded, continue with
[Application Structure](https://golazy.dev/guides/latest/application-structure/).

The sample application is available at
[github.com/golazy/sample_app](https://github.com/golazy/sample_app).

## License

GoLazy is released under the MIT License. See [LICENSE](LICENSE).

The `lazycookie` and `lazysession` packages include code adapted from Gorilla
`securecookie` and Gorilla `sessions`. Those package directories retain the
Gorilla BSD-style license notice.
