// Package lazyassets registers, fingerprints, serves, and exports application
// assets.
//
// In a GoLazy application, lazyapp registers Public files and generated asset
// sources into one Registry, installs view helpers such as asset_path and
// stylesheet, and serves registered files as the final public fallback.
// lazyapp passes Registry.Helpers to lazyview so templates can render the
// final asset URLs without naming lazyassets directly.
//
// The package is also independently usable for static file serving or deploy
// preparation:
//
//	registry := lazyassets.New()
//	err := registry.AddFS(os.DirFS("public"))
//	if err != nil {
//		return err
//	}
//	http.Handle("/", registry.Handler(nil))
//
// Registered assets keep a logical path, such as /styles.css, and usually get a
// permanent content-hashed path, such as /styles-2c26b46b68ff.css. Handler
// serves both paths. View helpers return the permanent path when one exists, so
// browser caches can keep immutable assets while templates still use stable
// source names.
//
// A Registry builds its manifest as assets are added. Callers do not create a
// manifest first: Add, AddReader, and AddFS compute hashes, ETags, integrity
// values, permanent paths, and CSS URL rewrites automatically. CSS url(...)
// references that point at other registered local assets are rewritten to the
// target asset's permanent path; remote URLs, data URLs, fragments, and missing
// assets are left unchanged.
//
// For deployment, Upload writes registered files through lazystorage and Unpack
// writes them to a local directory. Both can choose logical paths, permanent
// paths, or both.
package lazyassets
