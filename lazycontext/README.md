# lazycontext

## Description

The `lazycontext` package provides an accumulative context that can store values like `context.Context` or by type.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"io"

	"golazy.dev/lazycontext"
)

func main() {
	ctx := lazycontext.New()

	// Store values as with normal context
	var userKey string
	ctx.AddValue(userKey, "user_33")
	fmt.Println(ctx.Value(userKey))

	// Reference only by specific type
	type myConfig struct{ Name string }
	lazycontext.Set(ctx, myConfig{Name: "test"})
	cfg := lazycontext.Get[myConfig](ctx)
	fmt.Println(cfg.Name)

	// Output:
	// user_33
	// test
}
```

## Dependencies and Installation

To install the `lazycontext` package, use the following command:

```sh
go get golazy.dev/lazycontext
```

## Contributing and Reporting Issues

Contributions and issues are welcome. Please open an issue on the GitHub repository or submit a pull request with your changes.
