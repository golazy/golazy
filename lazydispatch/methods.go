package lazydispatch

import "net/http"

func MethodNotAllowed(allowed ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for _, method := range allowed {
			w.Header().Add("Allow", method)
		}
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
}
