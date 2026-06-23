// Package lazydeps records application dependency initialization and the
// dependency graph between services.
//
// Applications receive a *Scope from lazyapp.Config.Dependencies and initialize
// shared services with Service. Service returns a typed Ref; calling Ref.Use
// inside another service initializer records that dependency edge and returns
// the wrapped service value.
package lazydeps
