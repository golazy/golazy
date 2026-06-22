// Package lazyfiles catalogs stored files and routes them to named storages.
//
// lazyfiles owns file metadata, storage locations, fallback application URLs,
// and migration-friendly indirection. Byte-level storage is delegated to
// lazystorage implementations.
//
// Applications can use it as the durable file catalog behind uploads, imported
// assets, or generated media. lazymedia can then build representations on top
// of the catalog without taking ownership of storage itself.
package lazyfiles
