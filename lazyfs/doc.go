// Package lazyfs combines multiple read-only filesystems into one layered
// filesystem.
//
// Add layers from lowest to highest precedence. Higher files replace lower
// files, while directories merge:
//
//	files := lazyfs.New()
//	if err := files.Add(frameworkFS,
//		lazyfs.Name("framework"),
//		lazyfs.Owner("golazy.dev"),
//	); err != nil {
//		return err
//	}
//	if err := files.Add(appFS,
//		lazyfs.Name("application"),
//		lazyfs.Owner("example.com/myapp"),
//	); err != nil {
//		return err
//	}
//	if err := files.Seal(); err != nil {
//		return err
//	}
//
//	data, err := fs.ReadFile(files, "views/layout.html.tpl")
//	if err != nil {
//		return err
//	}
//	// data comes from appFS when both layers contain this path.
//
//	resolution, err := files.Resolve("views/layout.html.tpl")
//	if err != nil {
//		return err
//	}
//	// resolution.Winner identifies the application layer.
//
// Filesystems are added from lowest to highest precedence. Add is safe to call
// concurrently while a stack is being assembled. Seal prevents later changes
// once initialization is complete. Reads take a snapshot of the current stack,
// so they are safe both during assembly and after sealing.
//
// Files in higher layers replace files or directories at the same path.
// Directories in higher layers merge their entries with directories from lower
// layers, with each child following the same precedence rules. Mount exposes a
// filesystem below a path prefix without copying its contents.
package lazyfs
