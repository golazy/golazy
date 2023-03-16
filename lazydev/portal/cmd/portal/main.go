package main

import (
	"fmt"
	"os"
	"portal/apps/portal"
)

func main() {
	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println("cmd/portal/main.go Running in", d)
	portal.App.ListenAndServe("127.0.0.1:2000")
}
