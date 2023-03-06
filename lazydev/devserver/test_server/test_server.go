package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test_server"))
	}))

	addr := os.Getenv("LISTEN")
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			addr = "127.0.0.1:2001"
		} else {
			addr = "127.0.0.1:" + port
		}
	}
	fmt.Println("Listening on", "http://"+addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}

}
