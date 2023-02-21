package router

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

type Route struct {
	Verb    string
	Path    string
	Name    string
	Handler http.HandlerFunc // If Fn is defined, Target is ignored (for generated code)
	Target  reflect.Value
	Args    []string
	Rets    []string

	Controller     any
	ControllerName string // PostsController
	Plural         string // posts
	Singular       string // post
	ParamName      string
}

func (r *Route) String() string {
	base := fmt.Sprintf("%9s %s %s", r.Verb, r.Path, r.Name)
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
