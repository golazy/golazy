package main

import (
	"github.com/golazy/golazy/lazydev"
)

func main() {

//	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
//		w.Write([]byte("hello"))
//	}
//	handler := http.HandlerFunc(handlerFunc)

	lazydev.Serve(nil)

}
