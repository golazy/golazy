package lazydev

import (
	"net/http"
	"os"
)

const childEnvKey = "GOLAZY_CHILDPROCESS"

func init() {
	if os.Getenv(childEnvKey) != "" {
		DefaultServer.BootMode = ChildMode
	}
}

func Serve(handler http.Handler) error {
	s := Server{}
	s = *DefaultServer
	s.Handler = handler

	return s.ListenAndServe()

}
