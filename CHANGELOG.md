# Changelog

All notable changes to GoLazy are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and GoLazy uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Lazydev control-plane endpoints can enable or disable detailed request
  monitoring, which records per-request Go runtime traces, span JSON, and
  request-local log JSONL under `.tmp/traces`.
- Lazydev request span JSON now includes per-region total and self duration
  plus sampled allocation bytes, malloc counts, and free counts.
- Lazydev request span JSON now includes a goroutine id for each region so
  development tooling can highlight regions that cross goroutine boundaries.
- Framework telemetry records child regions for dispatched middleware, routing,
  dispatch, controller setup, action calls, view rendering, layouts, and
  partials when a request span is active. Request ids are attached to request
  spans, child regions, and log span-events.
- `golazy.dev/lazydoc` now records source file and line metadata for packages,
  values, functions, types, and methods so rendered package docs can link
  directly to repository source.
- `golazy.dev/lazyjobs`, a background job runner with typed JSON job payloads,
  an in-memory backend, retries, `lazyapp.Config.Jobs` wiring, context access,
  and read-only job state on the control plane.
- Separate control-plane listeners now serve a compact endpoint index at
  `GET /`, while same-listener control planes leave the application root route
  untouched.
- `golazy.dev/lazymigrate`, a backend-agnostic migration loader and planner
  with source/backend diffing, schema dump/load hooks, and a fake test backend.
- `golazy.dev/lazysupport/inflection.Irregular` for registering
  application-specific singular/plural pairs before conventional resource names
  are derived.
- Lazydev control-plane endpoints now expose the lazy asset manifest at
  `GET /assets` and can clear request trace sidecars with
  `POST /requests/traces/clear`.
- Lazydev control-plane endpoints now expose cache hit, miss, set, and toggle
  events through `GET /cache/events`.

### Changed

- `lazyapp.ListenAndServe` now automatically mounts standard pprof handlers on
  the control plane when `CONTROL_PLANE_ADDR` runs on a different listener from
  the application.
- Lazydev request trace capture is off by default and no longer depends on OTEL
  exporter environment variables. `OTEL_SDK_DISABLED=true` remains the hard
  disable for lazydev request telemetry.
- Lazydev request trace snapshots now support path and category filtering,
  include the middleware that handled each request, and classify requests as
  framework, asset, or other traffic.
- Lazydev middleware tracing records whether each middleware called the next
  handler, allowing the panel to identify the middleware that handled a
  request.
- `lazycontroller.Base.Layout` is now the layout-selection API. The older
  `SetLayout` alias was removed.
- `lazycontroller.Base.CacheKey` and `CacheKeyF` now return `true` when an
  existing cached response was written and `false` when the action should
  continue and render a body for storage.
- Controller and template render cache keys now include the app build version,
  and include the active render variant when present, so deploys and
  variant-specific renders do not share stale cached bodies.
- `lazycache.Cache.Subscribe` now provides a non-blocking development event
  hook for cache hits, misses, sets, and on/off toggles. Inspectable cache
  entries can include per-key counters, and the default in-memory backend
  records per-key hits and sets.
- `lazyfiles` and `lazymedia` now keep their append-only JSONL repository
  implementations in `golazy.dev/lazyfiles/jsonl` and
  `golazy.dev/lazymedia/jsonl`, with `JSONLRepository` types that satisfy the
  parent package `Repository` interfaces.
- `lazydeps` now logs dependency shutdown reasons, context cancellation, and
  cleanup duration for every registered service.
- `lazyapp.New` now initializes optional background jobs after dependencies and
  exposes `App.Jobs` when `lazyapp.Config.Jobs` is configured.

## [0.1.16] - 2026-06-27

### Added

- `golazy.dev/lazycache`, a cache contract with typed `Get` / `Set` helpers,
  in-memory storage, stats, runtime On/Off switching, and controller, partial,
  and Turbo frame caching helpers wired through `lazyapp.App`.
