# Lazy Dispatch

lazydispatch is a module of golazy to route http request into user defined actions.

This document covers the funcionality of lazydispatch alone. Please read the golazy intro first before aproaching this document.

## Description

lazydispatch is a package that provides a flexible and efficient way to route HTTP requests to user-defined actions. It allows you to define routes and associate them with specific handlers or controllers, making it easy to build web applications with a clean and organized structure.

## Usage

### Creating Routes

To create routes using lazydispatch, you need to create a new dispatcher and define your routes within a scope. Here's an example:

```go
package main

import (
	"net/http"
	"golazy.dev/lazydispatch"
)

func main() {
	dispatcher := lazydispatch.New()
	dispatcher.Draw(func(drawer *lazydispatch.Scope) {
		drawer.Resources(&PostsController{})
	})
	http.ListenAndServe(":2000", dispatcher)
}

type PostsController struct{}

func (c *PostsController) Index() {
	// Handle the index action
}

func (c *PostsController) Show(id string) {
	// Handle the show action
}
```

### Using Controllers

You can define controllers with methods that handle specific actions. The methods should have the appropriate signatures to match the expected parameters. Here's an example of a controller with an index and show action:

```go
type PostsController struct{}

func (c *PostsController) Index() {
	// Handle the index action
}

func (c *PostsController) Show(id string) {
	// Handle the show action
}
```

### Middleware

You can add middleware to the dispatcher to handle common tasks such as logging, authentication, or request modification. Middleware functions should have the signature `func(http.Handler) http.Handler`. Here's an example of adding a logging middleware:

```go
func main() {
	dispatcher := lazydispatch.New()
	dispatcher.Use(loggingMiddleware)
	dispatcher.Draw(func(drawer *lazydispatch.Scope) {
		drawer.Resources(&PostsController{})
	})
	http.ListenAndServe(":2000", dispatcher)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		next.ServeHTTP(w, r)
	})
}
```

## Dependencies and Installation

To use lazydispatch, you need to have Go installed on your system. You can install lazydispatch using the following command:

```sh
go get golazy.dev/lazydispatch
```

## Contributing and Reporting Issues

If you would like to contribute to the development of lazydispatch or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.

