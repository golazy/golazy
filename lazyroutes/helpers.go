package lazyroutes

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"golazy.dev/lazypath"
)

type ModelRoutes struct {
	Create string
	Update string
	Delete string
}

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

	routeValues, queryParams := lazypath.SplitValues(values)
	params := namedParamNamesFromPath(route.Path)
	if len(routeValues) != len(params) {
		return "", fmt.Errorf("lazyroutes: route %q requires %d params, got %d", name, len(params), len(routeValues))
	}

	path := route.Path
	for index, param := range params {
		value := url.PathEscape(fmt.Sprint(routeValues[index]))
		path = strings.ReplaceAll(path, "{"+param+"}", value)
	}
	return lazypath.AppendURLParams(path, queryParams), nil
}

func (s *Scope) ModelRoutesFor(model any) (ModelRoutes, bool) {
	t := modelType(model)
	if t == nil {
		return ModelRoutes{}, false
	}
	routes, ok := s.rootScope().models[t]
	return routes, ok
}

func (s *Scope) PathForModel(model any, action string) (string, error) {
	routes, ok := s.ModelRoutesFor(model)
	if !ok {
		return "", fmt.Errorf("lazyroutes: model route for %T not found", model)
	}

	var routeName string
	switch strings.ToLower(action) {
	case "create":
		routeName = routes.Create
	case "update":
		routeName = routes.Update
	case "delete":
		routeName = routes.Delete
	default:
		return "", fmt.Errorf("lazyroutes: model action %q is not supported", action)
	}
	if routeName == "" {
		return "", fmt.Errorf("lazyroutes: model action %q is not registered for %T", action, model)
	}
	if strings.EqualFold(action, "create") {
		return s.PathFor(routeName)
	}
	param, ok := modelRouteParam(model)
	if !ok {
		return "", fmt.Errorf("lazyroutes: model %T does not expose a route parameter", model)
	}
	return s.PathFor(routeName, param)
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

func modelRouteParam(model any) (any, bool) {
	if param, ok := model.(interface{ RouteParam() string }); ok {
		return param.RouteParam(), true
	}
	value := reflect.ValueOf(model)
	if value.IsValid() && value.Kind() != reflect.Ptr && value.CanAddr() {
		model = value.Addr().Interface()
	}
	if id, ok := model.(interface{ ID() int }); ok {
		return id.ID(), true
	}
	if id, ok := model.(interface{ ID() string }); ok {
		return id.ID(), true
	}
	return nil, false
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