- `lazycontroller.Base.SetLater` and `SetWhenNeeded` for request-local deferred
  view data that templates resolve only when `.Value` is read.
- Template route helpers `link_to`, `attr`, `data`, and `unless_current`, so
  views can render escaped anchors around `path_for` destinations and omit the
  anchor for the current page.
- OpenTelemetry-backed request telemetry, including OTEL environment config,
  request IDs, trace spans, context-aware logs, in-memory metrics, and
  Prometheus metrics on the control plane when configured.

### Changed

- `lazydev` control-plane handlers are now owned by their framework packages,
  and `lazyapp` aggregates them for development builds.
- `lazyapp` includes cache statistics in its Prometheus scrape output when
  telemetry metrics are enabled.

## [0.1.15] - 2026-06-25

### Added

- `golazy.dev/lazydeps`, a dependency-initialization helper that records service
  nodes and dependency edges while returning typed service references.
- `golazy.dev/lazyconfig`, a small environment-backed configuration loader with
  `Getenv[T]`, `MustGetenv[T]`, default field-to-env naming, struct tags for
  defaults and required values, slice entry loading, and optional
  `Validate() error` support.
- Framework-owned default controller views for `layouts/app.html.tpl` and
  `app/error.html.tpl`, used by `lazyapp` when an application does not provide
  its own files at those paths.
- `golazy.dev/lazyerrors`, an application error helper that prefixes errors
  with the caller, preserves `%w` wrapping, and records typed backtrace frames.

### Changed

- `lazyapp.Config.Dependencies` replaces `lazyapp.Config.Context`. The
  framework now creates a `*lazydeps.Scope`, passes it to the application
  dependency initializer, and keeps the resulting scope on `lazyapp.App`.
- `lazyapp.Config.SEO` is now a `func(context.Context) []lazyseo.Option`, so
  application SEO defaults are initialized after dependencies and can read
  dependency-backed values from the app context.
- `lazyapp` no longer serves an empty `/sitemap.xml` by default. Sitemap
  generation is enabled when `lazyapp.Config.Sitemap` has a base URL, URLs, or
  sources; `robots.txt` only advertises the generated sitemap when it exists.
- `lazyapp` now reads its framework-owned runtime environment variables,
  including `ADDR`, `PORT`, and `CONTROL_PLANE_ADDR`, through a local config
  struct backed by `lazyconfig`; the default application listen address is
  `127.0.0.1:3000`.
- In `lazydev` builds, `lazyapp` reads views and public files from build-time
  paths, and `lazyassets` serves logical development asset paths without
  permanent hashes or cache headers.
- `lazycontroller.Base.HandleError` now renders the shared `app/error` view
  before falling back to explicit static status files, and only exposes raw
  error details plus `lazyerrors` and recovered panic backtraces to that view
  when detail errors are enabled. The default error view shortens frame paths
  for display and, in `lazydev`, can open clicked frames in `$EDITOR` through
  `/_golazy/open-editor`, using VS Code's `-g file:line` form or a discovered
  terminal for terminal editors.
- Retracted pre-`v0.1.0` framework module versions, which were unstable
  snapshots before the current GoLazy release line.

## [0.1.14] - 2026-06-23

### Added

- `golazy.dev/lazytui/progress`, a small terminal progress package for running
  named tasks with compact status output, captured command output, warning
  results, mise command helpers, and temporary UI takeovers for interactive
  steps.
- SEO metadata can now include image alt text and article published time
  through `lazyseo.ImageAlt`, `lazyseo.PublishedTime`,
  `lazycontroller.Base.SEOImageAlt`, `lazycontroller.Base.PublishedTime`, and
  matching model metadata interfaces.

### Changed

- `lazyapp.Config.Context` now supports the preferred
  `func(context.Context) (context.Context, error)` initializer shape so
  application dependency setup can fail explicitly during startup. The previous
  `func(context.Context) context.Context` shape remains accepted for existing
  applications.
