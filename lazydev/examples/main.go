package main

import (
	"net/http"

	"github.com/golazy/golazy/lazydev"
)

func main() {

	lazydev.DefaultServeMux = http.HandlerFunc(func(w http.RespnoseWriter, r *http.Request) { w.Write([]byte("hello")) })

	lazydev.Serve()

}
