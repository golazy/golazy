package lazyapp

import "net/http"

type Middleware interface {
	http.Handler
	SetNext(http.Handler)
}

type AppMiddleWare interface {
	SetApp(*App)
}
