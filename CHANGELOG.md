# Changelog

All notable changes to GoLazy are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and GoLazy uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.4] - 2026-06-15

### Added

- `golazy.dev/lazyview` for framework-owned view rendering, helper
  registration, request-scoped helper data, and pluggable template engines.
- `golazy.dev/lazyview/gotmpl` to register the standard `html/template`
  engine.
- `golazy.dev/lazysupport/inflection` with pluralization, singularization,
  camelization, underscore, dasherize, titleize, tableize, classify,
  parameterize, and humanize helpers.
- Route helper registration from `lazyroutes.Scope`, exposing named-route
  helpers to view engines.

### Changed

- `lazycontroller` now renders through `lazyview` and no longer owns template
  parsing directly.
- `lazyapp` resolves views from the configured `Views` filesystem, allowing
  development builds to use local disk views while production builds keep the
  embedded single-binary behavior.
- Resource route naming now uses the shared inflection package.

## [0.1.3] - 2026-06-15

### Added

- `golazy.dev/lazyapp` to assemble application context, views, routes,
  dispatcher middleware, and public files into one `http.Handler`.
- `golazy.dev/lazydispatch` with middleware chaining, router dispatch,
  embedded public-file fallback, final `404 Not Found`, and static-file
  `405 Method Not Allowed` handling.
- Scope-based route DSL methods on `lazyroutes.Scope`, including HTTP verb
  methods, `Resources`, `Namespace`, `Path`, and `As`.
- Route table metadata with automatic route names, controller/action names,
  namespaces, and named route params.
- Request-context route metadata through `lazyroutes.RouteFromContext` and
  `lazyroutes.RouteFromRequest`.
- `printroutes` build-tag support that writes the route table as JSONL after
  application routes are drawn.

### Changed

- Application route drawers now receive `Draw(router *lazyroutes.Scope)`.
- Controller action binding is handled internally by `lazyroutes`; applications
  no longer call a public bind helper in routes.
- `lazyapp.New` now takes `lazyapp.Config` and returns `*lazyapp.App`.
- Root route metadata stores the user-facing path `/` while the router still
  registers the exact root pattern internally.

## [0.1.2] - 2026-06-12

### Added

- REST-style resource routing in `golazy.dev/lazyroutes`.
- Controller action binding in `golazy.dev/lazyroutes`.

### Removed

- Controller action binding from `golazy.dev/lazycontroller`.

## [0.1.1] - 2026-06-12

### Changed

- Renamed `golazy.dev/controller` to `golazy.dev/lazycontroller`.
- Renamed `golazy.dev/routes` to `golazy.dev/lazyroutes`.

## [0.1.0] - 2026-06-12

### Added

- Controller rendering with layouts, view data, and typed HTTP errors.
- Request-local controller action binding.
- Route construction with embedded public-file fallback.
- Method-not-allowed handling for application routes.

[Unreleased]: https://github.com/golazy/golazy/compare/v0.1.4...HEAD
[0.1.4]: https://github.com/golazy/golazy/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/golazy/golazy/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/golazy/golazy/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/golazy/golazy/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/golazy/golazy/releases/tag/v0.1.0
