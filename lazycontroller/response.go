package lazycontroller

import "net/http"

type responseState interface {
	WasResponseSent() bool
}

type responseResetter interface {
	Reset()
}

type responseUnwrapper interface {
	Unwrap() http.ResponseWriter
}

func WasResponseSent(w http.ResponseWriter) bool {
	if w == nil {
		return false
	}
	if state, ok := w.(responseState); ok {
		return state.WasResponseSent()
	}
	if unwrapper, ok := w.(responseUnwrapper); ok {
		next := unwrapper.Unwrap()
		if next != nil && next != w {
			return WasResponseSent(next)
		}
	}
	return false
}

func ResetResponse(w http.ResponseWriter) bool {
	if w == nil {
		return false
	}
	if resetter, ok := w.(responseResetter); ok {
		resetter.Reset()
		return true
	}
	if unwrapper, ok := w.(responseUnwrapper); ok {
		next := unwrapper.Unwrap()
		if next != nil && next != w {
			return ResetResponse(next)
		}
	}
	return false
}

func WriteError(w http.ResponseWriter, _ *http.Request, err error) {
	ResetResponse(w)
	status := StatusCode(err)
	http.Error(w, http.StatusText(status), status)
}
