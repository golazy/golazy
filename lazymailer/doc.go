// Package lazymailer renders mailer views into standard-library MIME messages
// and sends them through pluggable delivery implementations.
//
// The package has a deliberate boundary between rendering and delivery. Base
// holds template variables with Set, Build renders a Message, and Mail calls
// Build and then passes the message to a named Delivery in a Registry. Build is
// useful when tests need to inspect the rendered text, HTML, headers, or
// recipients before anything is sent. Mail is the normal application path when
// a message should cross the delivery boundary.
//
// Mailer templates are rendered by lazyview through the renderer stored in
// context by lazycontroller. The controller name passed to NewBase selects the
// view directory, and Options.Action selects the template basename. For example,
// NewBase(ctx, "notice_mailer", defaults) with Action "welcome" looks for
// notice_mailer/welcome.text.* and notice_mailer/welcome.html.* templates. Text
// and HTML are rendered independently; one format may be missing, but Build
// returns an error when neither format exists. Layout defaults to "mailer", so
// mailer layouts usually live next to application layouts as
// layouts/mailer.text.* and layouts/mailer.html.*.
//
// In a conventional GoLazy application, lazyapp creates the shared lazyview
// renderer and stores it in the application context through lazycontroller.
// Application setup then creates a Mailer with New, stores it with WithContext,
// and constructs app-specific mailer types that embed or hold Base. The same
// embedded view filesystem, template engine registration, and helpers used by
// lazycontroller views are therefore available to mailer views. lazyapp does
// not choose an SMTP server or test delivery by itself; applications keep that
// configuration at their dependency or context boundary and pass a Registry to
// New.
//
// The package is also usable without lazyapp. Create a lazycontroller renderer
// from an fs.FS, put it in context with lazycontroller.WithRenderer, import a
// lazyview engine such as golazy.dev/lazyview/gotmpl, and then use New,
// WithContext, and NewBase exactly as an application would.
//
// Delivery is intentionally small. SMTPDelivery sends the bytes returned by
// Message.Bytes with net/smtp. MemoryDelivery records Message values for tests
// and development. DeliveryFunc adapts a function when an application already
// has another mail transport, queue, or provider client.
package lazymailer
