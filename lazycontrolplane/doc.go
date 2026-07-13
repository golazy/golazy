// Package lazycontrolplane provides operational HTTP endpoints for GoLazy
// applications.
//
// A control plane owns framework and operations routes that should not be part
// of the application's route table. The zero Config is useful: it creates
// GET /livez, GET /readyz, and the empty developer-panel discovery document at
// GET /_golazy/panels. /livez reports that the process can answer HTTP requests.
// /readyz runs configured ReadinessCheck functions and returns 503 Service
// Unavailable when any dependency or runtime state says the app is not ready to
// receive traffic.
//
// The package can be used directly with net/http, but most applications pass a
// Config or *ControlPlane to lazyapp.Config.ControlPlane. lazyapp builds the
// control plane, adds package-owned endpoints for configured jobs and telemetry,
// and, when built with the lazydev tag, registers development control endpoints
// from packages such as lazyassets, lazybuildinfo, lazycache, lazycontroller,
// lazydeps, lazyjobs, lazyroutes, and lazytelemetry.
//
// lazyapp decides where the plane is served. In production builds it does not
// intercept application requests unless CONTROL_PLANE_ADDR is set to the same
// listen address as the app. When CONTROL_PLANE_ADDR points at a different
// address, lazyapp serves ControlPlane.StandaloneHandler on that listener; the
// standalone handler adds a small root index and keeps application "/" routes
// separate. In lazydev builds, lazyapp keeps the control plane available on the
// app handler so the development panel can call it.
//
// Owned operational endpoints, including endpoints contributed by add-ons, can
// be registered with ControlPlane.Register. Registration records a stable ID,
// owner, pattern, and description for duplicate detection and the standalone
// endpoint index. RegisterReadinessCheck provides the matching error-returning
// readiness API. Registrations are concurrency-safe, including while an early
// startup control plane is serving, and are closed explicitly with
// ControlPlane.Seal after application setup is complete.
//
// ControlPlane.Handle and ControlPlane.AddReadinessCheck remain available as
// panic-on-error compatibility APIs. Use Config.Metrics or a custom handler for
// /metrics, and use Config.Pprof or ControlPlane.EnablePprof to attach the
// standard net/http/pprof handlers.
package lazycontrolplane
