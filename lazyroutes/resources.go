package lazyroutes

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"

	"golazy.dev/lazysupport/inflection"
)

type Resource struct {
	scope      *Scope
	controller controllerConstructor

	path     string
	singular string
	plural   string
	param    string

	paramSet bool

	models []reflect.Type
	custom []resourceRoute
	parent *Resource
}

type resourceRoute struct {
	method string
	path   string
	member bool
	action any
}

func (s *Scope) Resources(controller any, configure ...func(*Resource)) *Resource {
	resource := newResource(s, controller)
	for _, fn := range configure {
		fn(resource)
	}
	resource.draw()
	return resource
}

func (r *Resource) Resources(controller any, configure ...func(*Resource)) *Resource {
	resource := newResource(r.scope, controller)
	resource.parent = r
	for _, fn := range configure {
		fn(resource)
	}
	resource.draw()
	return resource
}

func (r *Resource) Path(path string) *Resource {
	path = strings.Trim(path, "/")
	r.path = "/" + path
	return r
}

func (r *Resource) Singular(name string) *Resource {
	r.singular = strings.Trim(name, "/")
	if !r.paramSet {
		r.param = r.singular + "_id"
	}
	return r
}

func (r *Resource) Plural(name string) *Resource {
	r.plural = strings.Trim(name, "/")
	r.path = "/" + r.plural
	return r
}

func (r *Resource) Param(name string) *Resource {
	r.param = strings.Trim(name, "{}")
	r.paramSet = true
	return r
}

func (r *Resource) Model(models ...any) *Resource {
	for _, model := range models {
		t := modelType(model)
		if t == nil {
			panic(fmt.Errorf("lazyroutes: resource model must be a struct or pointer to struct"))
		}
		r.models = append(r.models, t)
	}
	return r
}

func (r *Resource) Get(path string, action any) {
	r.add(http.MethodGet, path, false, action)
}

func (r *Resource) Post(path string, action any) {
	r.add(http.MethodPost, path, false, action)
}

func (r *Resource) Put(path string, action any) {
	r.add(http.MethodPut, path, false, action)
}

func (r *Resource) Patch(path string, action any) {
	r.add(http.MethodPatch, path, false, action)
}

func (r *Resource) Delete(path string, action any) {
	r.add(http.MethodDelete, path, false, action)
}

func (r *Resource) MemberGet(path string, action any) {
	r.add(http.MethodGet, path, true, action)
}

func (r *Resource) MemberPost(path string, action any) {
	r.add(http.MethodPost, path, true, action)
}

func (r *Resource) MemberPut(path string, action any) {
	r.add(http.MethodPut, path, true, action)
}

func (r *Resource) MemberPatch(path string, action any) {
	r.add(http.MethodPatch, path, true, action)
}

func (r *Resource) MemberDelete(path string, action any) {
	r.add(http.MethodDelete, path, true, action)
}

func newResource(scope *Scope, controller any) *Resource {
	controllerConstructor := newControllerConstructor(controller)
	name := controllerNameFromType(controllerConstructor.controllerType)
	singular := inflection.Singularize(name)
	plural := inflection.Pluralize(singular)
	return &Resource{
		scope:      scope,
		controller: controllerConstructor,
		path:       "/" + plural,
		singular:   singular,
		plural:     plural,
		param:      singular + "_id",
	}
}

func (r *Resource) add(method string, path string, member bool, action any) {
	r.custom = append(r.custom, resourceRoute{
		method: method,
		path:   strings.Trim(path, "/"),
		member: member,
		action: action,
	})
}

func (r *Resource) draw() {
	r.registerAction(http.MethodGet, r.basePath(), "Index")
	r.registerAction(http.MethodGet, r.basePath()+"/new", "New")
	r.registerAction(http.MethodPost, r.basePath(), "Create")
	r.registerAction(http.MethodGet, r.memberPath(), "Show")
	r.registerAction(http.MethodGet, r.memberPath()+"/edit", "Edit")
	r.registerAction(http.MethodPatch, r.memberPath(), "Update")
	r.registerAction(http.MethodPut, r.memberPath(), "Update")
	r.registerAction(http.MethodDelete, r.memberPath(), "Delete")

	for _, route := range r.custom {
		path := r.basePath() + "/" + route.path
		if route.member {
			path = r.memberPath() + "/" + route.path
		}
		routeEntry := r.routeMetadata(route.path, path, route.member)
		routeEntry.Method = route.method
		routeEntry.Path = path
		routeEntry.Action = actionName(route.action)
		r.scope.register(route.method, path, routeEntry, r.controller.bind(r.scope.Context, r.scope.scopedPath(path), actionValue(route.action)))
	}
	for _, model := range r.models {
		r.scope.rootScope().models[model] = ModelRoutes{
			Create: r.scope.scopedName(r.namedRouteName("Create", r.basePath(), false)),
			Update: r.scope.scopedName(r.namedRouteName("Update", r.memberPath(), true)),
			Delete: r.scope.scopedName(r.namedRouteName("Delete", r.memberPath(), true)),
		}
	}
}

func (r *Resource) basePath() string {
	if r.parent == nil {
		return r.path
	}
	return r.parent.memberPath() + r.path
}

func (r *Resource) memberPath() string {
	return r.basePath() + "/{" + r.param + "}"
}

func (r *Resource) registerAction(method string, path string, actionName string) {
	if action, ok := methodAction(r.controller.controllerType, actionName); ok {
		route := r.routeMetadata(actionName, path, false)
		r.scope.register(method, path, route, r.controller.bind(r.scope.Context, r.scope.scopedPath(path), action))
	}
}

func (r *Resource) routeMetadata(actionName string, path string, member bool) Route {
	metadata := Route{
		Method:     "",
		Path:       path,
		Action:     actionName,
		Controller: r.plural,
	}
	metadata.Name = r.namedRouteName(actionName, path, member)
	return metadata
}

func (r *Resource) namedRouteName(actionName string, path string, member bool) string {
	switch actionName {
	case "Index":
		return r.withParentName(r.plural)
	case "New":
		return r.withParentName("new_" + r.singular)
	case "Create":
		return r.withParentName(r.plural)
	case "Show":
		return r.withParentName(r.singular)
	case "Edit":
		return r.withParentName("edit_" + r.singular)
	case "Update":
		return r.withParentName(r.singular)
	case "Delete":
		return r.withParentName(r.singular)
	}

	if member {
		return r.withParentName(pathToName(path) + "_" + r.singular)
	}
	return r.withParentName(pathToName(path) + "_" + r.plural)
}

func (r *Resource) withParentName(name string) string {
	if r.parent == nil {
		return name
	}
	return name + "_" + r.parent.singular
}

func pathToName(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segments) - 1; i >= 0; i-- {
		if strings.TrimSpace(segments[i]) == "" {
			continue
		}
		if strings.HasPrefix(segments[i], "{") && strings.HasSuffix(segments[i], "}") {
			continue
		}
		return segments[i]
	}
	return "route"
}

func methodAction(controllerType reflect.Type, name string) (reflect.Value, bool) {
	method, ok := controllerType.MethodByName(name)
	if !ok {
		return reflect.Value{}, false
	}
	return method.Func, true
}

func controllerNameFromType(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := t.Name()
	name = strings.TrimSuffix(name, "Controller")
	return toRouteWord(name)
}

func toRouteWord(name string) string {
	var out strings.Builder
	for index, r := range name {
		if unicode.IsUpper(r) {
			if index > 0 {
				out.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		out.WriteRune(r)
	}
	return out.String()
}

func modelType(model any) reflect.Type {
	if model == nil {
		return nil
	}
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	return t
}
