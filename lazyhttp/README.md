# lazyhttp

## Description

lazyhttp is a package that provides an HTTP server compatible with lazyapp. It allows you to create and run an HTTP server with ease, integrating seamlessly with the lazyapp framework.

## Usage

### Creating and Running the Server

To create and run an HTTP server using lazyhttp, you need to create a new HTTPService and configure it with the desired settings. Here's an example:

```go
package main

import (
	"context"
	"net/http"
	"time"
	"golazy.dev/lazyhttp"
)

func main() {
	addr := "localhost:8085"
	s := &lazyhttp.HTTPService{}
	s.Addr = addr
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	errCh := make(chan error)

	go func() {
		errCh <- s.Run(ctx)
	}()

	resp, err := http.Get("http://" + addr)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if string(body) != "Hello, world!" {
		panic("unexpected response: " + string(body))
	}

	if err = <-errCh; err != nil {
		panic(err)
	}
}
```

## Dependencies and Installation

To use lazyhttp, you need to have Go installed on your system. You can install lazyhttp using the following command:

```sh
go get golazy.dev/lazyhttp
```

## Contributing and Reporting Issues

If you would like to contribute to the development of lazyhttp or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
