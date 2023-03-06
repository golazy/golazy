package main

import (
	"net/http"
	"os"
)

func main() {

	http.ListenAndServe(":"+os.Getenv("PORT"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend"))
	}))

}
