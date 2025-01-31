# lazytest

## Description

The `lazytest` package provides utilities for testing applications built with the `lazyapp` package. It includes helpers for creating test applications, making requests, and verifying responses.

## Usage

To use the `lazytest` package, create a new test application and use the provided methods to make requests and verify responses.

```go
package main

import (
	"testing"
	"golazy.dev/lazyapp"
	"golazy.dev/lazytest"
)

func TestMyApp(t *testing.T) {
	app := lazyapp.New()
	at := lazytest.NewAppTest(t, app)

	response := at.Request("GET", nil, "/")
	response.ExpectCode(200).Contains("Hello, world!")
}
```

## Dependencies

The `lazytest` package depends on the following packages:

- `golazy.dev/lazyapp`
- `golazy.dev/lazydispatch`
- `golazy.dev/lazyservice`

## Installation

To install the `lazytest` package, run the following command:

```sh
go get golazy.dev/lazytest
```

## Contributing

If you would like to contribute to the `lazytest` package, please follow these steps:

1. Fork the repository
2. Create a new branch for your feature or bugfix
3. Commit your changes
4. Push your branch to your fork
5. Create a pull request

## Reporting Issues

If you encounter any issues with the `lazytest` package, please open an issue on the GitHub repository.
