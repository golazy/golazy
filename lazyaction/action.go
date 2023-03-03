package lazyaction

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golazy.dev/lazyaction/internal/args"
)

type Action struct {
	Method     string
	URL        url.URL
	Name       string
	Handler    http.HandlerFunc // If Fn is defined, Target is ignored (for generated code)
	Fn         *args.Fn
	Generators *map[string][]args.Gen
	Layout     *args.Fn

	Controller     any
	ControllerName string // PostsController
	Plural         string // posts
	Singular       string // post
	ParamName      string
}

func (r *Action) String() string {
	base := fmt.Sprintf("%9s %s %s", r.Method, r.URL.String(), r.Name)
	opts := []string{}

	if r.ControllerName != "" {
		opts = append(opts, r.ControllerName)
	}
	if r.Plural != "" {
		opts = append(opts, r.Plural)
	}
	if r.Singular != "" {
		opts = append(opts, r.Singular)
	}
	if r.ParamName != "" {
		opts = append(opts, r.ParamName)
	}

	return base + " " + strings.Join(opts, " ")
}
