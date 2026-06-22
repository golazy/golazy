package lazypath_test

import (
	"fmt"

	"golazy.dev/lazypath"
)

func ExampleAppendURLParams() {
	path := lazypath.AppendURLParams("/posts/hello#comments", lazypath.URLParams{
		"page": 2,
		"q":    "go lazy",
	})

	fmt.Println(path)
	// Output: /posts/hello?page=2&q=go+lazy#comments
}
