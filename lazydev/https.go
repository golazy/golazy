package lazydev

import (
	_ "embed"
	"net/http"

	"github.com/guillermo/golazy/lazydev/injector"
)

//go:embed injector.js
var injectScript string

func httpsHandler(h http.Handler) http.Handler {
	if h == nil {
		h = http.DefaultServeMux
	}
	injectHandler := injector.Inject(h, "<script>"+injectScript+"</script>")
	mux := http.NewServeMux()
	mux.Handle("/", injectHandler)

	return mux
}
