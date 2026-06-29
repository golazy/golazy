// Package lazydeps initializes application services, records how they depend on
// each other, and shuts them down in dependency-safe order.
//
// A Scope is the dependency container for one application process. It is not a
// service locator for request handlers; it is used during application startup to
// create long-lived services, attach those services to the application context,
// and remember the graph that was observed while each initializer ran. When the
// application shuts down, dependents are stopped before the services they use.
//
// Service runs one initializer in the scope and returns a typed Ref. The Ref is
// how another initializer says that it needs the service. Calling Ref.Use inside
// another Service initializer returns the wrapped value and records an edge from
// the current service to the referenced service. Calling Use after startup is a
// programming error because lazydeps would no longer know which service is
// declaring the dependency.
//
// The common GoLazy path is through lazyapp.Config.Dependencies. lazyapp creates
// the Scope, passes it to the application's Dependencies function, stores the
// final Scope on lazyapp.App.Dependencies, and continues building routes,
// renderers, jobs, SEO, and other application systems with the context returned
// by Scope.Context. Initializers normally return a derived context containing
// typed context values for packages such as lazycontroller handlers, lazyjobs
// configuration, or application services that run outside request handling.
//
// lazydeps also has development-only control-plane handlers. In lazydev builds,
// lazyapp registers them on the lazycontrolplane.ControlPlane, so the
// development panel can inspect GET /dependencies and run the shutdown
// simulation endpoints. Applications usually should not call
// RegisterLazyDevHandlers directly unless they are assembling a custom
// control-plane outside lazyapp.
//
// The package can also be used without lazyapp when a small server or worker
// wants typed startup services plus ordered cleanup. Create a Scope with New,
// call Service for each long-lived dependency, use Ref.Use inside dependent
// initializers, then call Scope.Shutdown when the process is draining.
package lazydeps
