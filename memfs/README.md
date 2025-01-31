# memfs

## Description

The `memfs` package provides an in-memory file system implementation. It allows you to create, read, and manipulate files and directories in memory without interacting with the actual file system. This can be useful for testing, temporary file storage, or other scenarios where you need a lightweight and fast file system.

## Usage

### Creating a New In-Memory File System

To create a new in-memory file system, use the `New` function:

```go
package main

import (
	"golazy.dev/memfs"
)

func main() {
	fs := memfs.New()
	// Use the file system
}
```

### Adding Files and Directories

You can add files and directories to the in-memory file system using the `Add` method:

```go
package main

import (
	"golazy.dev/memfs"
)

func main() {
	fs := memfs.New()
	fs.Add("/a/b/c/hello.txt", []byte("hello world"))
}
```

### Reading Files

To read files from the in-memory file system, use the `Open` method:

```go
package main

import (
	"bytes"
	"fmt"
	"golazy.dev/memfs"
)

func main() {
	fs := memfs.New()
	fs.Add("/a/b/c/hello.txt", []byte("hello world"))

	f, err := fs.Open("/a/b/c/hello.txt")
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	buf.ReadFrom(f)

	fmt.Println(buf.String()) // Output: hello world
}
```

### Listing Directory Contents

To list the contents of a directory, use the `ReadDir` method:

```go
package main

import (
	"fmt"
	"golazy.dev/memfs"
)

func main() {
	fs := memfs.New()
	fs.Add("file", []byte("file"))
	fs.Add("file/b", []byte("fb"))
	fs.Add("file/c", []byte("fc"))
	fs.Add("/c", []byte("c"))

	entries, err := fs.ReadDir("")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		fmt.Println(entry.Name())
	}
}
```

## Dependencies and Installation

To use the `memfs` package, you need to have Go installed on your system. You can install the `memfs` package using the following command:

```sh
go get golazy.dev/memfs
```

## Contributing and Reporting Issues

If you would like to contribute to the development of `memfs` or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
