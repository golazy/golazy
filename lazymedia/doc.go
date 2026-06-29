// Package lazymedia manages generated representations of stored files.
//
// A media variant is a derived file, not the original upload. Examples include
// thumbnails, Open Graph images, compressed previews, transcript sidecars, or a
// browser-friendly conversion of an uploaded document. The source file and the
// generated output are both regular files identified by file IDs; lazymedia only
// records that source file X has variant key Y whose output file is Z.
//
// The package deliberately does not own byte storage or the durable catalog for
// original files. Media uses FileStore to open a source file, save a generated
// output file, and return a URL for that output. Applications commonly satisfy
// FileStore with golazy.dev/lazyfiles, which in turn stores object bytes through
// golazy.dev/lazystorage backends such as the local filesystem, S3-compatible
// storage, or golazy.dev/pg/pgstorage. The lazymedia.Repository stores only the
// variant relationship and status; PostgreSQL applications can use
// golazy.dev/pg/pgmedia for that repository while using golazy.dev/pg/pgfiles
// for file metadata.
//
// Media.Variant first asks Repository for an existing ready variant. If one
// exists, Media opens the output file through FileStore and returns its file
// metadata without running the processor again. If no ready variant exists, or
// Regenerate is passed, Media opens the source file, calls Processor.Process,
// stores the returned body through FileStore.Put, and saves a ready Variant that
// points back to the new output file ID. Media.URL builds on Variant and then
// delegates URL generation to FileStore.URL.
//
// VariantKey is the stable name of the requested representation, such as
// "thumb", "preview", or "og". Spec carries opaque application-defined JSON
// that a processor can use for dimensions, codecs, quality settings, or any
// other generation input. lazymedia stores the spec on the Variant record but
// does not interpret it.
//
// Options follow the same pass-through convention used by lazystorage and
// lazyfiles: each layer consumes the options it understands and returns the
// rest. lazymedia consumes VariantKey, Spec, and Regenerate; generated file
// metadata is forwarded to FileStore.Put, including lazystorage.ContentType from
// Result.ContentType and OutputFilename from Result.Filename.
//
// lazymedia is useful outside a GoLazy app. A command-line importer can combine
// Media with a small JSONL repository and a filesystem-backed file service to
// generate previews locally, while a server can swap in lazyfiles plus
// PostgreSQL repositories without changing processors.
//
// The append-only JSONL repository implementation lives in the
// golazy.dev/lazymedia/jsonl subpackage. It is intended for local tools, tests,
// and simple single-process applications. Shared production deployments should
// use a database-backed Repository such as golazy.dev/pg/pgmedia.
package lazymedia
