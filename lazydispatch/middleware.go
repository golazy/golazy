package lazydispatch

import "net/http"

type middleware struct {
	name string
	m    http.Handler
}
