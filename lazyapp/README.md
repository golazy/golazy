# lazyapp

## Description

The `lazyapp` package is the core of the GoLazy framework, providing the main application structure and essential services for building web applications.

## Usage

```go
package main

import (
	"context"
	"net/http"
	"time"

	"golazy.dev/lazyapp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	app := lazyapp.New("MyApp", "1.0.0")
	app.LazyAssets.AddFile("index.html", []byte("Hello, World!"))

	errCh := app.Start(ctx)

	resp, err := http.Get("http://localhost:2000/index.html")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if string(body) != "Hello, World!" {
		panic("Unexpected response: " + string(body))
	}
	cancel()

	if err := <-errCh; err != nil {
		panic(err)
	}
}
```

## Dependencies and Installation

To install the `lazyapp` package, use the following command:

```sh
go get golazy.dev/lazyapp
```

## Contributing and Reporting Issues

Contributions and issues are welcome. Please open an issue on the GitHub repository or submit a pull request with your changes.
