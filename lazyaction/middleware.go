package lazyaction

import "net/http"

type Middleware interface {
	ServeHTTP(http.ResponseWriter, *http.Request, http.HandlerFunc)
	SetChild(http.Handler)
}
