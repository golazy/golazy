package lazyroutes

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"golazy.dev/lazypath"
	"golazy.dev/lazyview"
)

type ModelRoutes struct {
	Create string
	Update string
	Delete string
}

// RegisterHelpers returns template helpers provided by the router.
func (s *Scope) RegisterHelpers() map[string]any {
	return map[string]any{
		"path_for":       s.PathFor,
		"link_to":        lazyview.Helper(linkTo),
		"attr":           linkAttr,
		"data":           linkData,
		"unless_current": linkUnlessCurrent,
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

type linkOptions struct {
	attrs         []linkAttribute
	unlessCurrent bool
}

type linkOption struct {
	attrs         []linkAttribute
	unlessCurrent bool
}

type linkAttribute struct {
	name  string
	value string
}

func linkTo(ctx *lazyview.Context, args ...any) (any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("lazyroutes: link_to expects at least 2 arguments, got %d", len(args))
	}
	text := fmt.Sprint(args[0])
	href, err := linkHref(fmt.Sprint(args[1]))
	if err != nil {
		return nil, err
	}
	options, err := collectLinkOptions(args[2:])
	if err != nil {
		return nil, err
	}

	body := html.EscapeString(text)
	var request *http.Request
	if ctx != nil {
		request = ctx.Request
	}
	if options.unlessCurrent && linkDestinationIsCurrent(request, href) {
		return lazyview.Fragment{
			ContentType: "text/html; charset=utf-8",
			Body:        body,
		}, nil
	}

	var builder strings.Builder
	builder.WriteString(`<a href="`)
	builder.WriteString(html.EscapeString(href))
	builder.WriteByte('"')
	for _, attr := range options.attrs {
		builder.WriteByte(' ')
		builder.WriteString(attr.name)
		builder.WriteString(`="`)
		builder.WriteString(html.EscapeString(attr.value))
		builder.WriteByte('"')
	}
	builder.WriteByte('>')
	builder.WriteString(body)
	builder.WriteString(`</a>`)

	return lazyview.Fragment{
		ContentType: "text/html; charset=utf-8",
		Body:        builder.String(),
	}, nil
}

func linkAttr(name string, value any) (linkOption, error) {
	return newLinkAttribute(name, value)
}

func linkData(name string, value any) (linkOption, error) {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "data-")
	return newLinkAttribute("data-"+name, value)
}

func linkUnlessCurrent() linkOption {
	return linkOption{unlessCurrent: true}
}

func linkHref(href string) (string, error) {
	for _, r := range href {
		if r <= ' ' || r == 0x7f {
			return "", fmt.Errorf("lazyroutes: link_to href contains unsafe whitespace or control character")
		}
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("lazyroutes: link_to href %q is invalid: %w", href, err)
	}
	if parsed.Scheme != "" && !safeLinkScheme(parsed.Scheme) {
		return "", fmt.Errorf("lazyroutes: link_to href scheme %q is not allowed", parsed.Scheme)
	}
	return href, nil
}

func safeLinkScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "https", "mailto", "tel":
		return true
	default:
		return false
	}
}

func newLinkAttribute(name string, value any) (linkOption, error) {
	name = strings.TrimSpace(name)
	if err := validateLinkAttributeName(name); err != nil {
		return linkOption{}, err
	}
	if strings.EqualFold(name, "href") {
		return linkOption{}, fmt.Errorf("lazyroutes: link_to href is set by the destination")
	}
	if value == nil {
		return linkOption{}, nil
	}
	return linkOption{attrs: []linkAttribute{{
		name:  name,
		value: fmt.Sprint(value),
	}}}, nil
}

func collectLinkOptions(args []any) (linkOptions, error) {
	var options linkOptions
	for _, arg := range args {
		option, ok := arg.(linkOption)
		if !ok {
			return linkOptions{}, fmt.Errorf("lazyroutes: link_to option %T must come from attr, data, or unless_current", arg)
		}
		options.attrs = append(options.attrs, option.attrs...)
		options.unlessCurrent = options.unlessCurrent || option.unlessCurrent
	}
	return options, nil
}

func validateLinkAttributeName(name string) error {
	if name == "" {
		return fmt.Errorf("lazyroutes: link attribute name is required")
	}
	for _, r := range name {
		if ('a' <= r && r <= 'z') ||
			('A' <= r && r <= 'Z') ||
			('0' <= r && r <= '9') ||
			r == '-' ||
			r == '_' ||
			r == ':' ||
			r == '.' {
			continue
		}
		return fmt.Errorf("lazyroutes: link attribute name %q contains invalid character %q", name, r)
	}
	return nil
}

func linkDestinationIsCurrent(request *http.Request, href string) bool {
	if request == nil || request.URL == nil || href == "" {
		return false
	}
	destination, err := url.Parse(href)
	if err != nil {
		return false
	}
	if destination.Scheme != "" && request.URL.Scheme != "" && !strings.EqualFold(destination.Scheme, request.URL.Scheme) {
		return false
	}
	if destination.Host != "" && !strings.EqualFold(destination.Host, requestHost(request)) {
		return false
	}

	target := destination
	if !destination.IsAbs() && destination.Host == "" {
		target = request.URL.ResolveReference(destination)
	}

	if request.URL.EscapedPath() != target.EscapedPath() {
		return false
	}
	if destination.RawQuery == "" && !destination.ForceQuery {
		return true
	}
	return request.URL.RawQuery == target.RawQuery
}

func requestHost(request *http.Request) string {
	if request.URL != nil && request.URL.Host != "" {
		return request.URL.Host
	}
	return request.Host
}
