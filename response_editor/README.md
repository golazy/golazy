# response_editor

## Description

The `response_editor` package provides a buffered HTTP response writer that allows for editing the response before it is sent to the client. It supports the `http.Flusher`, `http.Pusher`, and `http.Hijacker` interfaces.

## Usage

### Example

```go
package main

import (
	"net/http"
	"response_editor"
)

func main() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	editHandler := response_editor.EditHandler{
		Handler: handler,
		Edit: func(response response_editor.Response) {
			body := response.Body()
			*body = append(*body, []byte(" Edited!")...)
		},
	}

	http.ListenAndServe(":8080", editHandler)
}
```

## Dependencies

- Go 1.16 or later

## Installation

To install the `response_editor` package, run:

```sh
go get github.com/golazy/golazy/response_editor
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## Reporting Issues

If you encounter any issues or have any questions, please open an issue on GitHub.
