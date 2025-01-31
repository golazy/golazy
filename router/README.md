# router

## Description

The `router` package provides a flexible and efficient way to route HTTP requests to user-defined handlers. It allows you to define routes and associate them with specific handlers, making it easy to build web applications with a clean and organized structure.

## Usage

### Creating Routes

To create routes using the `router` package, you need to create a new router and define your routes. Here's an example:

```go
package main

import (
	"fmt"
	"net/http"
	"golazy.dev/router"
)

func main() {
	r := router.NewRouter[string]()
	r.Add("GET", "/hello", "Hello, World!")
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		handler := r.Find(req.Method, req.URL.Path)
		if handler != nil {
			fmt.Fprintln(w, *handler)
		} else {
			http.NotFound(w, req)
		}
	})
	http.ListenAndServe(":8080", nil)
}
```

### Using Handlers

You can define handlers as any type that you want to associate with a route. In the example above, we used a simple string as the handler. You can also use more complex types, such as structs or functions.

### Middleware

You can add middleware to the router to handle common tasks such as logging, authentication, or request modification. Middleware functions should have the signature `func(http.Handler) http.Handler`. Here's an example of adding a logging middleware:

```go
func main() {
	r := router.NewRouter[string]()
	r.Add("GET", "/hello", "Hello, World!")
	http.HandleFunc("/", loggingMiddleware(func(w http.ResponseWriter, req *http.Request) {
		handler := r.Find(req.Method, req.URL.Path)
		if handler != nil {
			fmt.Fprintln(w, *handler)
		} else {
			http.NotFound(w, req)
		}
	}))
	http.ListenAndServe(":8080", nil)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Received request: %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
```

## Dependencies and Installation

To use the `router` package, you need to have Go installed on your system. You can install the `router` package using the following command:

```sh
go get golazy.dev/router
```

## Contributing and Reporting Issues

If you would like to contribute to the development of the `router` package or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.

## Variables

```golang
var Methods = []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE", "WS"}
```

## Functions

### func [IsMethod](/methods.go#L5)

`func IsMethod(method string) int`

### func [NewRouteTable](/route_tree.go#L75)

`func NewRouteTable[T any]() *routeTree[T]`

## Types

### type [Router](/router.go#L7)

`type Router[T any] struct { ... }`

#### func [NewRouter](/router.go#L29)

`func NewRouter[T any]() *Router[T]`

#### func (*Router[T]) [Add](/router.go#L48)

`func (r *Router[T]) Add(verb, path string, thing *T)`

#### func (*Router[T]) [All](/router.go#L40)

`func (r *Router[T]) All() []*T`

#### func (*Router[T]) [Find](/router.go#L60)

`func (r *Router[T]) Find(verb, path string) *T`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
