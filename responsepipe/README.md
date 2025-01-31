# responsepipe

## Description

responsepipe is a package that provides a way to intercept and modify HTTP responses in a Go web application. It allows you to capture the response body, headers, and status code, and make modifications before sending the response to the client.

## Usage

### Intercepting and Modifying Responses

To intercept and modify responses using responsepipe, you need to wrap your HTTP handler with the responsepipe handler. Here's an example:

```go
package main

import (
	"io"
	"net/http"
	"strings"
	"golazy.dev/responsepipe"
)

func main() {
	handler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reader, writer, code, err := responsepipe.New(w, r, h)
			if err != nil {
				return
			}

			io.Copy(writer, reader)

			data, err := io.ReadAll(reader)
			if err != nil {
				return
			}

			writer.WriteHeader()

			_, err = writer.Write([]byte(strings.ToUpper(string(data))))
			if err != nil {
				return
			}
		})
	}

	http.Handle("/", handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})))

	http.ListenAndServe(":8080", nil)
}
```

### Using the Response Struct

The responsepipe package provides a Response struct that allows you to access and modify the response body, headers, and status code. Here's an example of how to use the Response struct:

```go
package main

import (
	"net/http"
	"golazy.dev/responsepipe"
)

func main() {
	handler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reader, writer, code, err := responsepipe.New(w, r, h)
			if err != nil {
				return
			}

			response := responsepipe.Response{
				Body:   reader,
				Header: writer.Header(),
				Status: code,
			}

			// Modify the response body
			response.Body = strings.NewReader("modified response body")

			// Modify the response headers
			response.Header.Set("Content-Type", "text/plain")

			// Modify the response status code
			response.Status = http.StatusOK

			writer.WriteHeader(response.Status)
			io.Copy(writer, response.Body)
		})
	}

	http.Handle("/", handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})))

	http.ListenAndServe(":8080", nil)
}
```

## Dependencies and Installation

To use responsepipe, you need to have Go installed on your system. You can install responsepipe using the following command:

```sh
go get golazy.dev/responsepipe
```

## Contributing and Reporting Issues

If you would like to contribute to the development of responsepipe or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