- SEO rendering now emits richer social image metadata, including secure image
  URLs, image dimensions, image alt text, and `article:published_time`, and it
  avoids appending the site name when a complete title is already present.

## [0.1.13] - 2026-06-22

### Added

- `golazy.dev/lazycontrolplane`, a framework-owned control plane with liveness
  and readiness probes, optional metrics and pprof handlers, and standalone
  HTTP serving support.
- `lazyapp.Config.ControlPlane` and `CONTROL_PLANE_ADDR` integration for
  serving the control plane either on the application handler or on a separate
  HTTP server.

### Changed

- Expanded package documentation and runnable examples across the core
  framework packages, including `lazyapp`, `lazycontroller`, `lazyroutes`,
  `lazymailer`, `lazypath`, assets, dispatch, docs, files, forms, media, SEO,
  SSE, storage, tests, Turbo, and view helpers.

## [0.1.12] - 2026-06-22

### Added

- `lazyassets.Registry.Upload`, which writes registered assets to any
  `lazystorage.Writer`. The default mode uploads content-hashed permanent asset
  paths plus `manifest.json`, which fits CDN and static ingress deployments.
- `lazyassets.WithBaseURL`, which makes asset helpers and importmap fragments
  emit absolute asset URLs while keeping registered asset paths and routing
  path-based.
- `golazy.dev/lazystorage/s3`, a concrete S3-compatible storage backend with
  signed Open, Put, Delete, List, URL, Watch polling, and bucket creation
  support.
- `golazy.dev/lazydoc`, shared package documentation loading, extraction,
  JSON, and search models for the GoLazy website and `lazy docs`.

### Changed

- In `lazydev` builds, `lazyapp` reads the local view root from a path supplied
  by `lazy`. Production builds continue to use the application configured
  embedded view filesystem.

### Removed

- Removed the public `golazy.dev/lazyviews` package. The `lazy` CLI now owns
  local development view-path resolution and passes the resolved path to
  `lazyapp` in `lazydev` builds.

## [0.1.11] - 2026-06-21

### Added

- `golazy.dev/lazystorage`, `golazy.dev/lazyfiles`, and
  `golazy.dev/lazymedia`, early storage building blocks for object-style
  storage backends, logical file catalogs, fallback signed file URLs, and
  generated media variants.
- `golazy.dev/lazyseo`, an optional helper package for request-local document
  metadata with `{{seo}}` and `{{seo_lang}}` view helpers for title, document
  language, canonical URLs, alternates, JSON-LD, description, Open Graph,
  Twitter card, and related tags. Common schema.org values live in
  `golazy.dev/lazyseo/jsonld`.
- Controller SEO helpers on `lazycontroller.Base`, including `Metadata`,
  `Title`, `Description`, `Language`, `Canonical`, `Alternate`, `OpenGraph`,
  `TwitterCard`, `JSONLD`, `SEOImage`, `Kind`, `Type`, `SchemaType`, `Locale`,
  and `TwitterCardType`, plus shared `lazyseo.PageKind` constants for paired
  Open Graph and schema.org names.
- Rails-style view variants such as `show.svg+square.tpl`, tried before the
  non-variant template for the same format.
- `lazycontroller.Base.RenderSVGString`, which renders SVG templates with
  optional variants for social-image generation workflows.
- Generated `/robots.txt` and `/sitemap.xml` through `lazyapp.Config`, with
  permissive robots defaults, configurable crawler rules, sitemap entries and
  sources, alternate URLs, `<lastmod>`, and `Last-Modified` cache validation.
- `golazy.dev/lazymailer` for Rails-style mailer rendering, standard-library
  MIME message building, and pluggable delivery implementations.

## [0.1.10] - 2026-06-20

### Added

