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
})
```

`lazyapp.New`:

- Opens views and initializes the renderer.
- Calls the application context initializer.
- Creates the root `lazyroutes.Scope`.
- Calls the route drawer.
- Creates a `lazydispatch.Dispatcher`.
- Installs application middleware.
- Installs the router middleware.
- Installs public static-file middleware.

The returned `App` implements `http.Handler` and is normally passed directly to
`http.Server`.
