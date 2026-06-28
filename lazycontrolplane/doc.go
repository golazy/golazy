// Package lazycontrolplane provides operational HTTP endpoints for GoLazy
// applications.
//
// A control plane exposes framework-owned routes such as liveness and readiness
// probes outside the application's route table. The zero Config is useful: it
// creates a control plane with /livez and /readyz. Applications can pass
// Config to lazyapp.Config.ControlPlane to mount those endpoints with the app,
// or instantiate a ControlPlane directly when they want to serve it manually.
// Use Config.Pprof or ControlPlane.EnablePprof to attach the standard
// net/http/pprof handlers.
package lazycontrolplane