- `golazy.dev/lazytest`, a small HTTP testing helper for complete
  application-handler tests, named-route requests, body/header assertions, JSON
  decoding, and cookie-aware clients.
- `lazycontroller.Base.Decode`, which parses the current request form and
  decodes it with `lazyschema`, returning a bad-request HTTP error for invalid
  submissions.
- `lazycontroller.MustPathFor` and trailing `lazycontroller.URLParams` support
  for `PathFor`, so controllers can build named paths with query strings
  without manual URL assembly.
- Nested REST resources through `lazyroutes.Resource.Resources`, including
  nested path parameters and route names such as `comments_post`.
- `lazyapp.Config.ForceDetailErrors` for forcing detailed error responses
  outside lazy development builds when an application explicitly opts in.

### Changed

- Automatic controller rendering now keeps HTML as the default render format
  even when a Turbo Stream response is accepted, while explicit `Render` calls
  still honor the selected request format.
- Default session names derived from module-path app names now use the module
  basename, so `github.com/golazy/example` becomes `example_session`.
- Framework error handling logs unexpected errors to stderr and shows detailed
  responses in lazy development builds or when forced by app configuration.

## [0.1.9] - 2026-06-19

### Added

- Controller format negotiation with registered response formats, Accept
  handling, path suffixes such as `.html`, `.json`, and `.md`, and
  Turbo-frame-aware rendering.
- `lazyturbo` frame helpers and controller Turbo frame rendering support for
  server-rendered Hotwire interactions.
- Controller redirect helpers: `Redirect`, `RedirectTo`, `RedirectBackOrTo`,
  `RedirectBack`, and same-host `URLFrom` validation.
- Controller path helpers through `PathFor`, using the current application
  route table from controller actions.
- Controller response metadata helpers: `Status`, `Header`, `ContentType`,
  `Layout`, and `NoLayout`.
- `golazy.dev/lazysse` for Server-Sent Events, including event sends, JSON
  events, comments, heartbeats, source subscriptions, and `Last-Event-ID`
  access.
- `lazycontroller.Base.SSEStream` for starting SSE streams from controller
  actions.

### Changed

- Controller rendering is buffered before commit, so status, headers, layout
  selection, and render errors can be coordinated consistently.
- `lazyroutes.Scope` recognizes registered format suffixes and dispatches the
  underlying route with the requested format in request context.
- Long-lived streaming responses bypass dynamic route ETag generation and
  response-buffer rewrites.
- Cookie session middleware saves pending session changes before a streaming
  response is committed.
- The route action-call planner now lives under `lazyroutes/actioncall`.

## [0.1.8] - 2026-06-19

### Added

- Controller actions can receive route parameter arguments and generated
  request values from `GenX` controller methods. Generators may depend on
  standard action arguments and other generated values, and generated values
  are cached for the current request.
- `lazyroutes.Scope.Namespace` now prefixes route paths, route names, route
  metadata, resource routes, and root route names such as `admin_root`.

### Changed

- Namespaced controller routes render from matching nested view directories
  such as `views/admin/posts`, without falling back to the non-namespaced
  controller view.

## [0.1.7] - 2026-06-17

### Added

- `stylesheet` and `importmap` asset helpers for rendering fingerprinted
  stylesheet tags and inline import maps from registered assets.
- `golazy.dev/lazyviews`, used by `lazydev` builds to resolve local disk views
  from the running application tree.
- `golazy.dev/lazyschema`, adapted from Gorilla `schema`, for form field naming
  and bounded request-value decoding.
- `golazy.dev/lazyforms` with `form_for`, typed field helpers,
  `delete_button_for`, and automatic `lazyapp` helper registration.
- `lazydispatch/middlewares.MethodOverride` for route-scoped form `_method`
  handling and `CrossOriginProtection` for opt-in cross-origin request
  rejection.
- `lazyroutes.Resource.Model` to map model types to REST create, update, and
  delete routes.

