// Package lazyassets registers, fingerprints, serves, and uploads application
// assets.
//
// In a GoLazy application, lazyapp registers Public files and generated asset
// sources into one Registry, installs view helpers such as asset_path and
// stylesheet, and serves registered files as the final public fallback.
//
// The package is also independently usable for static file serving or deploy
// preparation. Create a Registry, add filesystem or generated assets, serve it
// with Handler, or export permanent assets through Upload and a lazystorage
// writer.
package lazyassets
