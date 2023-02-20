package lazyapp

import "net/http"

type handler struct {
	*App
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Router.ServeHTTP(w, r)

}
