# multihttp

## Description

multihttp is a package that provides a way to serve HTTP, HTTPS, and HTTP/3 on the same port. It allows you to define a server that can handle multiple protocols, making it easy to build web applications with support for different protocols.

## Usage

### Creating a Server

To create a server using multihttp, you need to create a new server and define your handler. Here's an example:

```go
package main

import (
	"net/http"
	"golazy.dev/multihttp"
)

func main() {
	s := &multihttp.Server{
		Addr: "127.0.0.1:1999",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hi"))
		}),
		TLSConfig: getTLSConfig(),
	}

	err := s.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func getTLSConfig() *tls.Config {
	// Implement your TLS configuration here
	return &tls.Config{}
}
```

### Closing the Server

You can close the server by calling the `Close` method. Here's an example:

```go
func main() {
	s := &multihttp.Server{
		Addr: "127.0.0.1:1999",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hi"))
		}),
		TLSConfig: getTLSConfig(),
	}

	done := make(chan struct{})
	go func() {
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
		done <- struct{}{}
	}()

	// Simulate some work
	time.Sleep(5 * time.Second)

	err := s.Close()
	if err != nil {
		panic(err)
	}

	<-done
}
```

## Dependencies and Installation

To use multihttp, you need to have Go installed on your system. You can install multihttp using the following command:

```sh
go get golazy.dev/multihttp
```

## Contributing and Reporting Issues

If you would like to contribute to the development of multihttp or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
