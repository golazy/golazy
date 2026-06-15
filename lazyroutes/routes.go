package lazyroutes

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Scope is the routing DSL entrypoint used by application routes.
// It embeds the standard library ServeMux so the same object is directly
// usable as an http.Handler.
type Scope struct {
	*http.ServeMux
	Context context.Context
	Routes  RouteTable

	root       *Scope
	pathPrefix string
	namePrefix string
	namespace  string
}

// New builds a scope with the framework's public-file fallback already wired.
func New(ctx context.Context) *Scope {
	scope := &Scope{
		ServeMux: http.NewServeMux(),
		Context:  ctx,
	}
	scope.root = scope
	return scope
}

// Namespace creates a child scope with path, route-name, and namespace prefixes.
func (s *Scope) Namespace(name string, draw ...func(*Scope)) *Scope {
	child := s.child(name, name, name)
	child.draw(draw...)
	return child
}

// Path creates a child scope with a path prefix.
func (s *Scope) Path(path string, draw ...func(*Scope)) *Scope {
	child := s.child(path, "", "")
	child.draw(draw...)
	return child
}

// As creates a child scope with a route-name prefix.
func (s *Scope) As(name string, draw ...func(*Scope)) *Scope {
	child := s.child("", name, "")
	child.draw(draw...)
	return child
}

func (s *Scope) register(method, path string, route Route, handler http.Handler) {
	path = s.scopedPath(path)
	if route.Path == "" {
		route.Path = path
	} else {
		route.Path = s.scopedPath(route.Path)
	}
	route.Method = strings.ToUpper(method)
	if route.Method == "" {
		route.Method = http.MethodGet
	}
	if route.Name == "" {
		route.Name = inferRouteName(route.Method, route.Path)
	}
	route.Name = joinRoutePart("_", s.namePrefix, route.Name)
	if route.Namespace == "" {
		route.Namespace = s.namespace
	}
	route.NamedParams = namedParamsFromPath(route.Path)

	pattern := route.Method + " " + serveMuxPath(route.Path)
	wrapped := routeContextMiddleware(handler, route)
	root := s.rootScope()
	root.ServeMux.Handle(pattern, wrapped)
	root.Routes = append(root.Routes, route)
}

func (s *Scope) action(method, path string, route Route, handlerAction Action) {
	s.register(method, path, route, Handle(handlerAction))
}

// HandleFunc registers a non-controller route action.
func (s *Scope) HandleFunc(method, path string, handlerAction Action) {
	if s == nil {
		panic(fmt.Errorf("lazyroutes: route scope is nil"))
	}
	s.action(method, path, Route{}, handlerAction)
}

func (s *Scope) HandlesPath(path string) bool {
	root := s.rootScope()
	for _, route := range root.Routes {
		if routePathMatches(route.Path, path) {
			return true
		}
	}
	return false
}

func routeContextMiddleware(handler http.Handler, route Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), routeContextKey{}, routeContextFromRequest(route, r)))
		handler.ServeHTTP(w, r)
	})
}

func normalizePath(path string) string {
	path = "/" + strings.Trim(path, "/")
	if path == "/" {
		return "/"
	}
	return path
}

func serveMuxPath(path string) string {
	if path == "/" {
		return "/{$}"
	}
	return path
}

func (s *Scope) child(pathPrefix, namePrefix, namespace string) *Scope {
	root := s.rootScope()
	return &Scope{
		ServeMux:   root.ServeMux,
		Context:    root.Context,
		root:       root,
		pathPrefix: joinRoutePart("/", s.pathPrefix, pathPrefix),
		namePrefix: joinRoutePart("_", s.namePrefix, namePrefix),
		namespace:  joinRoutePart("/", s.namespace, namespace),
	}
}

func (s *Scope) draw(draw ...func(*Scope)) {
	for _, fn := range draw {
		if fn != nil {
			fn(s)
		}
	}
}

func (s *Scope) rootScope() *Scope {
	if s == nil {
		panic(fmt.Errorf("lazyroutes: route scope is nil"))
	}
	if s.root != nil {
		return s.root
	}
	return s
}

func (s *Scope) scopedPath(path string) string {
	if s.pathPrefix == "" {
		return normalizePath(path)
	}
	if path == "/" || path == "/{$}" || strings.Trim(path, "/") == "" {
		return normalizePath(s.pathPrefix)
	}
	return normalizePath(joinRoutePart("/", s.pathPrefix, path))
}

func joinRoutePart(separator string, parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, separator)
		if strings.TrimSpace(part) == "" {
			continue
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return ""
	}
	return strings.Join(clean, separator)
}
