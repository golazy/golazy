package lazydev

import (
	"log"
	"net/http"
	"os"
)

const childEnvKey = "GOLAZY_CHILDPROCESS"

func isChild() bool {
	log.Println("is child", os.Getenv("GOLAZY_CHILDPROCESS") != "")
	return os.Getenv("GOLAZY_CHILDPROCESS") != ""
}

func Serve(handler http.Handler) error {
	if isChild() {
		return (&child{}).Serve(handler)
	}
	return (&parent{}).Serve(handler)

}
