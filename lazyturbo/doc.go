// Package lazyturbo provides Hotwire Turbo helpers for controllers and views.
//
// Turbo is the Hotwire browser protocol for replacing parts of a page without
// writing custom JavaScript for each interaction. A Turbo Frame is a named
// region of HTML, rendered as a <turbo-frame id="..."> element, that the
// browser can request and replace independently. A Turbo Stream response is an
// HTML response with media type text/vnd.turbo-stream.html that asks the
// browser to append, replace, remove, or otherwise mutate page elements.
//
// The package has two jobs. First, it exposes request helpers such as
// FrameID, IsFrameRequest, IsPrefetch, and AcceptsStream so higher-level
// packages can recognize Turbo requests. lazycontroller uses those helpers
// during format negotiation: a request with a Turbo-Frame header becomes the
// lazycontroller.TurboFrame format, while an Accept header that includes
// StreamMIME becomes lazycontroller.TurboStream.
//
// Second, it exposes lazyview helpers through Helpers. lazyapp installs those
// helpers automatically on its renderer, after route, asset, form, and SEO
// helpers and before application Config.Helpers. In a normal GoLazy app, views
// can call turbo_frame directly without importing this package. Direct use is
// for standalone lazyview renderers or lower-level apps that want the same
// helpers without constructing a lazyapp.App.
//
// The turbo_frame helper renders a partial named "_<id>_frame.html.tpl" and
// wraps the rendered body in a <turbo-frame>. Frame ids are intentionally
// restricted by ValidateFrameID because the same value becomes both a DOM id
// and a partial name. FrameTag is the lower-level wrapper when the body was
// already rendered by another package.
//
// lazycontroller connects this package to controller rendering. Base.Render
// sees Turbo-Frame requests and renders the matching "_<id>_frame" partial
// without the layout. Base.RenderTurboFrame renders a specific frame even when
// the current request was not negotiated from Turbo headers, and
// Base.SetTurboFrameOptions passes lazyturbo.FrameOption values such as Src,
// Loading, Action, and RefreshMorph into the generated frame tag.
package lazyturbo
