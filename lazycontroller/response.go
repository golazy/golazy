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

func WriteErrorDetail(w http.ResponseWriter, _ *http.Request, err error) {
	ResetResponse(w)
	status := StatusCode(err)
	http.Error(w, err.Error(), status)
}

// Status sets the HTTP status code used by the next controller render.
//
// It does not write the response immediately, so actions can still rely on
// automatic rendering after setting a non-200 status.
func (b *Base) Status(code int) {
	b.status = code
}

// Header returns the response header map for the current controller request.
func (b *Base) Header() http.Header {
	if b == nil || b.writer == nil {
		return http.Header{}
	}
	return b.writer.Header()
}

// ContentType sets the response Content-Type header.
func (b *Base) ContentType(value string) {
	b.Header().Set("Content-Type", value)
}
