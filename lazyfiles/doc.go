// Package lazyfiles catalogs stored files and routes them to named storages.
//
// lazyfiles owns file metadata, storage locations, fallback application URLs,
// and migration-friendly indirection. Byte-level storage is delegated to
// lazystorage implementations.
package lazyfiles
