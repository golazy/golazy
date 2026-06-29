// Package lazyfiles catalogs stored files and routes file IDs to named
// lazystorage backends.
//
// The package separates file metadata from object bytes. A File is the durable
// catalog record for one logical upload or generated file: its ID, display
// filename, content type, size, checksum, opaque application metadata, and
// timestamps. A Location says where the bytes for that file currently live:
// the named storage backend, object key, role, status, and location checksum.
// Files coordinates the two pieces by writing bytes through a lazystorage.Writer
// and then recording the resulting File and primary active Location in a
// Repository.
//
// This indirection is useful when an application needs stable file IDs while
// moving bytes between storage systems, mirroring objects, or replacing a
// legacy backend. The catalog can keep multiple locations for a file; lookup
// chooses a primary active location first, then any active location, then an
// unmarked location. Byte-level storage remains owned by lazystorage
// implementations such as lazystorage.NewFilesystem or lazystorage/s3.
//
// URL asks the chosen storage for a direct URL when that storage implements
// lazystorage.URLer. If the storage cannot provide one, URL returns an
// application route under RoutePrefix, defaulting to "/_lazy/files". Handler
// serves those fallback routes for GET and HEAD requests by verifying the path
// token, opening the cataloged file, and streaming the stored bytes. When
// SigningKey is set, fallback route tokens are HMAC-signed and can honor
// lazystorage.ExpiresIn or lazystorage.ExpiresAt; without a signing key the raw
// file ID is used as the token.
//
// lazyfiles is not tied to a GoLazy application. Use it directly when a service
// needs a small catalog around one or more object stores. In a larger GoLazy
// stack, lazymedia can build derived media records on top of lazyfiles without
// owning original storage. PostgreSQL-backed applications can pair this package
// with golazy.dev/pg/pgfiles for catalog metadata and golazy.dev/pg/pgstorage
// for object bytes; golazy.dev/pg/pgmedia provides the matching repository for
// lazymedia derivatives.
//
// The append-only JSONL repository implementation lives in the
// golazy.dev/lazyfiles/jsonl subpackage. It is useful for local tools, tests,
// and simple single-process applications. Production applications that need
// shared concurrency or database transactions should use a database-backed
// Repository such as golazy.dev/pg/pgfiles.
package lazyfiles
