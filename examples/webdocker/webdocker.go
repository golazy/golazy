package main

import (
	"github.com/golazy/golazy/lazyaction"
	"github.com/golazy/golazy/lazydev"
	"github.com/golazy/golazy/lazyview/nodes"
)

func main() {
	nodes.Beautify = true
	lazyaction.Route("/", new(Controller))

	lazydev.Serve(nil)
}
