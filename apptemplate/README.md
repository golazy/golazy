# apptemplate

## Description

The `apptemplate` package provides a framework for creating and managing application templates. It allows developers to define templates with various actions and run them with specific options.

## Usage

### Example

```go
package main

import (
	"apptemplate"
	"apptemplate/memfs"
	"log"
)

func main() {
	template := apptemplate.Template{
		Name: "example",
	}
	template.Copy(memfs.MemFS{
		"file1.txt": "Hello, World!",
		"file2.txt": "This is a test.",
	})

	opts := apptemplate.RunOpts{
		Dest: "./output",
		Vars: map[string]string{
			"Var1": "Value1",
			"Var2": "Value2",
		},
		Logger: apptemplate.DefaultLogger,
		OnConflict: func(path string, content io.Reader) error {
			log.Printf("File conflict: %s", path)
			return nil
		},
	}

	err := template.Run(opts)
	if err != nil {
		log.Fatalf("Error running template: %v", err)
	}
}
```

## Dependencies

- Go 1.16 or later

## Installation

To install the `apptemplate` package, run:

```sh
go get github.com/golazy/golazy/apptemplate
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## Reporting Issues

If you encounter any issues or have any questions, please open an issue on GitHub.
