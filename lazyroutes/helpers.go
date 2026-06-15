package lazyroutes

import (
	"fmt"
	"net/url"
	"strings"
)

// RegisterHelpers returns template helpers provided by the router.
func (s *Scope) RegisterHelpers() map[string]any {
	return map[string]any{
		"path_for": s.PathFor,
	}
}

// PathFor builds a path from a named route and route parameter values.
func (s *Scope) PathFor(name string, values ...any) (string, error) {
	route, ok := s.routeByName(name)
	if !ok {
		return "", fmt.Errorf("lazyroutes: route %q not found", name)
	}

	params := namedParamNamesFromPath(route.Path)
	if len(values) != len(params) {
		return "", fmt.Errorf("lazyroutes: route %q requires %d params, got %d", name, len(params), len(values))
	}

	path := route.Path
	for index, param := range params {
		value := url.PathEscape(fmt.Sprint(values[index]))
		path = strings.ReplaceAll(path, "{"+param+"}", value)
	}
	return path, nil
}

func (s *Scope) routeByName(name string) (Route, bool) {
	root := s.rootScope()
	for _, route := range root.Routes {
		if route.Name == name {
			return route, true
		}
	}
	return Route{}, false
}

func namedParamNamesFromPath(path string) []string {
	var names []string
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		if name == "$" || strings.TrimSpace(name) == "" {
			continue
		}
		name = strings.TrimSuffix(name, "...")
		names = append(names, name)
	}
	return names
}
