// Package lazypwa makes GoLazy applications installable as progressive web
// apps.
//
// The package owns web app manifest rendering, a small browser client,
// version-aware service-worker generation, opt-in offline cache manifests, and
// push-notification contracts. It uses lazyworkers for service-worker
// registration so applications can register additional workers without making
// PWA the only worker entrypoint.
//
// Conventional applications configure this package through lazyapp.Config.PWA.
// Direct users can create an App with New, register its worker in a
// lazyworkers.Registry, install its Helpers into a renderer, and mount its
// Handler in any net/http stack.
package lazypwa
