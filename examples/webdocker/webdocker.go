package main

import (
	"github.com/guillermo/golazy/lazyaction"
	"github.com/guillermo/golazy/lazydev"
	"github.com/guillermo/golazy/lazyview/nodes"
)

func main() {
	nodes.Beautify = true
	lazyaction.Route("/", new(Controller))

	lazydev.Serve(nil)
}
