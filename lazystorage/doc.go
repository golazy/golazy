// Package lazystorage defines small interfaces for object-style storage.
//
// The package follows the standard library io/fs pattern: Storage is the small
// read capability, and optional interfaces add writes, deletes, listing, URLs,
// and watching. Implementations consume the options they recognize and return
// the remaining options to callers that compose multiple storage layers.
//
// Higher-level packages such as lazyassets, lazyfiles, and lazymedia depend on
// these narrow interfaces instead of a concrete backend. Use the package
// directly for local filesystem storage, or use backend subpackages such as
// lazystorage/s3 when an application needs object storage.
package lazystorage
