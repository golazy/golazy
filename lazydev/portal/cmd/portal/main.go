package main

import (
	"fmt"
	"os"
	"portal/apps/portal"
)

func main() {
	addr := getListenAddr()

	fmt.Println("DevPortal Running in " + addr)
	err := portal.App.ListenAndServe(addr)
	if err != nil {
		panic(err)
	}

}

func getListenAddr() string {
	listen := os.Getenv("LISTEN")
	if listen != "" {
		return listen
	}
	port := os.Getenv("PORT")
	if port != "" {
		return ":" + port
	}

	return "127.0.0.1:2000"
}
