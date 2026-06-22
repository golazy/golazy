// Package lazymailer renders Rails-style mailer views into standard-library
// MIME messages and sends them through pluggable delivery implementations.
//
// In a GoLazy application, create a Mailer from the application context after
// lazyapp has installed the shared renderer, store it in context with
// WithContext, and embed Base in app mailer types. Mailer views use the same
// lazyview engine, helpers, layouts, variants, and embedded filesystem as
// controller views.
//
// The package is also usable without lazyapp when a lazycontroller.Renderer is
// available in context. Delivery is intentionally small: SMTPDelivery sends via
// net/smtp, MemoryDelivery captures messages for tests, and DeliveryFunc adapts
// custom transports.
package lazymailer
