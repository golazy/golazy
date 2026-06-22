// Package lazysse writes Server-Sent Events responses.
//
// Controllers usually start streams through lazycontroller.Base.SSEStream so
// GoLazy can mark the response as sent, bypass automatic rendering, and avoid
// dynamic response buffering. Use this package directly when an ordinary
// http.Handler needs the same event formatting and flush behavior without a
// controller.
package lazysse
