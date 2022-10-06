package main

import (
	"net/http"

	"github.com/golazy/golazy/lazydev/lazydev"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	err := lazydev.Serve(nil)
	if err != nil {
		panic(err)
	}
}
