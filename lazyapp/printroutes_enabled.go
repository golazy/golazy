//go:build printroutes

package lazyapp

import (
	"fmt"
	"os"

	"golazy.dev/lazyroutes"
)

func init() {
	afterDraw = func(router *lazyroutes.Scope) {
		if router == nil {
			os.Exit(0)
		}
		if err := writeRoutesJSONL(os.Stdout, router.Routes); err != nil {
			fmt.Fprintf(os.Stderr, "lazyapp: print routes: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
