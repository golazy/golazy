# lazydispatch

`lazydispatch` is the request dispatch layer used by `lazyapp`.

Most applications should not create a dispatcher directly. Configure dispatch
through `lazyapp.Config`, and let `lazyapp.New` create the dispatcher, install the
route-scoped response middleware, install the router middleware, install the
asset fallback, and return one application `http.Handler`.

## Current Responsibilities

`lazydispatch` currently owns:

- `Dispatcher`, the middleware runner and `http.Handler`.
- The `Middleware` interface.
- `MiddlewareFunc`, an adapter for ordinary functions.
- `RouteOnly`, a route-table gate for middleware.
- Router middleware.
- Response buffering for registered application routes.
- Dynamic `ETag` handling for eligible route responses.
- Low-level public static-file middleware for custom assemblies.
- Final `404 Not Found` behavior.

`lazydispatch` does not own:

- Route DSL construction.
- Route table metadata.
- Controller constructors and actions.
- Template rendering.
- Application service initialization.

Those stay in `lazyroutes`, `lazycontroller`, `lazyassets`, and the application.

## Use With lazyapp

Application middleware is configured through `lazyapp.Config.Middlewares`:

```go
func App() *lazyapp.App {
    return lazyapp.New(lazyapp.Config{
        Name:        "sample_app",
        Drawer:      Draw,
        Public:      app.Public,
        Views:       app.Views,
        Context:     Context,
        Middlewares: []lazydispatch.Middleware{
            requestIDMiddleware,
        },
    })
}
```

`lazyapp.New` builds this default chain:

```text
route-only response buffer and ETag handling
application middleware
router middleware
asset/public fallback middleware
404 final handler
```

Application middleware sees the request before route lookup and asset fallback.
Response buffering and dynamic ETags are gated by the route table, so public
assets are not buffered by the app response layer.

## Middleware

Middleware implements:

```go
type Middleware interface {
    Handler(next http.Handler) http.Handler
}
```

Use `MiddlewareFunc` for small middleware:

```go
var requestIDMiddleware = lazydispatch.MiddlewareFunc(
    func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(
            w http.ResponseWriter,
            r *http.Request,
        ) {
            next.ServeHTTP(w, r)
        })
    },
)
```

Middlewares run in the order they are listed in `lazyapp.Config.Middlewares`.

## Router Middleware

`lazyapp.New` installs the route scope through router middleware.

The router must implement:

```go
type RouteHandler interface {
    http.Handler
    HandlesPath(path string) bool
}
```

`lazyroutes.Scope` implements that interface.

The middleware calls `HandlesPath` first. If the router owns the path, the
router handles the request. If it does not, dispatch continues to the next
middleware.

Use `RouteOnly` to apply middleware only to routed application requests:

```go
dispatcher.Use(lazydispatch.RouteOnly(
    router,
    lazydispatch.ResponseBuffer(),
    lazydispatch.ETag(),
))
```

This is how `lazyapp` applies response buffering and ETag handling without
buffering public assets.

## ETag Responses

`ETag` uses buffered response state to add a SHA-256 validator to eligible
dynamic responses.

It applies to `GET` and `HEAD` `200 OK` responses. It skips responses with
`Cache-Control: no-store`, `Content-Encoding`, or `Set-Cookie`, and it honors an
existing `ETag` header instead of replacing it.

When `If-None-Match` matches, it resets the buffered response to:

```text
304 Not Modified
```

The 304 response keeps validator-related headers such as `ETag`,
`Cache-Control`, `Expires`, `Last-Modified`, and `Vary`, and drops body headers
such as `Content-Type`.

## Asset Fallback

`lazyapp.New` registers public files with `lazyassets` when
`lazyapp.Config.Public` is set:

```go
Public: app.Public,
```

The asset registry serves those files after route lookup with content type,
`ETag`, cache policy, and permanent hashed URLs.

For missing files, it calls the next handler. For existing assets with
unsupported methods, the asset fallback returns:

```text
405 Method Not Allowed
Allow: GET, HEAD
```

The lower-level `Public` middleware remains available for tests or custom
assemblies that intentionally want `http.FileServerFS` behavior without asset
hashing, helpers, generated assets, or cache metadata.

## Planned Request Logic

The dispatcher is the target package for request-wide behavior that should
surround controller actions:

- Request monitoring and tracing.
- Cookie lifecycle.
- Session lifecycle.
- Flash lifecycle.
- `HEAD` response adaptation.
- `Last-Modified` conditional responses.
- Action error response conversion.

## How to use without lazyapp

Use `lazydispatch` directly only when testing middleware or building a custom
application assembly.

Create a dispatcher manually:

```go
dispatcher := lazydispatch.NewDispatcher()
```

Register middleware manually:

```go
dispatcher.Use(middleware)
```

Install route-only response middleware manually:

```go
dispatcher.Use(lazydispatch.RouteOnly(
    router,
    lazydispatch.ResponseBuffer(),
    lazydispatch.ETag(),
))
```

Install a router manually:

```go
dispatcher.Use(lazydispatch.Router(router))
```

Install asset fallback manually:

```go
assets := lazyassets.New()
if err := assets.AddFS(publicFS); err != nil {
    return err
}
dispatcher.Use(lazydispatch.MiddlewareFunc(func(next http.Handler) http.Handler {
    return assets.Handler(next)
}))
```

Use the dispatcher as a handler:

```go
dispatcher.ServeHTTP(w, r)
```

Or build an explicit handler chain:

```go
handler := dispatcher.Handler(http.NotFoundHandler())
```

Manual assembly means you are responsible for creating the route scope,
initializing views, initializing application context, and ordering middleware
correctly.
