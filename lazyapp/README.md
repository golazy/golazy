# lazyapp

`lazyapp` wires a GoLazy application into one `http.Handler`.

It is intentionally small. Application code gives it the pieces that already
exist in the app:

```go
app := lazyapp.New(lazyapp.Config{
    Name:    "sample_app",
    Drawer:  Draw,
    Public:  app.Public,
    Views:   app.Views,
    Dependencies: Dependencies,
    Helpers: lazyapp.Helpers{helpers.RegisterHelpers()},
    Assets:  []lazyassets.Source{generatedAssets},
    Sessions: lazysession.Config{
        Key: os.Getenv("SECURE_COOKIE_KEY"),
    },
})
log.Fatal(app.ListenAndServe())
```

`lazyapp.New`:

- Opens views and initializes the renderer.
- Calls the application dependency initializer.
- Initializes optional background jobs through `lazyjobs.JobRunner`.
- Creates an asset registry.
- Registers public and generated assets.
- Creates the root `lazyroutes.Scope`.
- Calls the route drawer.
- Evaluates SEO defaults with the dependency-initialized context.
- Registers router helpers, asset helpers, form helpers, SEO helpers, and
  application helpers.
- Caches views after helpers are registered.
- Creates a `lazydispatch.Dispatcher`.
- Installs route-only method override, response buffering, and ETag handling.
- Builds an optional control plane for liveness, readiness, metrics, and Go
  diagnostics.
- Installs application middleware.
- Installs generated `robots.txt` and configured sitemap middleware.
- Installs the router middleware.
- Installs asset serving as the public fallback.

The returned `App` implements `http.Handler`. `App.ListenAndServe` is the
default server shortcut; it uses `ADDR`, then `PORT`, then
`127.0.0.1:3000`. When `CONTROL_PLANE_ADDR` is set, it activates the default
control plane and either mounts it into the app server when the addresses match
or starts a second server when the address differs. Separate control-plane
servers automatically include `/debug/pprof/` and the standard pprof subpaths.
It also sets the server base context to `app.Context`, so request contexts
include the dependencies initialized by `lazyapp.New`.

When using your own `http.Server`, set `BaseContext` manually:

```go
server := &http.Server{
    Addr:    ":3000",
    Handler: app,
    BaseContext: func(_ net.Listener) context.Context {
        return app.Context
    },
}
log.Fatal(server.ListenAndServe())
```

When sessions are enabled and `Sessions.Name` is empty, the cookie name defaults
to the application name followed by `_session`. If the application name is a
module path, the cookie name uses the last path segment. `lazysession.Config.Key`
is expanded deterministically before the cookie store is created.

For embedded application files, `MustSub` converts an embedded root filesystem
into the function shape expected by `Config`:

```go
//go:embed views public
var files embed.FS

lazyapp.New(lazyapp.Config{
    Public: lazyapp.MustSub(files, "public"),
    Views:  lazyapp.MustSub(files, "views"),
})
```
