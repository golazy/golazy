# lazyview

## Description

lazyview is a package that provides a flexible and efficient way to render views in web applications. It allows you to define and manage views, and render them using different engines, making it easy to build web applications with a clean and organized structure.

## Usage

### Creating Views

To create views using lazyview, you need to create a new instance of the Views struct and define your views within a scope. Here's an example:

```go
package main

import (
	"context"
	"golazy.dev/lazyview"
	"golazy.dev/memfs"
	"bytes"
	"fmt"
)

func main() {
	views := &lazyview.Views{
		FS: memfs.New().Add("test.tpl", []byte("{{.Name}}")),
		Engines: map[string]lazyview.Engine{
			"tpl": &lazyview.Engine{},
		},
	}

	buf := &bytes.Buffer{}
	vars := map[string]any{
		"Name": "John",
	}
	err := views.RenderTemplate(context.Background(), buf, vars, "test.tpl")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Rendered view:", buf.String())
}
```

### Using Engines

You can define engines to render views with different templating languages. The engines should implement the `lazyview.Engine` interface. Here's an example of a simple engine that renders raw text files:

```go
package raw

import (
	"context"
	"io"
	"golazy.dev/lazyview"
)

type Engine struct{}

var _ lazyview.Engine = &Engine{}

func (e *Engine) Render(ctx context.Context, views *lazyview.Views, w io.Writer, vars map[string]any, helpers map[string]any, file string) error {
	f, err := views.FS.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}
```

### Middleware

You can add middleware to the views to handle common tasks such as logging, authentication, or request modification. Middleware functions should have the signature `func(http.Handler) http.Handler`. Here's an example of adding a logging middleware:

```go
func main() {
	views := &lazyview.Views{
		FS: memfs.New().Add("test.tpl", []byte("{{.Name}}")),
		Engines: map[string]lazyview.Engine{
			"tpl": &lazyview.Engine{},
		},
	}

	buf := &bytes.Buffer{}
	vars := map[string]any{
		"Name": "John",
	}
	err := views.RenderTemplate(context.Background(), buf, vars, "test.tpl")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Rendered view:", buf.String())
}
```

## Dependencies and Installation

To use lazyview, you need to have Go installed on your system. You can install lazyview using the following command:

```sh
go get golazy.dev/lazyview
```

## Contributing and Reporting Issues

If you would like to contribute to the development of lazyview or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
