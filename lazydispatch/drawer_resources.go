package lazydispatch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazysupport"
)

// Scope
func (s *Scope) Resources(controller any, model ...any) *Resources {
	r := newResources(s.newChild(), controller)
	if len(model) > 0 {
		r.Model(model[0])
	}
	s.addrgen(r)
	return r
}

func newResources(parentScope *Scope, controller any) *Resources {
	r := &Resources{
		scope:      parentScope,
		Controller: controller,
	}
	validateResources(r)
	setResourcesDefaults(r)
	return r
}

// Name sets the controller name, and infers the path, plural, singular, and param name from it.
//
//	Resources(&PostsController{}).Name("articles")
//
//	// Is the same as:
//
//	r := Resources(&PostsController{})
//	r.Plural("articles")
//	r.Singular("article")
//	r.ParamName("article_id")
//	r.Path("articles")
func (r *Resources) Name(s string) *Resources {
	r.resourceName = s
	r.plural = lazysupport.Pluralize(r.resourceName)
	r.singular = lazysupport.Singularize(r.resourceName)
	r.paramName = lazysupport.Underscorize(r.singular) + "_id"
	r.scope.path = r.plural

	return r
}

// Singular sets the singular name of the resource and infers the param name from it.
func (r *Resources) Singular(s string) *Resources {
	r.singular = s
	r.paramName = lazysupport.Underscorize(r.singular) + "_id"
	return r
}

// Plural sets the plural name of the resource and infers the path from it.
func (r *Resources) Plural(s string) *Resources {
	r.plural = s
	r.scope.path = r.plural
	return r
}

// Pathnames sets the names of the new and edit paths
func (r *Resources) PathNames(new, edit string) *Resources {
	r.newPathName = new
	r.editPathName = edit
	return r
}

// Model sets the model associated with the controller
//
//	Resources(&PostsController{}).Model(&Post{})
//	PathFor(&Post{ID: 1}) // => "/posts/1"
//
// Look at []
func (r *Resources) Model(model any) *Resources {
	r.model = model
	return r
}
func (r *Resources) ParamName(paramName string) *Resources {
	r.paramName = paramName
	return r
}
func (r *Resources) Path(p string) *Resources {
	r.scope.path = p
	return r
}
func (r *Resources) As(a string) *Resources {
	r.plural = lazysupport.Pluralize(a)
	r.singular = lazysupport.Singularize(a)
	return r
}
func (r *Resources) Namespace(n string) *Resources {
	r.scope.namespace = n
	return r
}

// Record
type Resources struct {
	scope *Scope

	Controller any

	singular     string // "post" or empty to get that from the controller name
	plural       string // "psots" or empty to get that from the controller name
	resourceName string // "posts" or empty to get that from the controller name
	paramName    string // "post_id" or empty to get that from the controller name
	newPathName  string // "new" by default
	editPathName string // "edit" by d

	namespace string

	model any

	Scheme, Domain, Port string

	controllerFullName string
	controllerName     string
}

func (r *Resources) newScope() *Scope {
	s := r.scope.newChild()
	s.path = ":" + r.paramName
	s.as = r.singular
	s.namespace = r.namespace
	s.model = r.model
	return s
}

func (r *Resources) Draw(fn func(s *Scope)) {
	scope := r.newScope()
	fn(scope)
}

func (r *Resources) routes() []*Route {
	validateResources(r)
	routes := []*Route{}

	t := reflect.TypeOf(r.Controller)
	for i := 0; i < t.NumMethod(); i++ {
		route := r.routeForAction(t.Method(i).Name)
		if route == nil {
			continue
		}
		routes = append(routes, route.normalize())
	}
	return routes
}
func joinWithChar(c string, s ...string) string {
	out := []string{}
	for _, v := range s {
		if v == "" || v == c {
			continue
		}
		out = append(out, v)
	}
	return strings.Join(out, c)
}

func getMethodFromMethodName(name string) (method, rest string, ok bool) {
	if !methodPrefix.HasPrefix(strings.ToLower(name)) {
		return "", "", false
	}
	method, _ = methodPrefix.TrimPrefix(strings.ToLower(name))
	rest = strings.TrimPrefix(name[len(method):], "_")
	method = strings.ToUpper(method)

	if method == "PUT" {
		method = "PUT,PATCH"
	}
	return method, rest, true
}

