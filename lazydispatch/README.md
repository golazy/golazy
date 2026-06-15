# lazydispatch

`lazydispatch` is the request dispatch layer used by `lazyapp`.

Most applications should not create a dispatcher directly. Configure dispatch
through `lazyapp.Config`, and let `lazyapp.New` create the dispatcher, install the
router middleware, install the public-file middleware, and return one
application `http.Handler`.

## Current Responsibilities

`lazydispatch` currently owns:

- `Dispatcher`, the middleware runner and `http.Handler`.
- The `Middleware` interface.
- `MiddlewareFunc`, an adapter for ordinary functions.
- Router middleware.
- Public static-file middleware.
- `405 Method Not Allowed` for existing static files.
- Final `404 Not Found` behavior.

`lazydispatch` does not own:

- Route DSL construction.
- Route table metadata.
- Controller constructors and actions.
- Template rendering.
- Application service initialization.

Those stay in `lazyroutes`, `lazycontroller`, and the application until a later
refactor moves more request behavior into dispatch.

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
application middleware
router middleware
public static-file middleware
404 final handler
```

Application middleware sees the request before route lookup and public-file
fallback.

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

## Static Files

`lazyapp.New` installs static-file middleware when `lazyapp.Config.Public` is set:

```go
Public: app.Public,
```

The static middleware checks whether the requested file exists before handing
the request to `http.FileServerFS`.

For missing files, it calls the next handler. For existing files with
unsupported methods, it returns:

```text
405 Method Not Allowed
Allow: GET
```

## Planned Request Logic

The dispatcher is the target package for request-wide behavior that should
surround controller actions:

- Response buffering.
- `ETag` and `Last-Modified` conditional responses.
- Request monitoring and tracing.
- Cookie lifecycle.
- Session lifecycle.
- Flash lifecycle.
- `HEAD` response adaptation.
- Action error response conversion.

Response buffering should land before features that depend on final response
state, such as conditional responses and robust template failure handling.

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

Install a router manually:

```go
dispatcher.Use(lazydispatch.Router(router))
```

Install public files manually:

```go
dispatcher.Use(lazydispatch.Public(publicFS))
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
