# flattenfs

## Description

The `flattenfs` package provides a way to flatten a filesystem by replacing all directory separators with a custom separator. This can be useful for embedding files in a Go application using the `embed` package.

## Usage

### Example

```go
package main

import (
	"embed"
	"fmt"
	"log"

	"golazy.dev/flattenfs"
)

//go:embed testdata
var FS embed.FS

func main() {
	fs := flattenfs.FlattenFS{FS}

	files, err := fs.Glob("**/*.tpl")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(file)
	}
}
```

### Functions

- `Flat(pattern string) string`: Replaces all "/" with the custom separator in the pattern.
- `UnFlat(pattern string) string`: Replaces all custom separators with "/" in the pattern.
- `Glob(inputFS fs.FS, pattern string) ([]string, error)`: Returns a list of files matching the pattern, including support for double glob (`**/`).

### Types

- `FlattenFS`: A wrapper around `fs.FS` that adds support for flattening the filesystem.

## Dependencies

- Go 1.16 or later

## Installation

To install the package, run:

```sh
go get golazy.dev/flattenfs
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## Reporting Issues

If you encounter any issues, please open a GitHub issue.
