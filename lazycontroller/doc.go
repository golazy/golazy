// Package lazycontroller provides the request-local controller layer used by a
// GoLazy application: controller lifecycle hooks, view rendering, response
// helpers, cache-key helpers, deferred view values, typed HTTP errors,
// redirects, content negotiation, SEO convenience methods, and Server-Sent
// Events entrypoints.
//
// In a normal GoLazy app, package lazyapp wires this package together with
// lazyroutes and lazyview. lazyapp creates the lazyview renderer, stores it in
// the application context with WithRenderer, asks lazyroutes to create or reuse
// a controller for each matched route, calls BindRequest with the matched
// lazyview.Route, runs BeforeAction when the controller implements it, and
// reports returned action errors through ErrorHandler. Controllers normally
// embed Base:
//
//	type PostsController struct {
//		lazycontroller.Base
//	}
//
//	func (c *PostsController) Index() error {
//		c.Set("Title", "Posts")
//		return nil
//	}
//
// When an action returns nil without writing a response, the lazyapp/lazyroutes
// integration renders the view matching the route action. Calling Render("")
// explicitly does the same lookup inside Base: the empty view name falls back to
// the current route action, lower-cased. Render passes controller data, helpers,
// route metadata, selected variants, layout choice, request context, and the
// ResponseWriter to lazyview.
//
// The package can also be used without lazyapp. Create a Renderer from an
// fs.FS, put it in context with WithRenderer, construct Base with NewBase, and
// bind each request with BindRequest before rendering. See ExampleBase_Render
// for the smallest standalone HTTP rendering setup.
//
// # Rendering conventions
//
// NewRenderer is a thin wrapper around lazyview.New. The filesystem must contain
// the views and layouts lazyview expects, including the default app layout when
// using the default Base layout. NewBase reads the renderer from context and
// keeps that application context as the fallback for each request. BindRequest
// then layers the request context above the application context so request
// values win while renderer, route helper, cache, build version, and other
// application-scoped values remain visible.
//
// Set and Helper provide request-local variables and functions to lazyview.
// Helpers can also be registered in bulk with Helpers. Package lazyapp normally
// collects application helpers from packages such as lazyassets and passes them
// into the renderer path; controller helpers are for per-request additions or
// overrides. Layout selects a named layout, NoLayout disables layout rendering,
// and Variants asks lazyview to prefer variant-specific templates.
//
// Render chooses the request format with Format, renders HTML with the selected
// layout, and renders non-HTML formats without a layout. RenderHTML forces HTML.
// RenderSVGString renders an SVG view to a string without writing the response.
// Turbo Frame requests are detected through package lazyturbo: a Turbo-Frame
// request renders the matching HTML partial and wraps it in a frame response.
// RenderTurboFrame renders an explicit frame partial named "<id>_frame".
//
// # Response and cache helpers
//
// Status changes the status code used by the next render without writing the
// response immediately. Header and ContentType mutate the current response
// headers. Redirect and its aliases validate internal or absolute locations,
// write the redirect immediately, and mark the response as sent.
//
// CacheKey and CacheKeyF connect controller rendering to lazycache. They look up
// the cache from context, build a key from the build version, variants, route
// metadata, format, and the provided parts, and short-circuit rendering when a
// cached body is found. CacheKey scopes the provided parts after the route
// metadata. CacheKeyF treats the provided parts as the full key after the build
// and variant scope.
//
// # Formats and content negotiation
//
// FormatFromRequest resolves Turbo Frame headers first, then an explicit
// WithFormat context value, then Accept. FormatFromSuffix and
// FormatFromContentType expose the global registry used by lazyroutes and custom
// integrations. NewFormat registers a MIME type and route suffix for application
// formats beyond HTML, JSON, Turbo Stream, common images, and SSE.
//
// Wants and Respond run exactly one handler from a Formats map based on the
// negotiated request format. They set Vary headers for Accept and Turbo-Frame,
// return a 406 HTTPError when no offered format matches, and temporarily make
// Base.Format report the selected format while the handler runs.
//
// # Deferred view values
//
// SetLater and SetWhenNeeded store Valuer values in the view data map. A Valuer
// exposes Value() (any, error), which lazyview template engines can resolve when
// the template actually needs the value. SetLater starts the loader immediately
// in a goroutine, useful for overlapping slow data loading with other controller
// work. SetWhenNeeded waits until the first Value call. Loaders must be
// functions returning (value, error) and may accept context.Context; that
// context is the current request context layered with the application context.
//
// # Errors and development hooks
//
// Error wraps an error with an HTTP status, StatusCode reads that status, and
// PanicError converts a recovered panic into an error carrying a backtrace.
// ErrorHandler is middleware used by lazyapp to catch reported action errors and
// panics. If the controller implements HandleError(http.ResponseWriter,
// *http.Request, error), that method gets the first chance to render the error.
// Otherwise ErrorHandler writes development details when DetailErrors is enabled
// or falls back to static error pages registered with WithErrorPages.
//
// RegisterLazyDevHandlers exists only in files built with the lazydev build tag.
// lazyapp registers those handlers on its lazycontrolplane.ControlPlane during
// development builds. The current handler exposes the editor-opening endpoint
// used by detailed error pages; production builds do not include that endpoint.
package lazycontroller
