# lazycontroller

## Description

The `lazycontroller` package provides a base controller for handling HTTP requests in a web application. It includes features such as session management, CSRF protection, error handling, and rendering views.

## Usage

To use the `lazycontroller` package, create a new controller that embeds the `Base` struct and implements the necessary methods.

```go
package controllers

import (
	"golazy.dev/lazycontroller"
)

type MyController struct {
	lazycontroller.Base
}

func (c *MyController) Index() {
	// Your code here
}
```

## Dependencies

The `lazycontroller` package depends on the following packages:

- `github.com/gorilla/sessions`
- `golazy.dev/lazycontext`
- `golazy.dev/lazydispatch`
- `golazy.dev/lazyview`
- `golazy.dev/lazysupport`

## Installation

To install the `lazycontroller` package, run the following command:

```sh
go get golazy.dev/lazycontroller
```

## Contributing

If you would like to contribute to the `lazycontroller` package, please follow these steps:

1. Fork the repository
2. Create a new branch for your feature or bugfix
3. Commit your changes
4. Push your branch to your fork
5. Create a pull request

## Reporting Issues

If you encounter any issues with the `lazycontroller` package, please open an issue on the GitHub repository.
