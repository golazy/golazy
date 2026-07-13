// Package lazyfs combines multiple read-only filesystems into one layered
// filesystem.
//
// Filesystems are added from lowest to highest precedence. Add is safe to call
// concurrently while a stack is being assembled. Seal prevents later changes
// once initialization is complete. Reads take a snapshot of the current stack,
// so they are safe both during assembly and after sealing.
//
// Files in higher layers replace files or directories at the same path.
// Directories in higher layers merge their entries with directories from lower
// layers, with each child following the same precedence rules.
package lazyfs
