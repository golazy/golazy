// Package lazysse writes Server-Sent Events responses.
//
// Server-Sent Events, often called SSE, are long-lived HTTP responses that let
// a server push ordered text frames to a browser EventSource or another client
// that understands the text/event-stream format. A lazysse Stream sets the
// required response headers, commits the response, writes each event frame, and
// flushes after every send. Event data can be written directly with Send, as
// JSON with JSON, as comments with Comment, or by forwarding a Subscription
// from Subscribe.
//
// Streams require an http.ResponseWriter that can flush. Start also recognizes
// buffered GoLazy response writers that expose StartStream or Unwrap, so a
// stream can bypass response buffering after headers are committed. That is the
// path used by lazycontroller.Base.SSEStream: controllers call that helper, the
// controller marks the response as sent, and lazydispatch does not try to render
// a template, buffer the body, or calculate a dynamic ETag for the stream.
//
// Use this package directly outside a GoLazy app when a plain net/http handler
// needs the same SSE formatting and flush behavior. The package does not own an
// event queue; callers either send events themselves or provide a Source whose
// Subscription yields lazysse.Event values. LastEventID reads the browser's
// Last-Event-ID header so a Source can resume after a reconnect.
package lazysse
