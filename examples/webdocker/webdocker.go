package main

import (
	"golazy.dev/lazyaction"
	"golazy.dev/lazydev"
	"golazy.dev/lazyview/nodes"
)

func main() {
	nodes.Beautify = true
	lazyaction.Route("/", new(Controller))

	lazydev.Serve(nil)
}
