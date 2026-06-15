package lazyapp

import (
	"encoding/json"
	"fmt"
	"io"

	"golazy.dev/lazyroutes"
)

func writeRoutesJSONL(w io.Writer, routes lazyroutes.RouteTable) error {
	encoder := json.NewEncoder(w)
	for index, route := range routes {
		if route.NamedParams == nil {
			route.NamedParams = map[string]bool{}
		}
		if err := encoder.Encode(route); err != nil {
			return fmt.Errorf("encode route %d: %w", index, err)
		}
	}
	return nil
}
