package serve

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"

	"github.com/gorilla/handlers"
	"golazy.dev/lazyview/nodes"
)

func Page(e nodes.Element) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e.WriteTo(w)
	}
}

func ServePage(e nodes.Element) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "2000"
	}

	http.Handle("/", Page(e))

	log.Println("Listening on http://localhost:" + port)

	err := http.ListenAndServe(":"+port, handlers.CombinedLoggingHandler(os.Stdout, http.DefaultServeMux))
	//err := http.ListenAndServe(":"+port, http.DefaultServeMux)
	//err := http.ListenAndServe(":"+port, logHandler(http.DefaultServeMux.ServeHTTP))
	if err != nil {
		panic(err)
	}

}

func logHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		x, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}
		log.Println(fmt.Sprintf("%q", x))
		rec := httptest.NewRecorder()
		fn(rec, r)
		log.Println(fmt.Sprintf("%q", rec.Body))

		// this copies the recorded response to the response writer
		for k, v := range rec.HeaderMap {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		rec.Body.WriteTo(w)
	}
}
