package main

import (
	"golazy.dev/lazydev"
)

func main() {

	//	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
	//		w.Write([]byte("hello"))
	//	}
	//	handler := http.HandlerFunc(handlerFunc)

	lazydev.Serve(nil)

}
