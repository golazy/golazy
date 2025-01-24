package lazydispatch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"golazy.dev/lazysupport"
)

func (s *Scope) Resource(controller any) *Resource {
	r := newResource(s, controller)
	s.addrgen(r)
	return r
}

func newResource(s *Scope, controller any) *Resource {
	r := &Resource{
		parentScope: s,
		Controller:  controller,
	}
	validateResource(r)
	setResourceDefaults(r)
	return r
}

type Resource struct {
	parentScope *Scope
	Controller  any

	namespace string // "admin" or "admin/v1"

	path string

	name string // "session" or empty to get that from the controller name

	newPathName  string // "new" by default
	editPathName string // "edit" by d

	controllerFullName string
	controllerName     string
}

func (r *Resource) Name(name string) *Resource {
	r.name = name
	r.path = name
	return r
}

func (r *Resource) Path(path string) *Resource {
	r.path = path
	return r
}
func (r *Resource) actionScope(name string) (*Scope, string) {
	scope := r.parentScope.newChild()
	scope.method = "GET"
	scope.path = r.path
	scope.namespace = r.namespace
	scope.as = r.name

	switch {
	case name == "Show":
	case name == "Create":
		scope.method = "POST"
	case name == "Update":
		scope.method = "PUT,PATCH"
	case name == "Delete":
		scope.method = "DELETE"
	case name == "New":
		scope.path += "/" + r.newPathName
		scope.verb = "new"
	case name == "Edit":
		scope.path += "/" + r.editPathName
		scope.verb = "edit"
	default:
		method, rest, ok := getMethodFromMethodName(name)
		if !ok {
			return nil, ""
		}
		name = lazysupport.Underscorize(rest)
		scope.method = method
		scope.as = r.name + "_" + name
		scope.path += "/" + name
	}

	return scope, lazysupport.Underscorize(name)

}

func (r *Resource) routeForAction(name string) *Route {
	originalName := name

	scope, actionName := r.actionScope(name)
	if scope == nil {
		return nil
	}
	method, path, name, _, models := scope.routeInfo()

	route := &Route{
		Method:     method,
		URL:        path,
		Name:       name,
		Action:     actionName,
		Controller: r.name,
		Models:     models,
		Target:     fmt.Sprintf("%s#%s", r.controllerFullName, originalName),
	}
	route.Handler = forAction(r.Controller, originalName, func(ctx context.Context, req *http.Request) context.Context {
		c := context.WithValue(ctx, reflect.TypeOf(route), route)
		return c
	})
	return route

}

func (r *Resource) Draw(fn func(s *Scope)) *Scope {
	scope := r.newScope()
	fn(scope)
	return scope
}
func (r *Resource) newScope() *Scope {
	scope := r.parentScope.newChild()
	scope.path = r.path
	scope.as = r.name
	scope.namespace = r.namespace
	return scope
}
func (r *Resource) routes() []*Route {
	validateResource(r)

	routes := []*Route{}
	t := reflect.TypeOf(r.Controller)
	for i := 0; i < t.NumMethod(); i++ {
		route := r.routeForAction(t.Method(i).Name)
		if route == nil {
			continue
		}
		routes = append(routes, route)
	}
	return routes
}

//func (r Resource) urlForMethod(name string) string {
//	switch name {
//	case "New":
//		return path.Join(r.path, r.newPathName)
//	case "Edit":
//		return path.Join(r.path, r.editPathName)
//	case "Create", "Update", "Delete":
//		return r.path
//	default:
//		_, name = method_prefix.TrimPrefix(name)
//		return path.Join(r.path, lazysupport.Underscorize(name))
//	}
//}

func validateResource(rsrcs *Resource) {
	errs := []error{}
	if rsrcs.Controller == nil {
		errs = append(errs, fmt.Errorf("controller is nil"))
	} else {
		t := reflect.TypeOf(rsrcs.Controller)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			errs = append(errs, fmt.Errorf("controller must be a pointer to a struct"))
		}
	}
	if len(errs) > 0 {
		panic(errors.Join(errs...))
	}

}

func setResourceDefaults(r *Resource) {
	r.controllerFullName = lazysupport.NameOf(r.Controller)
	r.controllerName = getControllerName(r.Controller)

	r.name = lazysupport.Underscorize(r.controllerName)

	r.newPathName = "new"
	r.editPathName = "edit"
	r.path = r.name
}
