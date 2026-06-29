// Package lazystorage defines small interfaces for object-style storage.
//
// Storage is the minimum read contract: Open receives an object key and returns
// a File that can be read, closed, and inspected with Stat. Optional interfaces
// add capabilities one at a time. Writer stores bytes with Put, Deleter removes
// a key, Lister returns object metadata below a prefix, URLer resolves an
// externally usable URL, and Watcher streams backend change events. Callers
// should test for the optional interface they need instead of assuming every
// backend can write, list, delete, or sign URLs.
//
// Object keys are slash-separated logical paths, not host filesystem paths.
// ValidateKey applies io/fs path rules: keys must be relative, clean, and must
// not be ".". The local filesystem backend maps those keys below its configured
// root; S3-compatible and PostgreSQL backends store the same logical key in the
// remote bucket or database row. Prefixes passed to List and Watch use the same
// logical form and are matched by key prefix.
//
// The variadic options are intentionally open-ended so storage can be composed
// with higher-level packages. An implementation consumes the options it knows
// and returns the remaining options to its caller. For example, filesystem Put
// consumes ContentType and leaves unknown options untouched, while
// lazystorage/s3 and pg/pgstorage also consume CacheControl and
// ContentDisposition. URL options such as ExpiresIn, ExpiresAt, Public,
// Private, and DownloadName are requests; a backend may use them, pass them
// through, or reject the operation when it cannot provide the requested URL.
// Use Take when writing decorators or services that need this same
// consume-and-forward behavior.
//
// Info is metadata about the stored object. Key is always the logical key.
// ContentType, Size, Checksum, and ModifiedAt are best-effort backend metadata:
// filesystem writes compute a sha256 checksum and detect content type when not
// supplied, listing filesystem objects reports file size and modification time,
// and remote backends expose whatever their implementation can read cheaply.
// Metadata is reserved for backend-specific values that do not deserve their
// own portable field.
//
// Higher-level packages depend on these narrow contracts. lazyassets.Registry
// uploads generated assets through a Writer, forwarding content type and cache
// policy for permanent hashed files and the manifest. lazyfiles stores file
// records separately from bytes and writes/opens those bytes through configured
// lazystorage backends; when a backend implements URLer, lazyfiles can return
// that direct URL, otherwise it falls back to application routes. lazymedia
// stores original media and generated variants through lazyfiles and
// lazystorage-backed file stores.
//
// Use NewFilesystem for standalone local storage or development storage.
// Package lazystorage/s3 provides an S3-compatible backend for object stores
// such as AWS S3, MinIO, or SeaweedFS. The pg/pgstorage package in the
// golazy.dev/pg module provides a PostgreSQL implementation and migrations when
// keeping object bytes in the application database is the right tradeoff.
package lazystorage
