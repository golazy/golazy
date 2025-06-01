package lazydispatch

import "net/http"

type middleware struct {
	name string
	m    http.Handler
}

type Middleware func(http.Handler) http.Handler

var DefaultMiddlewares []Middleware
