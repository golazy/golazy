// Package lazycontroller provides request-local controller state, rendering
// helpers, response helpers, cache-key helpers, typed HTTP errors, redirects,
// content negotiation, SEO convenience methods, and Server-Sent Events
// entrypoints.
//
// In a normal GoLazy app, lazyapp creates a renderer, stores it in context, and
// lazyroutes binds a fresh controller instance to each request. Controllers then
// embed Base, call Set before rendering, return errors for framework error
// handling, and let empty successful actions render the matching view
// automatically.
//
// The package can also be used without lazyapp. Create a Renderer from an
// fs.FS, put it in context with WithRenderer, construct Base with NewBase, and
// bind each request with BindRequest before rendering.
package lazycontroller
