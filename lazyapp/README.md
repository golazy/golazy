# lazyapp

`lazyapp` wires a GoLazy application into one `http.Handler`.

It is intentionally small. Application code gives it the pieces that already
exist in the app:

```go
lazyapp.New(lazyapp.Config{
    Name:    "sample_app",
    Drawer:  Draw,
    Public:  app.Public,
    Views:   app.Views,
    Context: Context,
    Assets:  []lazyassets.Source{generatedAssets},
})
```

`lazyapp.New`:

- Opens views and initializes the renderer.
- Calls the application context initializer.
- Creates an asset registry.
- Registers public and generated assets.
- Creates the root `lazyroutes.Scope`.
- Calls the route drawer.
- Registers router helpers, asset helpers, and application helpers.
- Creates a `lazydispatch.Dispatcher`.
- Installs route-only response buffering and ETag handling.
- Installs application middleware.
- Installs the router middleware.
- Installs asset serving as the public fallback.

The returned `App` implements `http.Handler` and is normally passed directly to
`http.Server`.
