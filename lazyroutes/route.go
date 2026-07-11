package lazyroutes

import (
	"context"
	"maps"
	"net/http"
	"strings"
)

// Route is the metadata for one registered route.
type Route struct {
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Name        string          `json:"name"`
	Controller  string          `json:"controller,omitempty"`
	Action      string          `json:"action,omitempty"`
	Namespace   string          `json:"namespace,omitempty"`
	NamedParams map[string]bool `json:"params"`
}

// RouteTable is the list of routes registered during Draw.
type RouteTable []Route

type routeContextKey struct{}

type routeContext struct {
	Route  Route
	Values map[string]string
}

// RouteFromContext returns the route metadata and parameter values attached to a request context.
func RouteFromContext(ctx context.Context) (Route, map[string]string, bool) {
	routeContext, ok := ctx.Value(routeContextKey{}).(routeContext)
	if !ok {
		return Route{}, nil, false
	}
	values := map[string]string{}
	maps.Copy(values, routeContext.Values)
	return routeContext.Route, values, true
}

// RouteFromRequest returns the route metadata and parameter values attached to the request context.
func RouteFromRequest(r *http.Request) (Route, map[string]string, bool) {
	return RouteFromContext(r.Context())
}

func routeContextFromRequest(route Route, request *http.Request) routeContext {
	values := map[string]string{}
	for name := range route.NamedParams {
		values[name] = request.PathValue(name)
	}
	if len(values) == 0 {
		values = nil
	}
	return routeContext{
		Route:  route,
		Values: values,
	}
}

func inferRouteName(method string, path string) string {
	method = strings.ToUpper(method)
	if path == "/" || path == "/{$}" || strings.Trim(path, " /") == "" {
		return "root"
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 {
		return strings.ToLower(strings.Trim(method, "/"))
	}

	nonParams := make([]string, 0, len(segments))
	for _, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			continue
		}
		nonParams = append(nonParams, segment)
	}

	if len(nonParams) == 0 {
		return "route"
	}

	base := nonParams[len(nonParams)-1]
	if base == "new" && len(nonParams) >= 2 {
		return "new_" + nonParams[len(nonParams)-2]
	}
	if base == "edit" && len(nonParams) >= 2 {
		return "edit_" + nonParams[len(nonParams)-2]
	}

	if method == http.MethodGet && len(segments) == 1 {
		return base
	}

	return base
}

func namedParamsFromPath(path string) map[string]bool {
	params := map[string]bool{}
	segments := strings.SplitSeq(strings.Trim(path, "/"), "/")
	for segment := range segments {
		if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		if name == "$" {
			continue
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		name = strings.TrimSuffix(name, "...")
		params[name] = true
	}
	return params
}

func routePathMatches(pattern string, requestPath string) bool {
	pattern = strings.Trim(pattern, "/")
	requestPath = strings.Trim(requestPath, "/")

	if pattern == "{$}" {
		return requestPath == ""
	}

	patternSegments := splitRoutePath(pattern)
	requestSegments := splitRoutePath(requestPath)
	for i, segment := range patternSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "...}") {
			return len(requestSegments) >= i
		}
		if i >= len(requestSegments) {
			return false
		}
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			continue
		}
		if segment != requestSegments[i] {
			return false
		}
	}
	return len(patternSegments) == len(requestSegments)
}

func splitRoutePath(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return strings.Split(path, "/")
}
