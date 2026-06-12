package lazyroutes

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

type Resource[T any] struct {
	ctx     context.Context
	mux     *http.ServeMux
	factory Factory[T]

	path     string
	singular string
	plural   string
	param    string

	paramSet bool

	custom []resourceRoute[T]
}

type resourceRoute[T any] struct {
	method string
	path   string
	member bool
	action func(*T, http.ResponseWriter, *http.Request) error
}

func Resources[T any](
	ctx context.Context,
	mux *http.ServeMux,
	factory Factory[T],
	configure ...func(*Resource[T]),
) *Resource[T] {
	resource := newResource(ctx, mux, factory)
	for _, fn := range configure {
		fn(resource)
	}
	resource.draw()
	return resource
}

func (r *Resource[T]) Path(path string) *Resource[T] {
	path = strings.Trim(path, "/")
	r.path = "/" + path
	return r
}

func (r *Resource[T]) Singular(name string) *Resource[T] {
	r.singular = strings.Trim(name, "/")
	if !r.paramSet {
		r.param = r.singular + "_id"
	}
	return r
}

func (r *Resource[T]) Plural(name string) *Resource[T] {
	r.plural = strings.Trim(name, "/")
	r.path = "/" + r.plural
	return r
}

func (r *Resource[T]) Param(name string) *Resource[T] {
	r.param = strings.Trim(name, "{}")
	r.paramSet = true
	return r
}

func (r *Resource[T]) Get(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodGet, path, false, action)
}

func (r *Resource[T]) Post(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodPost, path, false, action)
}

func (r *Resource[T]) Put(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodPut, path, false, action)
}

func (r *Resource[T]) Patch(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodPatch, path, false, action)
}

func (r *Resource[T]) Delete(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodDelete, path, false, action)
}

func (r *Resource[T]) MemberGet(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodGet, path, true, action)
}

func (r *Resource[T]) MemberPost(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodPost, path, true, action)
}

func (r *Resource[T]) MemberPut(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodPut, path, true, action)
}

func (r *Resource[T]) MemberPatch(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodPatch, path, true, action)
}

func (r *Resource[T]) MemberDelete(path string, action func(*T, http.ResponseWriter, *http.Request) error) {
	r.add(http.MethodDelete, path, true, action)
}

func newResource[T any](ctx context.Context, mux *http.ServeMux, factory Factory[T]) *Resource[T] {
	name := controllerName[T]()
	singular := singularize(name)
	plural := pluralize(singular)
	return &Resource[T]{
		ctx:      ctx,
		mux:      mux,
		factory:  factory,
		path:     "/" + plural,
		singular: singular,
		plural:   plural,
		param:    singular + "_id",
	}
}

func (r *Resource[T]) add(
	method string,
	path string,
	member bool,
	action func(*T, http.ResponseWriter, *http.Request) error,
) {
	r.custom = append(r.custom, resourceRoute[T]{
		method: method,
		path:   strings.Trim(path, "/"),
		member: member,
		action: action,
	})
}

func (r *Resource[T]) draw() {
	r.registerAction(http.MethodGet, r.path, "Index")
	r.registerAction(http.MethodGet, r.path+"/new", "New")
	r.registerAction(http.MethodPost, r.path, "Create")
	r.registerAction(http.MethodGet, r.memberPath(), "Show")
	r.registerAction(http.MethodGet, r.memberPath()+"/edit", "Edit")
	r.registerAction(http.MethodPatch, r.memberPath(), "Update")
	r.registerAction(http.MethodPut, r.memberPath(), "Update")
	r.registerAction(http.MethodDelete, r.memberPath(), "Delete")

	for _, route := range r.custom {
		path := r.path + "/" + route.path
		if route.member {
			path = r.memberPath() + "/" + route.path
		}
		r.mux.Handle(route.method+" "+path, Bind(r.ctx, r.factory, route.action))
	}
}

func (r *Resource[T]) memberPath() string {
	return r.path + "/{" + r.param + "}"
}

func (r *Resource[T]) registerAction(method string, path string, actionName string) {
	if action, ok := methodAction[T](actionName); ok {
		r.mux.Handle(method+" "+path, Bind(r.ctx, r.factory, action))
	}
}

func methodAction[T any](name string) (func(*T, http.ResponseWriter, *http.Request) error, bool) {
	controllerType := reflect.TypeOf((*T)(nil))
	method, ok := controllerType.MethodByName(name)
	if !ok {
		return nil, false
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	writerType := reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
	requestType := reflect.TypeOf((*http.Request)(nil))
	if method.Type.NumIn() != 3 ||
		!method.Type.In(1).Implements(writerType) ||
		method.Type.In(2) != requestType ||
		method.Type.NumOut() != 1 ||
		!method.Type.Out(0).Implements(errorType) {
		panic(fmt.Errorf("lazyroutes: %s must have signature func(http.ResponseWriter, *http.Request) error", name))
	}

	return func(controller *T, w http.ResponseWriter, r *http.Request) error {
		values := method.Func.Call([]reflect.Value{
			reflect.ValueOf(controller),
			reflect.ValueOf(w),
			reflect.ValueOf(r),
		})
		if values[0].IsNil() {
			return nil
		}
		return values[0].Interface().(error)
	}, true
}

func controllerName[T any]() string {
	t := reflect.TypeOf((*T)(nil)).Elem()
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

func pluralize(singular string) string {
	if strings.HasSuffix(singular, "y") {
		return strings.TrimSuffix(singular, "y") + "ies"
	}
	if strings.HasSuffix(singular, "s") {
		return singular
	}
	return singular + "s"
}

func singularize(plural string) string {
	if strings.HasSuffix(plural, "ies") {
		return strings.TrimSuffix(plural, "ies") + "y"
	}
	return strings.TrimSuffix(plural, "s")
}
