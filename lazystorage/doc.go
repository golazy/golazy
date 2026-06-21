// Package lazystorage defines small interfaces for object-style storage.
//
// The package follows the standard library io/fs pattern: Storage is the small
// read capability, and optional interfaces add writes, deletes, listing, URLs,
// and watching. Implementations consume the options they recognize and return
// the remaining options to callers that compose multiple storage layers.
package lazystorage
