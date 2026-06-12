# Changelog

All notable changes to GoLazy are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and GoLazy uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/golazy/golazy/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/golazy/golazy/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/golazy/golazy/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/golazy/golazy/releases/tag/v0.1.0