func (r Resources) actionScope(name string) (actionScope *Scope, actionName string) {
	scope := r.scope.clone()
	scope.method = "GET"
	scope.namespace = r.namespace
	scope.as = r.plural

	switch {
	case name == "Index":
	case name == "Show":
		scope.model = r.model
		scope.as = r.singular
		if scope.path != "/" {
			scope.path += "/"
		}
		scope.path += ":" + r.paramName
	case name == "Create":
		scope.method = "POST"
	case name == "Update":
		scope.model = r.model
		if scope.path != "/" {
			scope.path += "/"
		}
		scope.path += ":" + r.paramName
		scope.as = r.singular
		scope.method = "PUT,PATCH"
	case name == "Delete":
		if scope.path != "/" {
			scope.path += "/"
		}
		scope.path += ":" + r.paramName
		scope.as = r.singular
		scope.model = r.model
		scope.method = "DELETE"
	case name == "New":
		if scope.path != "/" {
			scope.path += "/"
		}
		scope.path += r.newPathName
		scope.as = r.singular
		scope.verb = "new"
	case name == "Edit":
		if scope.path != "/" {
			scope.path += "/"
		}
		scope.path += ":" + r.paramName + "/" + r.editPathName
		scope.verb = "edit"
		scope.as = r.singular
	default:
		if strings.HasPrefix(name, "Member") {
			scope.as = r.singular
			if scope.path != "/" {
				scope.path += "/"
			}
			scope.path += ":" + r.paramName
			name = strings.TrimPrefix(name[len("Member"):], "_")
		}
		method, rest, ok := getMethodFromMethodName(name)
		if !ok {
			return nil, ""
		}
		name = lazysupport.Underscorize(rest)
		scope.method = method
		scope.verb = name
		if rest != "" {
			if scope.path != "/" {
				scope.path += "/"
			}
			scope.path += name
		}
	}

	return scope, lazysupport.Underscorize(name)

}

func (r Resources) routeForAction(name string) *Route {
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
		Models:     models,
		Action:     actionName,
		Controller: r.resourceName,
		Target:     fmt.Sprintf("%s#%s", r.controllerFullName, originalName),
	}
	route.Handler = forAction(r.Controller, originalName, func(ctx context.Context, req *http.Request) context.Context {
		c := context.WithValue(ctx, reflect.TypeOf(route), route)
		return c
	})
	return route
}

var methodPrefix = lazysupport.NewStringSet("get", "post", "delete", "patch", "put")

func validateResources(r *Resources) {
	errs := []error{}
	if r.Controller == nil {
		panic(fmt.Errorf("controller is nil"))
	} else {
		t := reflect.TypeOf(r.Controller)
		if t.Kind() != reflect.Ptr ||
			t.Elem().Kind() != reflect.Struct {
			panic(fmt.Errorf("controller must be a pointer to a struct"))
		}
	}
	if len(errs) > 0 {
		panic(errors.Join(errs...))
	}
}

func setResourcesDefaults(r *Resources) {
	r.controllerFullName = lazysupport.NameOf(r.Controller)
	r.controllerName = getControllerName(r.Controller)

	r.resourceName = lazysupport.Underscorize(r.controllerName)
	r.plural = lazysupport.Pluralize(r.resourceName)
	r.singular = lazysupport.Singularize(r.resourceName)
	r.paramName = lazysupport.Underscorize(r.singular) + "_id"

	r.newPathName = "new"
	r.editPathName = "edit"

	r.scope.path = r.plural
}

func getControllerName(obj any) string {
	name := lazysupport.NameOf(obj)
	if name == "Controller" {
		name = lazysupport.NameOfWithPackage(obj)
		name = name[strings.Index(name, ".")+1:]
		return lazysupport.Camelize(name)
	}

	if strings.HasSuffix(name, "Controller") {
		return lazysupport.Camelize(name[:len(name)-10])
	}
	return name
}
