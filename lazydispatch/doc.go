// Package lazydispatch composes HTTP middleware, routes, and static fallback
// handlers for GoLazy.
//
// A Dispatcher stores middleware in the order they should run. Handler wraps a
// final http.Handler with that chain, and ServeHTTP uses http.NotFoundHandler as
// the final handler. Middleware can stop a request by not calling next, or pass
// control to later middleware and the final handler by calling next.ServeHTTP.
// When a request already has a lazytracing span, each middleware call is
// recorded as a child tracing region.
//
// Router adapts a RouteHandler into middleware. The RouteHandler reports
// whether it owns a path with HandlesPath; known paths are served by the router,
// and unknown paths fall through to the next handler. RouteOnly uses the same
// HandlesPath check to apply middleware only to routed requests. lazyapp uses
// RouteOnly with its lazyroutes router so route lifecycle middleware such as
// lazydispatch/middlewares.DynamicRoute runs for controller responses without
// wrapping generated metadata files or public assets.
//
// Static and Public serve files from an fs.FS only when the requested logical
// path already exists in that filesystem. Missing files fall through to the next
// handler, and existing files accept only GET and HEAD. This package does not
// build an asset manifest, calculate permanent hashed URLs, rewrite CSS, or
// make original files unavailable. Those generated-asset behaviors belong to
// lazyassets.Registry; lazyapp normally uses that registry for public assets
// instead of Static.
//
// ResponseBuffer delays headers and body until the downstream handler returns.
// BufferedResponseWriter also lets route lifecycle code reset an in-progress
// response or start a stream by committing headers and returning to the wrapped
// writer. ETag adds conditional response handling for buffered dynamic
// responses: eligible GET and HEAD 200 responses get a strong ETag based on the
// final body unless one is already set, and matching If-None-Match requests are
// converted to 304 Not Modified. Streaming, compressed, no-store, Set-Cookie,
// non-GET/HEAD, and non-OK responses are skipped.
//
// lazyapp builds the normal application chain for most projects: optional
// telemetry middleware, route-only DynamicRoute handling, sessions, configured
// application middleware, generated metadata files, the lazyroutes router, and
// finally lazyassets-backed public assets. Use this package directly when
// composing a custom net/http stack that wants the same middleware interface,
// route-only middleware, buffered dynamic responses, ETags, or simple fs.FS
// static fallback behavior outside a full lazyapp.App.
package lazydispatch
