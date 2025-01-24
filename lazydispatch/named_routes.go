package lazydispatch

import (
	"fmt"
	"reflect"
	"strings"

	"golazy.dev/lazysupport"
)

type namedRoutes struct {
	routes []Route
}

func newNamedRoutes() *namedRoutes {
	return &namedRoutes{
		routes: []Route{},
	}
}

func (nr *namedRoutes) Add(route Route) {
	nr.routes = append(nr.routes, route)
}

func (nr *namedRoutes) PathFor(details ...any) string {
	args := details
	if len(args) == 0 {
		panic("PathFor requires at least one argument")
	}

	// Find route
	var r *Route
	if name, ok := args[0].(string); ok {
		args = args[1:]
		r = nr.findByName(name, args)
	} else {
		r = nr.findByModel(args...)
	}

	if r == nil {
		// TODO: What the hell is that?
		info := []string{}
		for _, d := range details {
			switch d := d.(type) {
			case string:
				info = append(info, d)
			case int, uint:
				info = append(info, fmt.Sprintf("%d", d))
			default:
				name := reflect.TypeOf(d).String()
				id, err := lazysupport.IDFor(d)
				if err == nil {
					name = fmt.Sprintf("%s(%s)", name, id)
				}
				info = append(info, name)
			}
		}
		panic(fmt.Sprintf("path not found for %v", info))
	}

	if len(args) != countRequiredParams(r.Path) {
		panic(fmt.Sprintf("path name %q requires %d arguments. Got %d", r.Name, countRequiredParams(r.Path), len(args)))
	}

	return buildRoute(r, args...)
}

func countRequiredParams(path string) int {
	segments := strings.Split(path, "/")
	count := 0
	for _, s := range segments {
		if strings.HasPrefix(s, ":") || s == "*" {
			count++
		}
	}
	return count
}

func buildRoute(r *Route, params ...any) string {
	path := r.Path
	segments := strings.Split(path, "/")
	paramI := 0
	for i, s := range segments {
		if strings.HasPrefix(s, ":") {
			if len(params) <= paramI {
				panic(fmt.Sprintf("not enough params to replace %s", s))
			}
			segments[i] = getID(params[paramI])
			paramI++
		}
	}

	return strings.Join(segments, "/")
}

func getID(a any) string {
	id, err := lazysupport.IDFor(a)
	if err != nil {
		panic(err)
	}
	return id
}

func sameModels(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		aName := reflect.TypeOf(v).String()
		bName := reflect.TypeOf(b[i]).String()
		if aName == bName {
			continue
		}
		return false
	}
	return true
}

func (nr *namedRoutes) findByModel(args ...any) *Route {
	for _, r := range nr.routes {
		if sameModels(r.Models, args) {
			return &r
		}
	}
	return nil
}

func (nr *namedRoutes) findByName(name string, _ ...any) *Route {
	for _, r := range nr.routes {
		if r.Name == name {
			return &r
		}
	}
	return nil
}
