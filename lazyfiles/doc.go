// Package lazyfiles catalogs stored files and routes them to named storages.
//
// lazyfiles owns file metadata, storage locations, fallback application URLs,
// and migration-friendly indirection. Byte-level storage is delegated to
// lazystorage implementations.
//
// Applications can use it as the durable file catalog behind uploads, imported
// assets, or generated media. lazymedia can then build representations on top
// of the catalog without taking ownership of storage itself.
//
// The append-only JSONL repository implementation lives in the
// golazy.dev/lazyfiles/jsonl subpackage.
package lazyfiles