### Changed

- `lazyapp` switches view loading to `lazyviews` when applications run with the
  `lazydev` build tag, keeping development view lookup in the framework instead
  of generated applications.
- `lazyview` partial rendering now accepts explicit render contexts, so helpers
  like `form_for` can render partials with a prepared model/form context.

## [0.1.6] - 2026-06-17

### Added

- `golazy.dev/lazycookie` with signed and encrypted secure-cookie support.
- `golazy.dev/lazysession` with cookie-backed session middleware and
  `lazyapp.Config.Sessions` integration.
- `lazyapp.App.ListenAndServe`, using `ADDR`, then `PORT`, then `:3000`.
- `lazyapp.MustSub` for wiring embedded `views` and `public` directories.
- Controller request lifecycle hooks for binding request state and running
  request-time setup through `BeforeAction`.
- `lazyview.Views.Cache` and template-engine cache hooks for precompiling
  templates after application helpers are registered.
- Controller action benchmarks covering direct writes and automatic rendering.

### Changed

- Controller constructors now run when routes are drawn. GoLazy keeps a
  prototype and uses pooled request instances, rebinding writer, request,
  route, and render state for each request.
- `lazyapp.ListenAndServe` installs `app.Context` as the server base context so
  request contexts inherit application dependencies.
- The `gotmpl` engine caches parsed templates and uses pooled executors so
  request-bound helpers still receive the current render context.
- `lazyapp.Helpers` makes helper registration less map-shaped in application
  configuration.

### Fixed

- Controller and route errors now route through framework error handling with
  support for static status pages and dynamic error templates.
- Automatic rendering is skipped when an action already called `Render`.
- Default session names derived from module-path application names are
  normalized into valid cookie names.

## [0.1.5] - 2026-06-16

### Added

- Automatic controller view rendering when an action returns without writing a
  response or calling `Render` explicitly.
- Route-scoped dynamic `ETag` responses for eligible `GET` and `HEAD`
  application routes, including `If-None-Match` handling.
- `golazy.dev/lazyassets` for registering filesystem and generated assets,
  content hashing, permanent asset URLs, ETags, integrity values, cache
  policies, and asset unpacking.
- Asset view helpers: `asset_path`, `asset_integrity`, and compatibility
  `permalink`.
- CSS `url(...)` rewriting so stylesheet references can point at permanent
  asset URLs.

### Changed

- `lazyapp` now serves configured public files through `lazyassets` after route
  lookup instead of the raw static-file middleware.
- Response buffering and dynamic route ETags are applied only to registered
  application routes; public assets keep their own asset-specific validator and
  cache policy.

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

[Unreleased]: https://github.com/golazy/golazy/compare/v0.1.16...HEAD
[0.1.16]: https://github.com/golazy/golazy/compare/v0.1.15...v0.1.16
[0.1.15]: https://github.com/golazy/golazy/compare/v0.1.14...v0.1.15
[0.1.14]: https://github.com/golazy/golazy/compare/v0.1.13...v0.1.14
[0.1.13]: https://github.com/golazy/golazy/compare/v0.1.12...v0.1.13
[0.1.12]: https://github.com/golazy/golazy/compare/v0.1.11...v0.1.12
[0.1.11]: https://github.com/golazy/golazy/compare/v0.1.10...v0.1.11
[0.1.10]: https://github.com/golazy/golazy/compare/v0.1.9...v0.1.10
[0.1.9]: https://github.com/golazy/golazy/compare/v0.1.8...v0.1.9
[0.1.8]: https://github.com/golazy/golazy/compare/v0.1.7...v0.1.8
[0.1.7]: https://github.com/golazy/golazy/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/golazy/golazy/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/golazy/golazy/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/golazy/golazy/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/golazy/golazy/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/golazy/golazy/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/golazy/golazy/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/golazy/golazy/releases/tag/v0.1.0
