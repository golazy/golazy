package lazyroutes

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"reflect"
	"strings"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazytelemetry"
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
	models     map[reflect.Type]ModelRoutes
}

// New builds a scope with the framework's public-file fallback already wired.
func New(ctx context.Context) *Scope {
	scope := &Scope{
		ServeMux: http.NewServeMux(),
		Context:  ctx,
		models:   map[reflect.Type]ModelRoutes{},
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
	namePath := path
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
		if s.namePrefix == "" {
			namePath = route.Path
		}
		route.Name = inferRouteName(route.Method, namePath)
	}
	route.Name = s.scopedName(route.Name)
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
	if s.handlesPath(path) {
		return true
	}
	strippedPath, _, ok := formatPath(path)
	return ok && s.handlesPath(strippedPath)
}

func (s *Scope) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	root := s.rootScope()
	if target, ok := root.trailingSlashRedirectPath(r); ok {
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	if strippedPath, format, ok := formatPath(r.URL.Path); ok && root.handlesPath(strippedPath) {
		r = requestWithFormat(r, strippedPath, format)
	}
	root.ServeMux.ServeHTTP(w, r)
}

func (s *Scope) handlesPath(path string) bool {
	root := s.rootScope()
	for _, route := range root.Routes {
		if routePathMatches(route.Path, path) {
			return true
		}
	}
	return false
}

func (s *Scope) handlesMethodPath(method string, path string) bool {
	root := s.rootScope()
	for _, route := range root.Routes {
		if !routeMethodMatches(route.Method, method) {
			continue
		}
		if routePathMatches(route.Path, path) {
			return true
		}
	}
	return false
}

func (s *Scope) recognizesMethodPath(method string, path string) bool {
	if s.handlesMethodPath(method, path) {
		return true
	}
	strippedPath, _, ok := formatPath(path)
	return ok && s.handlesMethodPath(method, strippedPath)
}

func routeMethodMatches(routeMethod string, requestMethod string) bool {
	routeMethod = strings.ToUpper(routeMethod)
	requestMethod = strings.ToUpper(requestMethod)
	return routeMethod == requestMethod || requestMethod == http.MethodHead && routeMethod == http.MethodGet
}

func (s *Scope) trailingSlashRedirectPath(r *http.Request) (string, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return "", false
	}
	if r.URL.Path == "/" || !strings.HasSuffix(r.URL.Path, "/") {
		return "", false
	}
	target := strings.TrimRight(r.URL.Path, "/")
	if target == "" {
		return "", false
	}
	if !s.recognizesMethodPath(r.Method, target) {
		return "", false
	}
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	return target, true
}

func formatPath(requestPath string) (string, lazycontroller.Format, bool) {
	suffix := path.Ext(requestPath)
	format, ok := lazycontroller.FormatFromSuffix(suffix)
	if !ok {
		return "", "", false
	}
	strippedPath := strings.TrimSuffix(requestPath, suffix)
	if strippedPath == "" {
		strippedPath = "/"
	}
	return strippedPath, format, true
}

func requestWithFormat(r *http.Request, path string, format lazycontroller.Format) *http.Request {
	clone := r.Clone(lazycontroller.WithFormat(r.Context(), format))
	url := *clone.URL
	url.Path = path
	url.RawPath = ""
	clone.URL = &url
	return clone
}

func routeContextMiddleware(handler http.Handler, route Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), routeContextKey{}, routeContextFromRequest(route, r))
		span := lazytelemetry.SpanFromContext(ctx)
		if span == nil {
			handler.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		span.AddAttributes(
			slog.String("http.route", route.Path),
			slog.String("route.name", route.Name),
			slog.String("controller", route.Controller),
			slog.String("action", route.Action),
		)
		ctx, region := lazytelemetry.StartRegion(ctx, "router",
			slog.String("http.route", route.Path),
			slog.String("route.name", route.Name),
			slog.String("controller", route.Controller),
			slog.String("action", route.Action),
		)
		if region != nil {
			defer region.End()
		}
		r = r.WithContext(ctx)
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

func (s *Scope) scopedName(name string) string {
	return joinRoutePart("_", s.namePrefix, name)
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
