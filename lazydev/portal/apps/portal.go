package portal

import (
	"golazy.dev/lazyapp"
	"golazy.dev/lazydev/server"
)

type InternalApp struct {
	*lazyapp.App
	s *server.Server
}

var Http = &lazyapp.App{
	Name: "http",
}

var Golazy = &lazyapp.App{
	Name: "golazy",
}

var Fallback = &lazyapp.App{
	Name: "fallback",
}

func init() {

	Http.Router.Route("/", func() string {
		return "Welcome to golazy"
	})

	Golazy.Router.Route("/", func() string {
		return "Welcome to the portal"
	})

	Fallback.Router.Route("/", func() string {
		return "Welcome to the portal"
	})

}
