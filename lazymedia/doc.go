// Package lazymedia manages generated representations of stored files.
//
// lazymedia does not own byte storage. It depends on a small FileStore
// interface so applications can back it with lazyfiles or another file service.
//
// Use it for application-level derivatives such as thumbnails, previews, or
// converted media files. Keep original file metadata and storage backends in
// lazyfiles and lazystorage, then let lazymedia coordinate representation
// lookup and generation.
package lazymedia
