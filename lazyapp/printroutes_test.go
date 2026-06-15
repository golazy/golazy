package lazyapp

import (
	"bytes"
	"strings"
	"testing"

	"golazy.dev/lazyroutes"
)

func TestWriteRoutesJSONL(t *testing.T) {
	routes := lazyroutes.RouteTable{
		{
			Method:     "GET",
			Path:       "/",
			Name:       "root",
			Controller: "home",
			Action:     "Index",
		},
		{
			Method:      "GET",
			Path:        "/posts/{post_id}",
			Name:        "post",
			Controller:  "posts",
			Action:      "Show",
			NamedParams: map[string]bool{"post_id": true},
		},
	}

	var out bytes.Buffer
	if err := writeRoutesJSONL(&out, routes); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], `"params":{}`) {
		t.Fatalf("first line = %s, want empty params object", lines[0])
	}
	if !strings.Contains(lines[1], `"params":{"post_id":true}`) {
		t.Fatalf("second line = %s, want post_id params object", lines[1])
	}
}
