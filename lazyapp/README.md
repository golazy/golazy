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
    Context: Context,
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
- Calls the application context initializer.
- Creates an asset registry.
- Registers public and generated assets.
- Creates the root `lazyroutes.Scope`.
- Calls the route drawer.
- Registers router helpers, asset helpers, form helpers, and application helpers.
- Caches views after helpers are registered.
- Creates a `lazydispatch.Dispatcher`.
- Installs route-only method override, response buffering, and ETag handling.
- Installs application middleware.
- Installs the router middleware.
- Installs asset serving as the public fallback.

The returned `App` implements `http.Handler`. `App.ListenAndServe` is the
default server shortcut; it uses `ADDR`, then `PORT`, then `:3000`. It also
sets the server base context to `app.Context`, so request contexts include the
dependencies initialized by `lazyapp.New`.

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
to the application name followed by `_session`. `lazysession.Config.Key` is
expanded deterministically before the cookie store is created.

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
