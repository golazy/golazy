package lazydev

import (
	"log"
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
	switch DefaultServer.BootMode {
	case ProductionMode: // Listen on 443 and 80 with lego(acme)
		log.Println("Starting lazydev in production mode")
		return DefaultServer.serveProduction(handler)
	case ParentMode: // Listen on tcp 3000, builds, start the child and pass the fd
		log.Println("Starting lazydev in parent mode")
		return DefaultServer.startParent(handler)
	case ChildMode: // Takes the fd 3 and listen on http and https
		log.Println("Starting lazydev in child mode")
		return DefaultServer.serveChild(handler)
	default:
		panic("Unknown boot mode")
	}
}
